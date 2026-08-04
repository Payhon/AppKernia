#!/usr/bin/env python3
from __future__ import annotations
import json
import os
import pathlib
import re
import sys
from collections import defaultdict, deque

HERE = pathlib.Path(__file__).resolve().parent
MOBILE = HERE.parent
BLUEPRINT = MOBILE.parent
PROJECT_ROOT = BLUEPRINT.parent
BACKEND_DIR = pathlib.Path(os.environ.get('AK_BACKEND_BLUEPRINT_DIR', str(BLUEPRINT / 'backend'))).resolve()

errors: list[str] = []
warnings: list[str] = []

def load(rel: str):
    p = MOBILE / rel
    try:
        return json.loads(p.read_text(encoding='utf-8'))
    except Exception as exc:
        errors.append(f'{rel}: JSON load failed: {exc}')
        return {}

def unique(items, key, label):
    seen = {}
    for item in items:
        value = item.get(key)
        if value in seen:
            errors.append(f'duplicate {label}: {value}')
        seen[value] = item
    return seen

routes_doc = load('spec/mobile-route-registry.json')
routes = routes_doc.get('routes', [])
routes_by_key = unique(routes, 'route_key', 'route_key')
unique(routes, 'path', 'route path')
unique(routes, 'file', 'route file')

if routes_doc.get('rules', {}).get('first_page') not in routes_by_key:
    errors.append('first_page does not exist')

# Tabs
tabs = load('spec/mobile-tabbar.json').get('tabs', [])
unique(tabs, 'tab_key', 'tab_key')
for tab in tabs:
    route = routes_by_key.get(tab.get('route_key'))
    if not route:
        errors.append(f"tab route missing: {tab.get('route_key')}")
    elif route.get('kind') != 'tab' or route.get('tab_key') != tab.get('tab_key'):
        errors.append(f"tab/route mismatch: {tab.get('tab_key')}")

# APIs
baseline = load('integration/app-api-baseline.json').get('endpoints', [])
delta = load('integration/app-api-delta.json').get('endpoints', [])
api_keys = {f"{e['method']} {e['path']}" for e in baseline + delta}
if len(api_keys) != len(baseline) + len(delta):
    errors.append('duplicate API endpoint across baseline/delta')
page_api = load('spec/page-api-map.json').get('pages', [])
unique(page_api, 'route_key', 'page-api route_key')
for entry in page_api:
    if entry.get('route_key') not in routes_by_key:
        errors.append(f"page-api unknown route: {entry.get('route_key')}")
    for endpoint in entry.get('endpoints', []):
        if endpoint not in api_keys:
            errors.append(f"unknown page API: {entry.get('route_key')} -> {endpoint}")
for r in routes:
    mapped = next((x for x in page_api if x.get('route_key') == r.get('route_key')), None)
    if mapped is None or mapped.get('endpoints') != r.get('apis'):
        errors.append(f"route/page-api mismatch: {r.get('route_key')}")

# Permissions
permission_delta = load('integration/app-permissions.delta.json').get('permissions', [])
perm_codes = {p['code'] for p in permission_delta}
backend_perm_path = BACKEND_DIR / 'spec' / 'core-permissions.json'
if backend_perm_path.exists():
    backend = json.loads(backend_perm_path.read_text(encoding='utf-8'))
    perm_codes |= {p['code'] for p in backend.get('permissions', [])}
else:
    # Standalone delivery validation knows the subset used from the backend blueprint.
    perm_codes |= {
        'iam.session.read_self','iam.session.revoke_self','iam.device.read_self','iam.device.revoke_self',
        'notify.message.read_self','notify.message.mark_read_self','iam.mfa.manage_self',
        'iam.oauth.manage_self','storage.file.upload_self'
    }
    warnings.append('blueprint/backend not present beside mobile; used known permission subset')

matrix = load('spec/page-permission-matrix.json').get('pages', [])
unique(matrix, 'route_key', 'permission matrix route_key')
for entry in matrix:
    if entry.get('route_key') not in routes_by_key:
        errors.append(f"permission matrix unknown route: {entry.get('route_key')}")
    for code in entry.get('view_permissions', []) + entry.get('action_permissions', []):
        if code not in perm_codes:
            errors.append(f"unknown permission: {entry.get('route_key')} -> {code}")
for r in routes:
    m = next((x for x in matrix if x.get('route_key') == r.get('route_key')), None)
    expected = (m.get('view_permissions', []) + m.get('action_permissions', [])) if m else []
    if m is None or expected != r.get('required_permissions', []):
        errors.append(f"route/permission mismatch: {r.get('route_key')}")

