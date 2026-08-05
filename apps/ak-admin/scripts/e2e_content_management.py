from __future__ import annotations
import json, os, shutil
from pathlib import Path
from playwright.sync_api import Route, sync_playwright

ROOT=Path(__file__).resolve().parents[3]; BASE=os.environ.get("AK_E2E_BASE_URL","http://127.0.0.1:4173"); AXE=ROOT/"apps/ak-admin/node_modules/axe-core/axe.min.js"; OUT=ROOT/"apps/ak-admin/artifacts/ui-ux-pro-max/AKADM-content-management/screenshots"; OUT.mkdir(parents=True,exist_ok=True)
PERMS=["content.article.read","content.article.create","content.article.update","content.article.delete","content.article.publish","content.article.archive","content.category.read","content.category.create","content.category.update","content.category.delete","storage.file.read"]
CAT={"id":"123e4567-e89b-12d3-a456-426614174000","slug":"news","status":"active","sort_order":1,"lock_version":1,"created_at":"2026-08-05T00:00:00Z","updated_at":"2026-08-05T00:00:00Z","translations":{"zh-CN":{"name":"新闻","description":""},"en-US":{"name":"News","description":""}}}
ARTICLE={"id":"223e4567-e89b-12d3-a456-426614174000","category_id":CAT["id"],"slug":"welcome","status":"draft","featured":True,"sort_order":1,"cover_file_id":None,"cover_url":None,"reading_minutes":5,"lock_version":1,"published_at":None,"created_at":"2026-08-05T00:00:00Z","updated_at":"2026-08-05T00:00:00Z","translations":{"zh-CN":{"title":"欢迎使用","summary":"欢迎","body_format":"markdown","body":"正文"},"en-US":{"title":"Welcome","summary":"Welcome","body_format":"markdown","body":"Body"}}}
def respond(route:Route,status:int,data:object): route.fulfill(status=status,content_type="application/json",body=json.dumps({"code":"OK","message":"OK","data":data,"request_id":"content-e2e"}))
def handler(route:Route):
 u=route.request.url
 if u.endswith("/auth/public-config"): return respond(route,200,{"locale":"zh-CN","default_locale":"zh-CN","supported_locales":["zh-CN","en-US"],"feature_flags":{},"settings":{}})
 if u.endswith("/auth/login"): return respond(route,200,{"access_token":"e2e","token_type":"Bearer","expires_in":900,"csrf_token":"csrf-e2e-value-long-enough"})
 if u.endswith("/auth/context"): return respond(route,200,{"user":{"id":"323e4567-e89b-12d3-a456-426614174000","email":"e2e@app.test","display_name":"E2E","locale":"zh-CN","time_zone":"UTC","avatar_url":None},"active_tenant":{"id":"423e4567-e89b-12d3-a456-426614174000","name":"E2E","code":"e2e"},"available_tenants":[],"roles":[],"permissions":PERMS,"menus":[],"feature_flags":{},"menu_revision":"1","permission_revision":"1","server_time":"2026-08-05T00:00:00Z"})
 if u.endswith("/me") and route.request.method == "PATCH": return respond(route,200,{"id":"323e4567-e89b-12d3-a456-426614174000","email":"e2e@app.test","display_name":"E2E","locale":"en-US","time_zone":"UTC","avatar_url":None})
 if "/content/categories" in u: return respond(route,200,{"items":[CAT],"page":1,"page_size":20,"total":1})
 if "/content/articles" in u: return respond(route,200,{"items":[ARTICLE],"page":1,"page_size":20,"total":1})
 if "/files?" in u: return respond(route,200,{"items":[],"page":1,"page_size":50,"total":0})
 return respond(route,200,{})
def axe(page,name,evidence):
 page.add_script_tag(path=str(AXE)); result=page.evaluate("async()=>await axe.run({exclude:[['.ant-table-measure-row']]},{resultTypes:['violations']})"); severe=[v for v in result["violations"] if v["impact"] in("serious","critical")]; evidence[name]={"serious_critical":len(severe)}; assert not severe,severe
def main():
 print("content-e2e:start",flush=True)
 evidence={}
 with sync_playwright() as p:
  b=p.chromium.launch(); c=b.new_context(locale="zh-CN",viewport={"width":1440,"height":900}); page=c.new_page(); page.route("**/admin-api/**",handler)
  page.on("console",lambda m: print(f"browser:{m.type}:{m.text}",flush=True) if m.type=="error" else None)
  page.on("requestfailed",lambda r: print(f"requestfailed:{r.url}:{r.failure}",flush=True))
  page.set_default_timeout(10_000)
  print("content-e2e:login-load",flush=True); page.goto(BASE+"/login",wait_until="domcontentloaded"); print("content-e2e:login-fill",flush=True); page.get_by_label("账号").fill("e2e@app.test"); page.get_by_label("密码").fill("password"); print("content-e2e:login-submit",flush=True); page.locator('button[type="submit"]').click(); page.wait_for_url("**/dashboard**"); print("content-e2e:dashboard",flush=True)
  print("content-e2e:articles-load",flush=True); page.evaluate("window.history.pushState({}, '', '/system/content/articles'); window.dispatchEvent(new PopStateEvent('popstate'))"); page.get_by_role("heading",name="文章管理").wait_for(); page.get_by_text("欢迎使用").wait_for(); print(f"content-e2e:articles-url:{page.url}",flush=True); assert "sort=published_at%3Adesc" in page.url; axe(page,"zh-CN.1440",evidence); page.screenshot(path=OUT/"1440x900-light.png",full_page=True)
  print("content-e2e:article-drawer",flush=True); page.locator("button:has-text('编 辑')").click(); page.get_by_role("dialog").wait_for(); page.screenshot(path=OUT/"drawer-zh-CN-1440.png",full_page=True); page.keyboard.press("Escape")
  print("content-e2e:categories-load",flush=True); page.get_by_role("tab",name="分类").click(); page.get_by_role("heading",name="分类管理").wait_for(); page.get_by_text("新闻").wait_for(); axe(page,"zh-CN.categories.1440",evidence)
  print("content-e2e:english-mobile",flush=True); page.get_by_label("显示语言").click(); page.get_by_role("menuitem",name="English").click(); page.get_by_role("heading",name="Category management").wait_for(); page.get_by_role("heading",name="Category management").click(); page.wait_for_timeout(200); page.set_viewport_size({"width":375,"height":812}); axe(page,"en-US.categories.375",evidence); page.screenshot(path=OUT/"375x812-en-US.png",full_page=True)
  (OUT/"axe-results.json").write_text(json.dumps(evidence,indent=2),encoding="utf-8"); b.close(); print("content-e2e:done",flush=True)
if __name__=="__main__": main()
