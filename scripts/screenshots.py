#!/usr/bin/env python3
"""Оригинальные скриншоты работающего GoBpmBank через Chrome DevTools, без сторонних пакетов."""
import base64
import json
import os
from pathlib import Path
import socket
import struct
import time
import uuid
import urllib.request

ROOT=Path(__file__).resolve().parents[1]
BASE=os.environ.get('BANK_URL','http://127.0.0.1:8090')
DEBUG=os.environ.get('CHROME_DEBUG','http://127.0.0.1:9229')
OUT=ROOT/'docs/images'
OUT.mkdir(parents=True,exist_ok=True)

def get(url):
    with urllib.request.urlopen(url,timeout=10) as r:return json.load(r)

def cdp(method,params=None):
    target=next(t for t in get(DEBUG+'/json/list') if t['type']=='page')
    from urllib.parse import urlsplit
    u=urlsplit(target['webSocketDebuggerUrl'])
    s=socket.create_connection((u.hostname,u.port),timeout=20)
    s.sendall((f'GET {u.path} HTTP/1.1\r\nHost: localhost\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n').encode())
    h=b''
    while not h.endswith(b'\r\n\r\n'):h+=s.recv(1)
    assert b'101' in h,h
    payload=json.dumps({'id':1,'method':method,'params':params or {}}).encode()
    mask=os.urandom(4)
    head=bytes([0x81,0x80|len(payload)]) if len(payload)<126 else b'\x81\xfe'+struct.pack('!H',len(payload))
    s.sendall(head+mask+bytes(v^mask[i%4] for i,v in enumerate(payload)))
    def read(n):
        b=b''
        while len(b)<n:
            chunk=s.recv(n-len(b))
            if not chunk:raise RuntimeError('WebSocket закрыт')
            b+=chunk
        return b
    while True:
        h=read(2);n=h[1]&127
        if n==126:n=struct.unpack('!H',read(2))[0]
        elif n==127:n=struct.unpack('!Q',read(8))[0]
        msg=json.loads(read(n))
        if msg.get('id')==1:
            s.close()
            if 'error' in msg:raise RuntimeError(msg['error'])
            return msg['result']

def evaluate(expression):
    r=cdp('Runtime.evaluate',{'expression':expression,'returnByValue':True})
    if 'exceptionDetails' in r:raise RuntimeError(r)
    return r['result'].get('value')

def shot(name,path,ready='document.readyState === "complete"'):
    cdp('Page.navigate',{'url':BASE+path})
    for _ in range(100):
        if evaluate(ready):break
        time.sleep(.1)
    else:raise RuntimeError('Страница не готова: '+path)
    time.sleep(.3)
    error=evaluate('document.querySelector("#error:not([hidden])")?.textContent || ""')
    assert not error,error
    save(name)

def save(name):
    image=cdp('Page.captureScreenshot',{'format':'png','captureBeyondViewport':False})
    (OUT/name).write_bytes(base64.b64decode(image['data']))
    print('Скриншот:',name,flush=True)

def child(app,process):
    instances=get(BASE+'/api/v1/instances?businessKey='+app)['instances']
    return next(i['instance_uid'] for i in instances if i['process_name']==process)

def history(name,uid):shot(name,'/ui/history/?instanceId='+uid,'document.querySelectorAll("#canvas .djs-element").length > 3')

