from __future__ import annotations
import json, os, re, uuid
from pathlib import Path
from playwright.sync_api import Page, sync_playwright

ROOT=Path(__file__).resolve().parents[3];OUTPUT=ROOT/"output"/"playwright";AXE=ROOT/"apps"/"ak-admin"/"node_modules"/"axe-core"/"axe.min.js";BASE=os.environ.get("AK_E2E_BASE_URL","http://127.0.0.1:4174").rstrip("/");EMAIL=os.environ["AK_E2E_EMAIL"];PASSWORD=os.environ["AK_E2E_PASSWORD"]
def audit(page:Page,name:str,evidence:dict[str,object]):
 page.add_script_tag(path=str(AXE));result=page.evaluate("async()=>await axe.run({exclude:[['.ant-table-measure-row']]},{resultTypes:['violations']})");violations=[{"id":x["id"],"impact":x["impact"],"nodes":len(x["nodes"])} for x in result["violations"]];severe=[x for x in violations if x["impact"] in("critical","serious")];dimensions=page.evaluate("()=>({client:document.documentElement.clientWidth,scroll:document.documentElement.scrollWidth})");evidence[name]={"violations":violations,"critical_or_serious":len(severe),"dimensions":dimensions};assert not severe,severe;assert dimensions["scroll"]<=dimensions["client"],dimensions
def shot(page:Page,name:str,evidence:dict[str,object]):
 audit(page,name,evidence);page.evaluate("()=>{if(document.activeElement instanceof HTMLElement)document.activeElement.blur();document.querySelector('.ak-skip-link')?.blur()}");page.screenshot(path=OUTPUT/f"admin-operations.{name}.png",full_page=True)
def login(page:Page):
 page.goto(f"{BASE}/login",wait_until="networkidle");page.get_by_label("账号",exact=True).fill(EMAIL);page.get_by_label("密码",exact=True).fill(PASSWORD);page.locator('button[type="submit"]').click();page.wait_for_url(lambda value:value.startswith(f"{BASE}/dashboard"));
 if page.locator("html").get_attribute("lang")=="zh-CN":page.get_by_label("显示语言",exact=True).select_option(label="English");page.get_by_label("Display language",exact=True).wait_for();page.wait_for_timeout(500)
def ensure_english(page:Page):
 if page.locator("html").get_attribute("lang")=="zh-CN":page.get_by_label("显示语言",exact=True).select_option(label="English")
 page.get_by_label("Display language",exact=True).wait_for();page.wait_for_timeout(500)
def main():
 OUTPUT.mkdir(parents=True,exist_ok=True);evidence:dict[str,object]={};console_errors:list[str]=[];raw_subject=f"203.0.113.{int(uuid.uuid4().hex[:2],16)%200+1}"
 with sync_playwright() as p:
  browser=p.chromium.launch();context=browser.new_context(locale="zh-CN",viewport={"width":1440,"height":900});page=context.new_page();page.on("console",lambda m:console_errors.append(m.text) if m.type=="error" else None);login(page)
  page.locator('.ak-desktop-sider a[href="/system/security/block-rules"]').click();page.wait_for_url(f"{BASE}/system/security/block-rules");ensure_english(page);page.get_by_role("heading",name="Access Rules",exact=True).wait_for();shot(page,"block-rules.en-US.1440.empty",evidence);page.get_by_role("button",name="Create rule",exact=True).click();page.get_by_label("Subject value",exact=True).fill(raw_subject);page.get_by_label("Reason",exact=True).fill("E2E impact verification");save=page.get_by_role("button",name="Save",exact=True);assert save.is_disabled();page.get_by_role("checkbox",name="I verified the subject, scope, action, and lifetime",exact=True).check();
  with page.expect_response(lambda r:r.url.endswith("/admin-api/v1/block-rules") and r.request.method=="POST") as created:save.click()
  assert created.value.status==201,created.value.text();payload=created.value.json()["data"];assert raw_subject not in json.dumps(payload);assert payload["subject_hint"]!=raw_subject;page.get_by_text("Access rule saved",exact=True).wait_for();row=page.get_by_role("row").filter(has_text=payload["subject_hint"]);row.wait_for();assert raw_subject not in page.locator("body").inner_text();shot(page,"block-rules.en-US.1440.saved",evidence);page.get_by_label("Display language",exact=True).select_option(label="简体中文");page.get_by_role("heading",name="访问控制",exact=True).wait_for()
  for width,height in((1024,768),(768,1024),(375,812)):
   page.set_viewport_size({"width":width,"height":height});page.wait_for_timeout(200);shot(page,f"block-rules.zh-CN.{width}.saved",evidence)
  page.set_viewport_size({"width":1440,"height":900});page.wait_for_timeout(500);row=page.get_by_role("row").filter(has_text=payload["subject_hint"]);revoke_button=row.get_by_role("button",name=re.compile(r"撤\s*销"));revoke_button.wait_for();revoke_button.click();dialog=page.get_by_role("dialog",name="撤销访问控制规则？",exact=True)
  with page.expect_response(lambda r:"/block-rules/" in r.url and r.request.method=="DELETE") as revoked:dialog.get_by_role("button",name=re.compile(r"撤\s*销")).click()
  assert revoked.value.status==200;page.get_by_text("访问控制规则已撤销",exact=True).wait_for();page.get_by_label("显示语言",exact=True).select_option(label="English");page.locator('.ak-desktop-sider a[href="/system/monitoring/health"]').click();page.wait_for_url(f"{BASE}/system/monitoring/health");ensure_english(page);page.get_by_role("heading",name="Service Status",exact=True).wait_for();body=page.locator("body").inner_text().lower();assert "postgres://" not in body and "appkernia-dev-only" not in body and "/users/" not in body;shot(page,"health.en-US.1440",evidence);page.get_by_label("Display language",exact=True).select_option(label="简体中文")
  for width,height in((768,1024),(375,812)):
   page.set_viewport_size({"width":width,"height":height});page.wait_for_timeout(200);shot(page,f"health.zh-CN.{width}",evidence)
  evidence["runtime"]={"raw_subject_returned":False,"subject_hint":payload["subject_hint"],"revoke_status":revoked.value.status,"health_secret_scan":True,"locales":["zh-CN","en-US"]};evidence["unexpected_console_errors"]=console_errors;assert not console_errors,console_errors;(OUTPUT/"admin-operations-e2e-results.json").write_text(json.dumps(evidence,ensure_ascii=False,indent=2)+"\n",encoding="utf-8");context.close();browser.close()
 print(json.dumps(evidence,ensure_ascii=False,indent=2))
if __name__=="__main__":main()
