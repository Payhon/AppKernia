from __future__ import annotations

import json, os, re, subprocess
from pathlib import Path
from playwright.sync_api import Page, Route, sync_playwright

ROOT=Path(__file__).resolve().parents[3];OUTPUT=ROOT/"output/playwright";AXE=ROOT/"apps/ak-admin/node_modules/axe-core/axe.min.js"
BASE=os.environ.get("AK_E2E_BASE_URL","http://127.0.0.1:4173").rstrip("/");EMAIL=os.environ["AK_E2E_EMAIL"];PASSWORD=os.environ["AK_E2E_PASSWORD"]
CONTAINER=os.environ.get("AK_E2E_POSTGRES_CONTAINER","appkernia-postgres-1")
def audit(page:Page,name:str,evidence:dict[str,object])->None:
    page.add_script_tag(path=str(AXE));result=page.evaluate("async()=>await axe.run({exclude:[['.ant-table-measure-row']]},{resultTypes:['violations']})");evidence[name]={"violations":[{"id":v["id"],"impact":v["impact"],"nodes":len(v["nodes"])} for v in result["violations"]]};assert not result["violations"],result["violations"]
    size=page.evaluate("()=>({client:document.documentElement.clientWidth,scroll:document.documentElement.scrollWidth})");assert size["scroll"]<=size["client"],size
def shot(page:Page,name:str,evidence:dict[str,object])->None:audit(page,name,evidence);page.screenshot(path=OUTPUT/f"admin-{name}.png",full_page=True)
def sql(statement:str)->str:
    result=subprocess.run(["docker","exec",CONTAINER,"psql","-U","appkernia","-d","appkernia","-At","-v","ON_ERROR_STOP=1","-c",statement],check=True,capture_output=True,text=True);return result.stdout.strip()
def login(page:Page)->None:
    page.goto(f"{BASE}/login",wait_until="networkidle");page.get_by_label("账号",exact=True).fill(EMAIL);page.get_by_label("密码",exact=True).fill(PASSWORD);page.locator('button[type="submit"]').click();page.wait_for_url(lambda u:u.startswith(f"{BASE}/dashboard"))
def fail_second_part_once(page:Page)->None:
    state={"failed":False}
    def handler(route:Route)->None:
        if not state["failed"]:state["failed"]=True;route.abort("failed")
        else:route.continue_()
    page.route("**/files/upload-sessions/*/parts/2",handler)