# Feature flags
flags = load('spec/feature-flags.json').get('flags', [])
flag_keys = set(unique(flags, 'key', 'feature flag'))
for r in routes:
    flag = r.get('feature_flag')
    if flag and flag not in flag_keys and flag != 'onboarding_enabled':
        errors.append(f"unknown route feature flag: {r.get('route_key')} -> {flag}")
# onboarding_enabled is intentionally a route but must be declared.
if any(r.get('feature_flag') == 'onboarding_enabled' for r in routes) and 'onboarding_enabled' not in flag_keys:
    errors.append('onboarding_enabled route flag missing from feature-flags.json')

# Components
components_doc = load('spec/component-compatibility-matrix.json')
components = components_doc.get('components', [])
unique(components, 'wrapper', 'component wrapper')
valid_status = set(components_doc.get('statuses', {}))
for c in components:
    if c.get('status') not in valid_status:
        errors.append(f"invalid component status: {c.get('wrapper')}")
    for platform in ('android','ios','harmony'):
        if platform not in c:
            errors.append(f"missing component platform field: {c.get('wrapper')} {platform}")

# Platform matrix
platforms = load('spec/platform-matrix.json').get('targets', [])
platform_keys = {x.get('platform') for x in platforms}
if platform_keys != {'android','ios','harmony'}:
    errors.append(f'platform matrix must be exactly android/ios/harmony, got {platform_keys}')

# Privacy capability matrix
privacy = load('spec/privacy-capability-matrix.json').get('capabilities', [])
unique(privacy, 'key', 'privacy capability')
for capability in privacy:
    for platform in ('android','ios','harmony'):
        if platform not in capability:
            errors.append(f"missing privacy platform field: {capability.get('key')} {platform}")

# Task DAG
tasks = load('spec/agent-task-backlog.json').get('tasks', [])
task_by_id = unique(tasks, 'id', 'task id')
task_doc = (MOBILE / 'docs' / 'AGENT_TASK_BACKLOG.md').read_text(encoding='utf-8')
documented_tasks = set(re.findall(r'^## (AKMOB-\d{3})\b', task_doc, re.M))
if documented_tasks != set(task_by_id):
    errors.append(
        'task docs/spec mismatch: '
        f'missing-in-spec={sorted(documented_tasks-set(task_by_id))}, '
        f'missing-in-docs={sorted(set(task_by_id)-documented_tasks)}'
    )
indegree = {i: 0 for i in task_by_id}
graph = defaultdict(list)
for task in tasks:
    for dep in task.get('depends_on', []):
        if dep not in task_by_id:
            errors.append(f"task dependency missing: {task.get('id')} -> {dep}")
            continue
        graph[dep].append(task['id'])
        indegree[task['id']] += 1
q = deque([k for k,v in indegree.items() if v == 0])
visited = 0
while q:
    n = q.popleft(); visited += 1
    for nxt in graph[n]:
        indegree[nxt] -= 1
        if indegree[nxt] == 0: q.append(nxt)
if visited != len(task_by_id):
    errors.append('agent task graph contains cycle')

# Required docs
required = [
    'AGENTS.md','AK_MOBILE_BLUEPRINT.md','README.md',
    'docs/ARCHITECTURE.md','docs/UVIEW_ULTRA_INTEGRATION.md','docs/INFORMATION_ARCHITECTURE.md',
    'docs/PAGE_SPECIFICATIONS.md','docs/API_STATE_SECURITY.md','docs/PLATFORM_COMPATIBILITY.md',
    'docs/UI_UX_PRO_MAX_WORKFLOW.md','docs/TESTING_QUALITY_GATES.md','docs/AGENT_TASK_BACKLOG.md',
    'docs/PRIVACY_PERMISSION_MATRIX.md','docs/THIRD_PARTY_NOTICES.md','docs/THIRD_PARTY_PATCHES.md',
    'integration/BACKEND_CHANGES_REQUIRED.md','prompts/AGENT_BOOTSTRAP_PROMPT.md'
]
for rel in required:
    if not (MOBILE / rel).is_file():
        errors.append(f'missing required file: {rel}')

print('AppKernia Mobile Blueprint Validation')
print(f'Routes: {len(routes)}')
print(f'Tabs: {len(tabs)}')
print(f'Baseline APIs: {len(baseline)}')
print(f'API delta: {len(delta)}')
print(f'Permission delta: {len(permission_delta)}')
print(f'Components: {len(components)}')
print(f'Privacy capabilities: {len(privacy)}')
print(f'Tasks: {len(tasks)}')
print(f'Platforms: {len(platforms)}')
for w in warnings:
    print(f'WARNING: {w}')
for e in errors:
    print(f'ERROR: {e}')
if errors:
    print(f'FAILED: {len(errors)} error(s), {len(warnings)} warning(s)')
    sys.exit(1)
print(f'PASSED: 0 errors, {len(warnings)} warning(s)')