if __name__=='__main__':
    results=json.loads((ROOT/'.run/e2e-results.json').read_text())
    cases={r['scenario']:r for r in results}
    cdp('Emulation.setDeviceMetricsOverride',{'width':1600,'height':1000,'deviceScaleFactor':1,'mobile':False})
    cdp('Emulation.setDeviceMetricsOverride',{'width':1600,'height':1200,'deviceScaleFactor':1,'mobile':False})
    shot('01-home.png','/')
    cdp('Emulation.setDeviceMetricsOverride',{'width':1600,'height':1000,'deviceScaleFactor':1,'mobile':False})
    shot('02-processes.png','/ui/','document.querySelectorAll("#processes-body tr").length === 8')
    history('03-main-process.png',cases['happy']['instanceId'])
    diligence=child(cases['incomplete']['applicationId'],'DueDiligence')
    history('04-due-diligence.png',diligence)
    history('05-parallel-checks.png',child(cases['happy']['applicationId'],'AutomatedChecks'))
    history('06-core-opening.png',child(cases['core_pending']['applicationId'],'OpenCoreAccount'))
    # Настоящее нажатие на элемент открывает историю вызовов DMN и таблицу правил.
    history('07-dmn-context.png',diligence)
    pos=evaluate('(()=>{const r=document.querySelector("[data-element-id=RouteApplication] .djs-visual").getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2}})()')
    for kind in ['mousePressed','mouseReleased']:
        cdp('Input.dispatchMouseEvent',{'type':kind,'x':pos['x'],'y':pos['y'],'button':'left','clickCount':1})
    time.sleep(.4)
    # Название кнопки берётся из самой консоли, без подмены интерфейса.
    evaluate('(()=>{const b=[...document.querySelectorAll("button")].find(b=>!b.hidden && /правил|DMN/i.test(b.textContent));if(b)b.click()})()')
    time.sleep(.8)
    save('07-dmn-rules.png')
    history('08-compliance.png',child(cases['pep']['applicationId'],'ComplianceReview'))
    # Реальный запуск из консоли и заполнение задачи стандартным редактором переменных.
    app=str(uuid.uuid4())
    shot('09-start.png','/ui/#/start','document.querySelector("#start-scenario") !== null')
    evaluate('document.querySelector("#start-key").value='+json.dumps(app))
    evaluate('document.querySelector("#start-form").requestSubmit()')
    for _ in range(100):
        if evaluate('!document.querySelector("#start-result").hidden'):break
        time.sleep(.1)
    else:raise RuntimeError('Запуск из консоли не прошёл')
    shot('10-user-tasks.png','/ui/#/tasks?user=operator&groups=operators',
         'document.querySelector("#tasks-open-count").textContent !== "0"')
    evaluate('(()=>{const c=[...document.querySelectorAll("li.card")].find(c=>c.textContent.includes('+json.dumps(app)+'));if(!c)throw Error("Задача не найдена");[...c.querySelectorAll("button")].find(b=>b.textContent==="Взять").click()})()')
    for _ in range(100):
        if evaluate('[...document.querySelectorAll("li.card")].some(c=>c.textContent.includes('+json.dumps(app)+') && c.textContent.includes("Выполнить"))'):break
        time.sleep(.1)
    else:raise RuntimeError('Задача не взята')
    evaluate('(()=>{const c=[...document.querySelectorAll("li.card")].find(c=>c.textContent.includes('+json.dumps(app)+'));[...c.querySelectorAll("button")].find(b=>b.textContent==="Выполнить").click()})()')
    for _ in range(100):
        if evaluate('document.querySelector("#complete").open'):break
        time.sleep(.1)
    else:raise RuntimeError('Редактор переменных не открыт')
    evaluate('document.querySelector("#complete-editor .vars-tools button:last-child").click()')
    sample={'applicantName':'Учебный Клиент','birthDate':'1990-01-15','citizenship':'RU','passportRef':'DEMO-DOCUMENT','address':'Учебный адрес','purpose':'Личные нужды','taxResidency':'RU','identityVerified':True,'documentsComplete':True}
    evaluate('document.querySelector("#complete-editor textarea").value='+json.dumps(json.dumps(sample,ensure_ascii=False,indent=2)))
    save('11-task-variables.png')
    history('12-intake.png',child(cases['happy']['applicationId'],'CollectApplication'))
    history('13-contract.png',child(cases['happy']['applicationId'],'ConcludeContract'))
    history('14-post-opening.png',child(cases['happy']['applicationId'],'PostOpening'))
    # Сведения для воспроизведения; скриншоты не дорисовываются.
    (ROOT/'docs/screenshot-manifest.json').write_text(json.dumps({'viewport':[1600,1000],'homeViewport':[1600,1200],'bankURL':BASE,'cases':cases,'taskApplication':app},ensure_ascii=False,indent=2)+'\n')