def main()->None:
    OUTPUT.mkdir(parents=True,exist_ok=True);evidence:dict[str,object]={};errors:list[str]=[];payload=b"A"*(5*1024*1024+2048)
    with sync_playwright() as p:
        browser=p.chromium.launch();context=browser.new_context(viewport={"width":1440,"height":900},locale="zh-CN",accept_downloads=True);page=context.new_page();page.on("console",lambda m:errors.append(m.text) if m.type=="error" and "ERR_FAILED" not in m.text else None);login(page)
        page.locator("select.ak-locale-switcher").select_option("en-US");page.locator(".ak-desktop-sider").get_by_role("link",name="Files",exact=True).click();page.get_by_role("heading",name="File storage",exact=True).wait_for()
        sql("DELETE FROM storage.file_usages WHERE module_code='e2e' AND file_id IN (SELECT id FROM storage.files WHERE original_name IN ('resume-e2e.txt','cancel-e2e.txt'));")
        while page.get_by_role("row").filter(has_text="resume-e2e.txt").count()>0:
            stale=page.get_by_role("row").filter(has_text="resume-e2e.txt").first;stale.get_by_role("button",name="Delete",exact=True).click();page.get_by_text("The file object will be removed and cannot be recovered.",exact=True).wait_for();page.get_by_role("button",name="Delete",exact=True).last.click();page.get_by_text("File deleted.",exact=True).wait_for();page.get_by_role("row").filter(has_text="resume-e2e.txt").wait_for(state="detached")
        shot(page,"files.en-US.1440.empty",evidence)
        fail_second_part_once(page);page.locator('input[type="file"]').set_input_files({"name":"resume-e2e.txt","mimeType":"text/plain","buffer":payload});page.get_by_text("Upload failed. Resume to retry missing parts.",exact=True).wait_for()
        with page.expect_response(lambda r:"/files/upload-sessions/" in r.url and r.url.endswith("/complete")) as completed_response:page.get_by_role("button",name="Resume",exact=True).click()
        response=completed_response.value;assert response.status==201,response.status;file_id=response.json()["data"]["id"];page.get_by_text("File uploaded and verified.",exact=True).wait_for();page.get_by_text("resume-e2e.txt",exact=True).first.wait_for();evidence["multipart_resume"]={"complete_status":response.status,"file_id":file_id}
        row=page.get_by_role("row").filter(has_text="resume-e2e.txt");row.get_by_role("button",name="View details",exact=True).click();page.wait_for_url(f"{BASE}/system/storage/files/{file_id}");page.get_by_role("heading",name="resume-e2e.txt",exact=True).wait_for();shot(page,"files.en-US.1440.detail",evidence);page.get_by_role("link",name="Back",exact=True).click();page.wait_for_url(f"{BASE}/system/storage/files");row=page.get_by_role("row").filter(has_text="resume-e2e.txt")
        with page.expect_download() as download_info:
            row.get_by_role("button",name="Download",exact=True).click()
        assert download_info.value.suggested_filename=="resume-e2e.txt";evidence["download"]="passed"
        inserted=sql(f"INSERT INTO storage.file_usages(file_id,tenant_id,module_code,entity_type,entity_id,field_name) SELECT id,tenant_id,'e2e','storage.file',owner_user_id,'attachment' FROM storage.files WHERE id='{file_id}' RETURNING id;");assert inserted
        row.get_by_role("button",name="Delete",exact=True).click();warning=page.get_by_text(re.compile(r"This file is referenced .* cannot be deleted\."));warning.wait_for();evidence["delete_in_use_warning"]=warning.inner_text();shot(page,"files.en-US.1440.in-use",evidence);page.get_by_role("button",name="OK",exact=True).click();page.locator(".ant-modal-wrap").wait_for(state="hidden")
        sql(f"DELETE FROM storage.file_usages WHERE file_id='{file_id}' AND module_code='e2e';");row.get_by_role("button",name="Delete",exact=True).click();page.get_by_text("The file object will be removed and cannot be recovered.",exact=True).wait_for();page.get_by_role("button",name="Delete",exact=True).last.click();page.get_by_text("File deleted.",exact=True).wait_for();evidence["delete_unused"]="passed"
        page.unroute("**/files/upload-sessions/*/parts/2");fail_second_part_once(page);page.locator('input[type="file"]').set_input_files({"name":"cancel-e2e.txt","mimeType":"text/plain","buffer":payload});page.get_by_text("Upload failed. Resume to retry missing parts.",exact=True).wait_for();page.get_by_role("button",name="Cancel upload",exact=True).click();page.get_by_text("Upload session cancelled and uploaded parts discarded.",exact=True).wait_for();evidence["cancelled_sessions"]=int(sql("SELECT count(*) FROM storage.upload_sessions WHERE original_name='cancel-e2e.txt' AND status='aborted';"));assert evidence["cancelled_sessions"]>=1
        page.locator("select.ak-locale-switcher").select_option("zh-CN");page.get_by_role("heading",name="文件存储",exact=True).wait_for();shot(page,"files.zh-CN.1440",evidence);page.set_viewport_size({"width":375,"height":812});shot(page,"files.zh-CN.375",evidence)
        evidence["audits"]=int(sql("SELECT count(*) FROM audit.operation_logs WHERE action_name IN ('storage.file.upload.complete','storage.file.upload.cancel','storage.file.delete');"));assert evidence["audits"]>=3;browser.close()
    evidence["unexpected_console_errors"]=errors;assert not errors,errors;(OUTPUT/"admin-files-e2e-results.json").write_text(json.dumps(evidence,ensure_ascii=False,indent=2)+"\n",encoding="utf-8")
if __name__=="__main__":main()
