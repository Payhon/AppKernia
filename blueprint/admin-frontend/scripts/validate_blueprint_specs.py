#!/usr/bin/env python3
from __future__ import annotations
import json,re,sys
from collections import defaultdict,deque
from pathlib import Path
from typing import Any
ROOT=Path(__file__).resolve().parents[1]
PROJECT_ROOT=ROOT.parents[1]
errors=[]
def fail(x:str)->None: errors.append(x)
def load(rel:str)->Any:
    try: return json.loads((ROOT/rel).read_text(encoding='utf-8'))
    except Exception as e: fail(f'{rel}: {e}'); return {}
def unique(xs,label):
    seen=set()
    for x in xs:
        if x in seen: fail(f'duplicate {label}: {x}')
        seen.add(x)
def nep(method,path): return method.upper(),re.sub(r'\{[^{}]+\}','{}',path.rstrip('/'))
menu_doc=load('spec/admin-menu-seed.json'); route_doc=load('spec/admin-route-registry.json')
perm_matrix=load('spec/page-permission-matrix.json'); page_map=load('spec/page-api-schema-map.json')
perm_snapshot=load('integration/backend-core-permissions.snapshot.json'); perm_delta=load('integration/core-permissions.delta.json')
try:
    backend_permission_source=json.loads((PROJECT_ROOT/'blueprint/backend/spec/core-permissions.json').read_text(encoding='utf-8'))
except Exception as e:
    fail(f'backend permission source: {e}'); backend_permission_source={}
schema_snapshot=load('integration/backend-core-schema-tables.snapshot.json'); api_snapshot=load('integration/backend-admin-api.snapshot.json'); api_delta=load('integration/admin-api-delta.json')
schema_coverage=load('spec/schema-ui-coverage.json')
menu_copy=load('integration/backend-core-menus.v2.json'); backlog=load('spec/agent-task-backlog.json')
menus=menu_doc.get('menus',[]); routes=route_doc.get('routes',[]); menu_by={m.get('code'):m for m in menus}; route_by={r.get('component_key'):r for r in routes}
unique([m.get('code') for m in menus],'menu code'); unique([m.get('path') for m in menus],'menu path')
roots=sorted(m['code'] for m in menus if not m.get('parent'))
if roots!=['dashboard','system']: fail(f'unexpected roots: {roots}')
max_depth=menu_doc.get('constraints',{}).get('max_depth',3)
for m in menus:
    code=m.get('code'); parent=m.get('parent')
    if parent and parent not in menu_by: fail(f'missing parent {parent} for {code}'); continue
    if m.get('type') not in {'directory','page','external'}: fail(f'invalid menu type {code}')
    if not str(m.get('path','')).startswith('/'): fail(f'invalid menu path {code}')
    seen=set(); cur=m; depth=1
    while cur.get('parent'):
        if cur['code'] in seen: fail(f'menu cycle {code}'); break
        seen.add(cur['code']); cur=menu_by[cur['parent']]; depth+=1
    if depth>max_depth: fail(f'menu too deep {code}: {depth}')
    if m.get('type')=='page' and m.get('component_key') not in route_by: fail(f'missing route for menu {code}')
if menu_doc!=menu_copy: fail('backend menu copy differs from menu spec')
unique([r.get('component_key') for r in routes],'route key'); unique([r.get('path') for r in routes],'route path'); unique([r.get('file') for r in routes],'route file')
menu_codes=set(menu_by); menu_page_keys={m['component_key'] for m in menus if m.get('type')=='page'}
for r in routes:
    key=r.get('component_key')
    route_path=str(r.get('path','')).strip('/') or 'index'
    route_source=PROJECT_ROOT/'apps/ak-admin/src/routes'/f'{route_path.replace("/", ".")}.tsx'
    if not route_source.is_file(): fail(f'missing TanStack route source for {key}: {route_source.relative_to(PROJECT_ROOT)}')
    if r.get('kind')=='page' and key not in menu_page_keys: fail(f'visible route without menu: {key}')
    if r.get('kind','').startswith('hidden') and r.get('menu_code'): fail(f'hidden route with menu_code: {key}')
    if r.get('active_menu_code') and r['active_menu_code'] not in menu_codes: fail(f'unknown active menu: {key}')
for key in ('auth.login','auth.register','auth.forgot-password','auth.reset-password','profile.basic','profile.security','profile.connections'):
    if key not in route_by: fail(f'missing required hidden route: {key}')
    if key in menu_page_keys: fail(f'hidden route leaked into menu: {key}')
existing={p['code'] for p in perm_snapshot.get('permissions',[])}; delta={p['code'] for p in perm_delta.get('permissions',[])}
if perm_snapshot.get('permissions',[])!=backend_permission_source.get('permissions',[]): fail('backend permission snapshot differs from actual Backend seed')
if existing&delta: fail(f'permission delta overlaps existing: {sorted(existing&delta)}')
allp=existing|delta
for m in menus:
    if m.get('permission') and m['permission'] not in allp: fail(f'unknown menu permission {m["permission"]}')
unique([p.get('component_key') for p in perm_matrix.get('pages',[])],'permission page')
for p in perm_matrix.get('pages',[]):
    if p.get('component_key') not in route_by: fail(f'permission page unknown route {p.get("component_key")}')
    for code in p.get('view',[])+list(p.get('actions',{}).values()):
        if code not in allp: fail(f'unknown permission {code} in {p.get("component_key")}')
    for code,status in p.get('permission_status',{}).items():
        expected='existing' if code in existing else 'delta_required'
        if status!=expected: fail(f'wrong permission status {code}: {status}')
