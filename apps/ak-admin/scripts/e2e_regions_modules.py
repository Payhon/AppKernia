from __future__ import annotations

import json, os
from pathlib import Path
from playwright.sync_api import Page, sync_playwright

ROOT=Path(__file__).resolve().parents[3]; OUTPUT=ROOT/"output/playwright"; AXE=ROOT/"apps/ak-admin/node_modules/axe-core/axe.min.js"
BASE=os.environ.get("AK_E2E_BASE_URL","http://127.0.0.1:4174").rstrip("/"); EMAIL=os.environ["AK_E2E_EMAIL"]; PASSWORD=os.environ["AK_E2E_PASSWORD"]
def audit(page:Page,name:str,evidence:dict[str,object])->None:
    page.add_script_tag(path=str(AXE)); result=page.evaluate("async()=>await axe.run({exclude:[['.ant-table-measure-row']]},{resultTypes:['violations']})")
    evidence[name]={"violations":[{"id":v["id"],"impact":v["impact"],"nodes":len(v["nodes"])} for v in result["violations"]]}; assert not result["violations"],result["violations"]
    size=page.evaluate("()=>({client:document.documentElement.clientWidth,scroll:document.documentElement.scrollWidth})"); assert size["scroll"]<=size["client"],size
def shot(page:Page,name:str,evidence:dict[str,object])->None: audit(page,name,evidence);page.screenshot(path=OUTPUT/f"admin-{name}.png",full_page=True)
def main()->None:
    OUTPUT.mkdir(parents=True,exist_ok=True); evidence:dict[str,object]={}; errors:list[str]=[]
    with sync_playwright() as p:
        browser=p.chromium.launch(); context=browser.new_context(viewport={"width":1440,"height":900},locale="zh-CN"); page=context.new_page();page.on("console",lambda m:errors.append(m.text) if m.type=="error" else None)
        page.goto(f"{BASE}/login",wait_until="networkidle");page.get_by_label("账号",exact=True).fill(EMAIL);page.get_by_label("密码",exact=True).fill(PASSWORD);page.locator('button[type="submit"]').click();page.wait_for_url(lambda u:u.startswith(f"{BASE}/dashboard"))
        page.get_by_label("显示语言",exact=True).select_option(label="English");page.locator(".ak-desktop-sider").get_by_role("link",name="Regions",exact=True).click();page.get_by_role("heading",name="Regions",exact=True).wait_for();shot(page,"regions.en-US.1440",evidence)
        with page.expect_response(lambda r:"/admin-api/v1/regions?" in r.url and "parent_code=110000" in r.url) as child_response: page.get_by_role("button",name="Expand region",exact=True).first.click()
        assert child_response.value.status==200;page.get_by_text("市辖区",exact=True).wait_for();evidence["lazy_child_status"]=child_response.value.status
        page.locator(".ak-desktop-sider").get_by_role("link",name="Modules",exact=True).click();page.get_by_role("heading",name="Modules",exact=True).wait_for();page.get_by_text("e2e.catalog",exact=True).wait_for();assert page.get_by_role("button",name="Install",exact=True).count()==0;shot(page,"modules.en-US.1440",evidence)
        page.get_by_label("Display language",exact=True).select_option(label="简体中文");page.locator(".ak-desktop-sider").get_by_role("link",name="地区管理",exact=True).click();page.get_by_role("heading",name="地区管理",exact=True).wait_for();shot(page,"regions.zh-CN.1440",evidence)
        page.set_viewport_size({"width":375,"height":812});shot(page,"regions.zh-CN.375",evidence);page.get_by_role("button",name="打开导航",exact=True).click();page.get_by_role("link",name="模块信息",exact=True).click();page.get_by_role("heading",name="模块信息",exact=True).wait_for();shot(page,"modules.zh-CN.375",evidence)
        browser.close()
    evidence["unexpected_console_errors"]=errors;assert not errors,errors;(OUTPUT/"admin-regions-modules-e2e-results.json").write_text(json.dumps(evidence,ensure_ascii=False,indent=2)+"\n",encoding="utf-8")
if __name__=="__main__":main()
