#!/usr/bin/env python3
"""Воспроизводимые BPMN/DMN с диаграммами. Логика исполнения генерируется go-bpm-gen."""
from pathlib import Path
import xml.etree.ElementTree as E

ROOT = Path(__file__).resolve().parents[1]
NS = {'bpmn':'http://www.omg.org/spec/BPMN/20100524/MODEL',
      'gobpm':'https://github.com/nicolas-vp/go-bpm/schema/1.0',
      'bpmndi':'http://www.omg.org/spec/BPMN/20100524/DI',
      'dc':'http://www.omg.org/spec/DD/20100524/DC',
      'di':'http://www.omg.org/spec/DD/20100524/DI',
      'xsi':'http://www.w3.org/2001/XMLSchema-instance'}
for k,v in NS.items(): E.register_namespace(k,v)
def sub(parent, tag, **attrs):
    prefix, name = tag.split(':')
    return E.SubElement(parent, '{'+NS[prefix]+'}'+name, attrs)

def process(pid, name, nodes, edges):
    root=E.Element('{'+NS['bpmn']+'}definitions', {'id':'Definitions_'+pid,'targetNamespace':'https://github.com/nicolas-vp/go-bpm-bank'})
    p=sub(root,'bpmn:process',id=pid,name=name,isExecutable='true')
    plane=sub(sub(root,'bpmndi:BPMNDiagram',id='Diagram_'+pid),'bpmndi:BPMNPlane',id='Plane_'+pid,bpmnElement=pid)
    geometry={}; elements={}
    for kind,nid,label,x,y,opt in nodes:
        attrs={'id':nid,'name':label}
        if kind=='boundaryEvent': attrs.update(attachedToRef=opt['attached'],cancelActivity='true')
        n=sub(p,'bpmn:'+kind,**attrs); elements[nid]=n
        if kind=='callActivity':
            sub(sub(n,'bpmn:extensionElements'),'gobpm:calledElement',processId=opt['process'],propagateAllChildVariables='true')
        if kind=='serviceTask':
            sub(sub(n,'bpmn:extensionElements'),'gobpm:taskDefinition',type=opt['job'],retries='6')
        if kind=='userTask':
            sub(sub(n,'bpmn:extensionElements'),'gobpm:assignmentDefinition',candidateGroups=opt['group'])
        if kind=='businessRuleTask':
            sub(sub(n,'bpmn:extensionElements'),'gobpm:calledDecision',decisionId='OnboardingRoute',resultVariable='route')
        if kind in ('intermediateCatchEvent','boundaryEvent'):
            sub(sub(n,'bpmn:timerEventDefinition'),'bpmn:timeDuration').text=opt.get('duration','PT1S')
        w,h=(160,85)
        if kind.endswith('Event'): w=h=36
        if kind.endswith('Gateway'): w=h=50
        geometry[nid]=(x,y,w,h)
        attrs={'id':nid+'_di','bpmnElement':nid}
        if kind=='callActivity': attrs['isExpanded']='false'
        shape=sub(plane,'bpmndi:BPMNShape',**attrs)
        sub(shape,'dc:Bounds',x=str(x),y=str(y),width=str(w),height=str(h))
    for i,(a,b,condition,label) in enumerate(edges):
        fid='Flow_'+str(i+1)
        f=sub(p,'bpmn:sequenceFlow',id=fid,sourceRef=a,targetRef=b,name=label)
        if condition=='default': elements[a].set('default',fid)
        elif condition: sub(f,'bpmn:conditionExpression',**{'{'+NS['xsi']+'}type':'bpmn:tFormalExpression'}).text=condition
        edge=sub(plane,'bpmndi:BPMNEdge',id=fid+'_di',bpmnElement=fid)
        x,y,w,h=geometry[a]; xx,yy,ww,hh=geometry[b]
        points=[(x+w,y+h/2),(xx,yy+hh/2)]
        if abs(y-yy)>60: points=[(x+w/2,y+h),(x+w/2,yy+hh/2),(xx,yy+hh/2)]
        if xx<x: points=[(x+w/2,y+h),(x+w/2,max(y+h,yy+hh)+80+i*3),(xx+ww/2,max(y+h,yy+hh)+80+i*3),(xx+ww/2,yy+hh)]
        for px,py in points: sub(edge,'di:waypoint',x=str(px),y=str(py))
    E.indent(root)
    (ROOT/'process'/f'{pid}.bpmn').write_bytes(E.tostring(root,encoding='utf-8',xml_declaration=True))

def n(kind,nid,label,x,y=150,**opt):return kind,nid,label,x,y,opt
def e(a,b,condition='',label=''):return a,b,condition,label