known_tables=set(schema_snapshot.get('tables',[])); existing_eps={nep(x['method'],x['path']) for x in api_snapshot.get('endpoints',[])}; delta_eps={nep(x['method'],x['path']) for x in api_delta.get('endpoints',[])}
if existing_eps&delta_eps: fail(f'API delta overlaps existing: {sorted(existing_eps&delta_eps)}')
pattern=re.compile(r'^(GET|POST|PATCH|PUT|DELETE|HEAD|OPTIONS)\s+(/\S+)$'); map_pages=page_map.get('pages',[]); unique([p.get('component_key') for p in map_pages],'page contract')
for p in map_pages:
    key=p.get('component_key')
    if key not in route_by: fail(f'page contract unknown route {key}')
    for table in p.get('schema',[]):
        if table not in known_tables: fail(f'unknown table {table} in {key}')
    cov=[]
    for ep in p.get('endpoints',[]):
        m=pattern.fullmatch(ep)
        if not m: fail(f'bad endpoint {ep} in {key}'); continue
        n=nep(m.group(1),m.group(2))
        if n in existing_eps: cov.append('existing')
        elif n in delta_eps: cov.append('delta')
        else: fail(f'endpoint uncovered {ep} in {key}')
    if cov:
        expected='existing' if set(cov)=={'existing'} else 'delta_required' if set(cov)=={'delta'} else 'partial_delta'
        if p.get('backend_status')!=expected: fail(f'wrong backend status {key}: {p.get("backend_status")} vs {expected}')
required=menu_page_keys|{'profile.basic','profile.security','profile.connections','auth.login','auth.register','auth.forgot-password','auth.reset-password'}
missing=required-{p['component_key'] for p in map_pages}
if missing: fail(f'missing page contracts: {sorted(missing)}')
# Every backend schema table must be explicitly classified for UI exposure.
coverage_rows=schema_coverage.get('tables',[]); unique([x.get('table') for x in coverage_rows],'schema coverage table')
coverage_by={x.get('table'):x for x in coverage_rows}
if set(coverage_by)!=known_tables:
    fail(f'schema coverage mismatch: missing={sorted(known_tables-set(coverage_by))}, extra={sorted(set(coverage_by)-known_tables)}')
allowed_coverage={'page_backed','indirect_aggregate','backend_only'}
map_by_key={p.get('component_key'):p for p in map_pages}
for table,row in coverage_by.items():
    status=row.get('coverage'); pages=row.get('pages',[])
    if status not in allowed_coverage: fail(f'invalid schema coverage {table}: {status}')
    for key in pages:
        if key not in route_by: fail(f'schema coverage {table} points to unknown route {key}')
    if status=='page_backed':
        if not pages: fail(f'page_backed table without page: {table}')
        for key in pages:
            if table not in map_by_key.get(key,{}).get('schema',[]): fail(f'schema coverage drift {table} -> {key}')
    elif status=='backend_only' and pages:
        fail(f'backend_only table unexpectedly has pages: {table}')
# Docs API lists must match JSON.
md=(ROOT/'docs/PAGE_SPECIFICATIONS.md').read_text(encoding='utf-8'); blocks={m.group(1):m.group(2) for m in re.finditer(r'^## `([^`]+)`[^\n]*\n(.*?)(?=^## `|\Z)',md,re.M|re.S)}
for p in map_pages:
    key=p['component_key']
    if key not in blocks: fail(f'missing page doc block {key}'); continue
    part=blocks[key].split('**API**',1)[1].split('**页面验收**',1)[0]
    doc_eps=re.findall(r'^- `((?:GET|POST|PATCH|PUT|DELETE|HEAD|OPTIONS) /[^`]+)`$',part,re.M)
    if doc_eps!=p['endpoints']: fail(f'page doc API drift {key}')
# Task DAG.
tasks=backlog.get('tasks',[]); unique([t.get('id') for t in tasks],'task id'); by={t.get('id'):t for t in tasks}; indeg={k:0 for k in by}; graph=defaultdict(list)
backlog_md=(ROOT/'docs/AGENT_TASK_BACKLOG.md').read_text(encoding='utf-8')
documented_tasks=set(re.findall(r'`(AKADM-\d{3})`',backlog_md))
if documented_tasks!=set(by):
    fail(f'task docs/spec mismatch: missing-in-spec={sorted(documented_tasks-set(by))}, missing-in-docs={sorted(set(by)-documented_tasks)}')
for t in tasks:
    for dep in t.get('depends_on',[]):
        if dep not in by: fail(f'task {t.get("id")} unknown dep {dep}')
        else: graph[dep].append(t['id']); indeg[t['id']]+=1
q=deque([k for k,v in indeg.items() if v==0]); visited=0
while q:
    x=q.popleft(); visited+=1
    for y in graph[x]:
        indeg[y]-=1
        if indeg[y]==0:q.append(y)
if visited!=len(tasks): fail('task dependency cycle')
if errors:
    print('FAILED'); [print('-',e) for e in errors]; sys.exit(1)
print(f'PASSED: {len(menus)} menus, {len(routes)} routes, {len(allp)} permissions, {len(known_tables)} schema tables fully classified, {len(existing_eps)} existing APIs + {len(delta_eps)} deltas, {len(map_pages)} page contracts, {len(tasks)} agent tasks')
