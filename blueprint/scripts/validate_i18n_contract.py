#!/usr/bin/env python3
from pathlib import Path
import json, re, sys

ROOT = Path(__file__).resolve().parents[2]
BP = ROOT / "blueprint"
errors=[]

def load(p):
    try: return json.loads(p.read_text(encoding="utf-8"))
    except Exception as e:
        errors.append(f"cannot parse {p}: {e}"); return {}

def placeholders(value):
    if not isinstance(value,str): return set()
    return set(re.findall(r"\{([A-Za-z_][A-Za-z0-9_]*)\}", value))

contract=load(BP/"i18n-contract.json")
if contract.get("default_locale")!="zh-CN": errors.append("default_locale must be zh-CN")
if contract.get("fallback_locale")!="zh-CN": errors.append("fallback_locale must be zh-CN")
if [x.get("code") for x in contract.get("supported_locales",[])] != ["zh-CN","en-US"]:
    errors.append("supported locales must be exactly zh-CN, en-US for Core 1.0")
required_namespaces=contract.get("required_namespaces",[])
if not isinstance(required_namespaces,list) or not required_namespaces or not all(isinstance(name,str) and name for name in required_namespaces):
    errors.append("required_namespaces must be a non-empty string array")
    required_namespaces=[]
elif len(required_namespaces) != len(set(required_namespaces)):
    errors.append("required_namespaces must not contain duplicates")

for area in ("backend","admin","mobile"):
    zh=load(BP/f"i18n/{area}/zh-CN.json")
    en=load(BP/f"i18n/{area}/en-US.json")
    if set(zh)!=set(en):
        errors.append(f"{area}: key mismatch missing-en={sorted(set(zh)-set(en))} missing-zh={sorted(set(en)-set(zh))}")
    for k in set(zh)&set(en):
        if placeholders(zh[k]) != placeholders(en[k]):
            errors.append(f"{area}: placeholder mismatch at {k}")

admin_app_root=ROOT/"apps/ak-admin/src/locales"
if admin_app_root.exists():
    namespace_names=set(required_namespaces)
    app_catalogs={}
    for locale in ("zh-CN","en-US"):
        locale_dir=admin_app_root/locale
        files={p.stem for p in locale_dir.glob("*.json")}
        if files != namespace_names:
            errors.append(f"admin app {locale}: namespace mismatch missing={sorted(namespace_names-files)} extra={sorted(files-namespace_names)}")
        merged={}
        for path in sorted(locale_dir.glob("*.json")):
            values=load(path)
            overlap=set(merged)&set(values)
            if overlap: errors.append(f"admin app {locale}: duplicate namespace keys {sorted(overlap)}")
            merged.update(values)
        app_catalogs[locale]=merged
        reference=load(BP/f"i18n/admin/{locale}.json")
        if merged != reference:
            errors.append(f"admin app {locale}: generated catalogs differ from blueprint reference")

menu_path=BP/"admin-frontend/spec/admin-menu-seed.json"
if menu_path.exists():
    menu=load(menu_path)
    admin_zh=load(BP/"i18n/admin/zh-CN.json")
    admin_en=load(BP/"i18n/admin/en-US.json")
    for m in menu.get("menus",[]):
        key=m.get("i18n_key")
        if not key: errors.append(f"menu {m.get('code')} missing i18n_key")
        elif key not in admin_zh or key not in admin_en: errors.append(f"menu key absent from packs: {key}")

route_path=BP/"admin-frontend/spec/admin-route-registry.json"
if route_path.exists():
    routes=load(route_path)
    admin_zh=load(BP/"i18n/admin/zh-CN.json")
    admin_en=load(BP/"i18n/admin/en-US.json")
    for r in routes.get("routes",[]):
        key=r.get("title_key")
        if not key: errors.append(f"route {r.get('path')} missing title_key")
        elif key not in admin_zh or key not in admin_en: errors.append(f"route key absent from packs: {key}")

required=[
 BP/"I18N_CONTRACT.md",
 BP/"backend/docs/I18N_ARCHITECTURE.md",
 BP/"admin-frontend/docs/I18N_ARCHITECTURE.md",
 BP/"mobile/docs/I18N_ARCHITECTURE.md",
 ROOT/"AGENTS.md",
 ROOT/"CODEX_PROMPT.md",
]
for p in required:
    if not p.exists(): errors.append(f"missing required file: {p}")

if errors:
    print("AppKernia i18n blueprint validation: FAILED")
    for e in errors: print("-",e)
    sys.exit(1)
print("AppKernia i18n blueprint validation: PASSED")
print("Locales: zh-CN, en-US")
print("Default/fallback: zh-CN")
print("Reference packs: backend, admin, mobile")