(ROOT/'process').mkdir(exist_ok=True)
process('RetailAccountOpening','GoBpmBank · Открытие счёта',[
 n('startEvent','Start','Заявление',30),n('callActivity','Intake','01 · Приём заявления',110,process='CollectApplication'),
 n('exclusiveGateway','Eligible','В нашем сценарии?',310),n('callActivity','Diligence','02 · Проверки и рассмотрение',420,process='DueDiligence'),
 n('exclusiveGateway','Approved','Можно продолжить?',620),n('callActivity','Contract','03 · Договор',740,process='ConcludeContract'),
 n('exclusiveGateway','Signed','Подписан?',940),n('callActivity','Open','04 · Открытие в АБС',1060,process='OpenCoreAccount'),
 n('exclusiveGateway','Opened','Зарегистрирован?',1260),n('callActivity','Post','05 · Постобработка',1380,process='PostOpening'),
 n('endEvent','Success','Счёт открыт',1600),n('endEvent','Stopped','Заявление завершено',740,410)],
 [e('Start','Intake'),e('Intake','Eligible'),e('Eligible','Diligence','outcome == "CONTINUE"','Да'),e('Eligible','Stopped','default','Другой исход'),
 e('Diligence','Approved'),e('Approved','Contract','outcome == "CONTINUE"','Да'),e('Approved','Stopped','default','Другой исход'),
 e('Contract','Signed'),e('Signed','Open','contractAccepted','Да'),e('Signed','Stopped','default','Клиент отказался'),
 e('Open','Opened'),e('Opened','Post','accountOpened','Да'),e('Opened','Stopped','default','Требует разбирательства'),e('Post','Success')])

process('CollectApplication','01 · Приём и идентификация',[
 n('startEvent','Start','',30),n('userTask','IdentifyApplicant','Заполнить и проверить анкету',120,group='operators'),
 n('task','ValidateApplication','Проверить границы примера',360),n('endEvent','End','Анкета получена',600),
 n('boundaryEvent','IntakeDeadline','Срок ожидания',240,218,attached='IdentifyApplicant',duration='PT5M'),
 n('task','ExpireApplication','Закрыть незаполненное заявление',360,400),n('endEvent','Expired','Ожидание истекло',600,400)],
 [e('Start','IdentifyApplicant'),e('IdentifyApplicant','ValidateApplication'),e('ValidateApplication','End'),e('IntakeDeadline','ExpireApplication'),e('ExpireApplication','Expired')])

process('AutomatedChecks','02a · Автоматические проверки',[
 n('startEvent','Start','',30,260),n('parallelGateway','Fork','Проверить параллельно',120,260),
 n('serviceTask','Document','Документ',270,80,job='checkDocument'),n('serviceTask','Screening','Перечни / ПДЛ',270,260,job='checkScreening'),
 n('serviceTask','Tax','Налоговые сведения',270,440,job='checkTax'),n('parallelGateway','Join','Собрать результаты',540,260),n('endEvent','End','Результаты собраны',680,260)],
 [e('Start','Fork'),e('Fork','Document'),e('Fork','Screening'),e('Fork','Tax'),e('Document','Join'),e('Screening','Join'),e('Tax','Join'),e('Join','End')])

process('DueDiligence','02 · Проверки и рассмотрение',[
 n('startEvent','Start','',30),n('callActivity','Checks','Проверить сведения',120,process='AutomatedChecks'),
 n('businessRuleTask','RouteApplication','DMN · Выбрать маршрут',350),n('exclusiveGateway','Route','Как продолжить?',570),
 n('callActivity','Clarify','Уточнить анкету',740,380,process='CollectApplication'),
 n('callActivity','Review','Ручное рассмотрение',740,150,process='ComplianceReview'),
 n('exclusiveGateway','ReviewResult','Решение сотрудника',960),n('task','Accept','Продолжить оформление',1150),n('endEvent','End','Рассмотрено',1370),
 n('task','Decline','Зафиксировать правовое основание',1150,380),n('exclusiveGateway','Clarified','Уточнение получено?',960,520),n('endEvent','Incomplete','Другой исход',1150,600)],
 [e('Start','Checks'),e('Checks','RouteApplication'),e('RouteApplication','Route'),e('Route','Accept','route == "CONTINUE"','Продолжить'),
 e('Route','Clarify','route == "REQUEST_INFO"','Уточнить'),e('Route','Review','default','Рассмотреть'),
 e('Review','ReviewResult'),e('ReviewResult','Accept','reviewDecision == "CONTINUE"','Продолжить'),
 e('ReviewResult','Clarify','reviewDecision == "REQUEST_INFO"','Уточнить'),e('ReviewResult','Decline','default','Мотивированный исход'),
 e('Clarify','Clarified'),e('Clarified','Checks','outcome == "CONTINUE"','Повторить проверки'),e('Clarified','Incomplete','default','Не продолжать'),
 e('Accept','End'),e('Decline','End')])

