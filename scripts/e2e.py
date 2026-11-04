#!/usr/bin/env python3
"""Сквозные проверки запущенного учебного стенда. Только вымышленные данные, без очистки БД."""
import json
import os
import pathlib
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

BANK = os.environ.get('BANK_URL', 'http://127.0.0.1:8090')
MOCK = os.environ.get('MOCK_URL', 'http://127.0.0.1:8091')
OUT = pathlib.Path(__file__).resolve().parents[1] / '.run'
OUT.mkdir(exist_ok=True)

def request(base, path, data=None, headers=None, expected=None):
    req = urllib.request.Request(base + path, data=json.dumps(data).encode() if data is not None else None,
                                 headers={'Content-Type': 'application/json', **(headers or {})})
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            status, body = resp.status, resp.read()
    except urllib.error.HTTPError as exc:
        status, body = exc.code, exc.read()
    if expected is None:
        assert 200 <= status < 300, (path, status, body.decode())
    else:
        assert status in expected, (path, status, body.decode())
    try: return json.loads(body) if body else None
    except json.JSONDecodeError: return body.decode()

def bank(path, data=None, **kw): return request(BANK, path, data, **kw)

def run(scenario, expected_outcome, expect_open):
    app_id = str(uuid.uuid4())
    command = {'applicationId': app_id, 'scenario': scenario}
    root = bank('/bank/api/applications', command)['id']
    assert bank('/bank/api/applications', command)['id'] == root, 'дубликат заявления'
    bank('/bank/api/applications', {**command, 'scenario': 'happy' if scenario != 'happy' else 'pep'}, expected=[409])
    seen = set()
    completed = []
    expires = time.monotonic() + 100
    while time.monotonic() < expires:
        instance = bank('/api/v1/instances/' + root)
        assert instance['status'] != 'failed', (scenario, instance)
        if instance['status'] == 'completed': break
        tasks = bank('/api/v1/tasks?businessKey=' + urllib.parse.quote(app_id))['tasks']
        for task in tasks:
            uid, element = task['task_uid'], task['element_id']
            if uid in seen: continue
            seen.add(uid)
            if element == 'IdentifyApplicant':
                user = 'operator'
                variables = {'applicantName': 'Учебный Клиент', 'birthDate': '1990-01-15', 'citizenship': 'RU',
                             'passportRef': 'DEMO-DOCUMENT', 'address': 'Учебный адрес', 'purpose': 'Личные нужды',
                             'taxResidency': '' if scenario == 'tax_refusal' else 'RU', 'identityVerified': True,
                             'documentsComplete': scenario != 'incomplete' or 'IdentifyApplicant' in completed}
                # Нельзя подменить роль запросом claim или добавить служебную переменную.
                bank(f'/api/v1/tasks/{uid}/claim', {'userId': 'aml', 'groups': ['operators']}, expected=[403])
            elif element == 'ReviewClient':
                user = 'aml'
                variables = {'reviewDecision': 'DECLINE' if scenario == 'tax_refusal' else 'CONTINUE',
                             'reason': 'Учебное непредставление налоговых сведений' if scenario == 'tax_refusal' else 'Учебное совпадение разрешено',
                             'legalBasisCode': 'NK142_4_NO_TAX_DATA' if scenario == 'tax_refusal' else ''}
            elif element == 'AuthorizeClient':
                user, variables = 'chief', {'reviewDecision': 'CONTINUE', 'authorized': True, 'reason': 'Второй сотрудник проверил учебные сведения'}
            elif element == 'ConfirmContract':
                user, variables = 'operator', {'contractAccepted': scenario != 'withdrawn', 'signedDocumentRef': 'DEMO-SIGNATURE-REFERENCE'}
            else: raise AssertionError(('неизвестная задача', task))
            bank(f'/api/v1/tasks/{uid}/claim', {'userId': user})
            bank(f'/api/v1/tasks/{uid}/complete', {'userId': user, 'variables': {'accountOpened': True}}, expected=[403])
            bank(f'/api/v1/tasks/{uid}/complete', {'userId': user, 'variables': variables})
            completed.append(element)
        time.sleep(.15)
    else: raise AssertionError(('таймаут', scenario, root))
    records = bank(f'/api/v1/instances/{root}/history?withChildren=true')['records']
    snapshots = [r['variables'] for r in sorted(records, key=lambda r:r['id']) if r.get('variables')]
    assert snapshots, scenario
    latest = snapshots[-1]
    assert latest['outcome'] == expected_outcome, (scenario, latest)
    assert latest.get('accountOpened', False) == expect_open, (scenario, latest)
    state = request(MOCK, '/v1/state')
    openings = [v for v in state['openings'] if v['request_id'] == app_id + ':opening:v1']
    assert len(openings) == (1 if expect_open or scenario == 'core_reject' else 0), (scenario, openings)
    if expect_open:
        assert openings[0]['status'] == 'OPENED' and openings[0]['registered_at']
        assert latest['taxReportStatus'] == 'CONFIRMED'
        for kind in ['archive', 'tax-report', 'notification']:
            effects = [x for x in state['effects'] if x['key'] == app_id + ':' + kind + ':v1' and x['kind'] == kind]
            assert len(effects) == 1, (scenario, kind, effects)
    if scenario == 'incomplete': assert completed.count('IdentifyApplicant') == 2
    if scenario in ('happy', 'incomplete'): assert 'ReviewClient' not in completed
    if scenario == 'screening_match': assert 'ReviewClient' in completed
    if scenario == 'pep': assert 'AuthorizeClient' in completed
    if scenario == 'core_pending': assert openings[0]['polls'] >= 2
    if scenario == 'tax_report_retry':
        tax = next(x for x in state['effects'] if x['key'] == app_id + ':tax-report:v1')
        assert tax['hits'] >= 2
    if scenario == 'lost_response':
        core_command = {'applicationId': app_id, 'scenario': scenario, 'customerId': latest['customerId'], 'contractId': latest['contractId']}
        headers = {'Idempotency-Key': app_id + ':opening:v1'}
        repeated = request(MOCK, '/v1/account-openings', core_command, headers)
        assert repeated['accountId'] == latest['accountId']
        request(MOCK, '/v1/account-openings', {**core_command, 'contractId': 'CHANGED'}, headers, expected=[409])
    result = {'scenario': scenario, 'applicationId': app_id, 'instanceId': root, 'outcome': latest['outcome'],
              'accountOpened': latest.get('accountOpened', False), 'historyRecords': len(records), 'tasks': completed}
    print('PASS', scenario, expected_outcome, 'history=', len(records), flush=True)
    return result

if __name__ == '__main__':
    processes = bank('/api/v1/processes')['processes']
    assert len(processes) == 8, processes
    assert all(not p['id'].startswith('Test') for p in processes)
    bank('/api/v1/processes/CollectApplication/instances', {}, expected=[403])
    scenarios = [('happy','OPENED',True), ('incomplete','OPENED',True), ('screening_match','OPENED',True),
                 ('pep','OPENED',True), ('tax_refusal','DECLINED_WITH_BASIS',False), ('withdrawn','WITHDRAWN',False),
                 ('lost_response','OPENED',True), ('core_pending','OPENED',True), ('core_reject','REFERRED_CORE',False),
                 ('tax_report_retry','OPENED',True)]
    results = [run(*s) for s in scenarios]
    (OUT/'e2e-results.json').write_text(json.dumps(results, ensure_ascii=False, indent=2)+'\n')
    print('Все 10 сценариев пройдены; результаты: .run/e2e-results.json')