process('ComplianceReview','02b · Ручное рассмотрение',[
 n('startEvent','Start','',30),n('task','ResetReview','Начать новое рассмотрение',120),n('userTask','ReviewClient','Разрешить совпадение / рассмотреть',340,group='compliance'),
 n('exclusiveGateway','Special','Требуется второй сотрудник?',570),n('userTask','AuthorizeClient','Рассмотреть руководителю',700,group='supervisors'),
 n('task','NormalizeReview','Зафиксировать решение',940),n('endEvent','End','Решение принято',1180),
 n('boundaryEvent','ReviewDeadline','Эскалация',460,218,attached='ReviewClient',duration='PT2M')],
 [e('Start','ResetReview'),e('ResetReview','ReviewClient'),e('ReviewClient','Special'),e('Special','AuthorizeClient','screeningStatus == "PEP" and reviewDecision == "CONTINUE"','Да'),
 e('Special','NormalizeReview','default','Нет'),e('AuthorizeClient','NormalizeReview'),e('NormalizeReview','End'),e('ReviewDeadline','AuthorizeClient')])

process('ConcludeContract','03 · Заключение договора',[
 n('startEvent','Start','',30),n('serviceTask','Prepare','Подготовить учебный договор',120,job='prepareContract'),
 n('userTask','ConfirmContract','Подтвердить подписание / отказ',360,group='operators'),n('task','RecordConsent','Зафиксировать исход',600),n('endEvent','End','',840)],
 [e('Start','Prepare'),e('Prepare','ConfirmContract'),e('ConfirmContract','RecordConsent'),e('RecordConsent','End')])

process('OpenCoreAccount','04 · Открытие в мок-АБС',[
 n('startEvent','Start','',30),n('serviceTask','Customer','Найти / создать клиента',120,job='resolveCustomer'),
 n('serviceTask','Submit','Идемпотентная команда открытия',350,job='submitOpening'),n('serviceTask','Poll','Сверить результат в АБС',580,job='pollOpening'),
 n('exclusiveGateway','Status','Состояние операции',810),n('intermediateCatchEvent','Wait','Повторить через секунду',580,400,duration='PT1S'),
 n('task','ConfirmOpened','Подтвердить регистрацию',950),n('endEvent','End','Счёт открыт',1180),
 n('task','ReferCore','Передать разбирательство',950,400),n('endEvent','Referred','Не выдавать за отказ клиенту',1180,400)],
 [e('Start','Customer'),e('Customer','Submit'),e('Submit','Poll'),e('Poll','Status'),e('Status','Wait','coreStatus == "PENDING"','Ожидание'),
 e('Wait','Poll'),e('Status','ConfirmOpened','coreStatus == "OPENED"','Открыт'),e('Status','ReferCore','default','Отклонено АБС'),e('ConfirmOpened','End'),e('ReferCore','Referred')])

process('PostOpening','05 · Постобработка открытого счёта',[
 n('startEvent','Start','',30),n('serviceTask','Archive','Сохранить учебное досье',120,job='archiveDossier'),
 n('serviceTask','TaxReport','Сообщить ФНС (мок)',350,job='reportTax'),n('serviceTask','Notify','Уведомить клиента (мок)',580,job='notifyApplicant'),
 n('task','Finish','Завершить оформление',810),n('endEvent','End','Готово',1040)],
 [e('Start','Archive'),e('Archive','TaxReport'),e('TaxReport','Notify'),e('Notify','Finish'),e('Finish','End')])

# FIRST: маршрутизация учебного банка, не юридическое решение об отказе.
dmn='''<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="https://www.omg.org/spec/DMN/20191111/MODEL/" id="Definitions_Routing" name="Учебная маршрутизация" namespace="https://github.com/nicolas-vp/go-bpm-bank">
<decision id="OnboardingRoute" name="Маршрут заявления — учебные правила v1">
<decisionTable id="RoutingTable" hitPolicy="FIRST">
<input id="DocumentInput" label="Документ"><inputExpression id="DocumentExpr" typeRef="string"><text>documentStatus</text></inputExpression></input>
<input id="ScreeningInput" label="Проверочные списки"><inputExpression id="ScreeningExpr" typeRef="string"><text>screeningStatus</text></inputExpression></input>
<input id="TaxInput" label="Налоговые сведения"><inputExpression id="TaxExpr" typeRef="string"><text>taxStatus</text></inputExpression></input>
<output id="RouteOutput" name="route" label="Следующий шаг" typeRef="string"/>
'''
for rid,inputs,result in [
 ('MissingDocument',['"INCOMPLETE"','-','-'],'REQUEST_INFO'),
 ('DocumentReview',['"REVIEW"','-','-'],'REVIEW'),
 ('ScreeningReview',['-','"MATCH","PEP"','-'],'REVIEW'),
 ('TaxReview',['-','-','"REVIEW"'],'REVIEW'),
 ('Clear',['"VALID"','"CLEAR"','"VALID"'],'CONTINUE'),
 ('Unknown',['-','-','-'],'REVIEW')]:
    dmn+=f'<rule id="{rid}">'
    for j,text in enumerate(inputs):dmn+=f'<inputEntry id="{rid}_in{j}"><text>{text}</text></inputEntry>'
    dmn+=f'<outputEntry id="{rid}_out"><text>"{result}"</text></outputEntry></rule>\n'
dmn+='</decisionTable></decision></definitions>'
(ROOT/'process'/'routing.dmn').write_text(dmn)
print('Создано 8 BPMN и таблица решений DMN')
