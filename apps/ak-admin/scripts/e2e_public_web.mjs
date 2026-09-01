// Uses only an isolated fixture. Does not visit external stores or install packages.
import fs from 'node:fs/promises';
import assert from 'node:assert/strict';
import {createRequire} from 'node:module';
const {chromium}=await import(process.env.AK_PLAYWRIGHT_MODULE || 'playwright');
const c=JSON.parse(await fs.readFile(process.env.AK_H5_E2E_FIXTURE,'utf8'));
const api=process.env.AK_E2E_API_URL||'http://localhost:18080';
const ui=process.env.AK_E2E_BASE_URL||'http://localhost:14173';
const base=api+'/h5/apps/'+c.app_id;
const output=new URL('../../../output/playwright/public-web/',import.meta.url);await fs.mkdir(output,{recursive:true});
const evidence={checks:[],screenshots:[],realDevice:false};
function passed(x){evidence.checks.push(x);console.log('PASS '+x)}
const browser=await chromium.launch({headless:true});
const require=createRequire(import.meta.url);const axe=await fs.readFile(require.resolve('axe-core/axe.min.js'),'utf8');
try{
 if (!process.env.AK_H5_ADMIN_ONLY) {
 for(const route of ['/articles/reading-notes','/pages/privacy-policy','/download']){
  const a=await fetch(base+route,{headers:{'Accept-Language':'en-US','User-Agent':'iPhone'}});
  assert.equal(a.status,200);assert.equal(a.headers.get('content-language'),'en-US');assert.equal(a.headers.get('cache-control'),'no-cache');assert.match(a.headers.get('content-security-policy'),/script-src 'self'/);
  const html=await a.text();assert.doesNotMatch(html,/<script>alert|onerror=|hx-get=/);assert.match(html,/rel="canonical"/);
  const b=await fetch(base+route,{headers:{'Accept-Language':'en-US','User-Agent':'Android'}});assert.equal(await b.text(),html);
 }
 passed('SSR complete, language/security/cache headers and identical platform responses');
 for(const suffix of ['/articles/draft-note','/articles/not-found','/apk','/assets/'+crypto.randomUUID()])assert.equal((await fetch(base+suffix,{redirect:'manual'})).status,404);
 passed('draft, missing content, disabled APK and unrelated asset denied');
 const old=await fetch(api+'/s/reading-notes?app_id='+c.app_id);assert.equal(old.status,200);assert.match(await old.text(),new RegExp('/h5/apps/'+c.app_id+'/articles/reading-notes'));
 passed('legacy sharing renders same article with new canonical');
 for(const lang of ['zh-CN','en-US'])for(const theme of ['light','dark'])for(const width of [390,1440]){
  const ctx=await browser.newContext({viewport:{width,height:1000},locale:lang,colorScheme:theme,reducedMotion:'reduce'});const page=await ctx.newPage();
  for(const [kind,route] of [['article','/articles/reading-notes'],['download','/download'],['page','/pages/privacy-policy']]){
   const errors=[];page.on('pageerror',e=>errors.push(e.message));
   await page.goto(base+route+'?lang='+lang);await page.waitForLoadState('networkidle');
   assert.equal(await page.locator('html').getAttribute('lang'),lang);assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true);
   await page.locator('img').last().scrollIntoViewIfNeeded().catch(()=>{}); await page.waitForTimeout(150); const images=await page.locator('img').evaluateAll(xs=>xs.map(x=>({src:x.src,ok:x.complete&&x.naturalWidth>0}))); assert.equal(images.every(x=>x.ok),true,JSON.stringify(images));
   await page.evaluate(axe);const result=await page.evaluate(async()=>await axe.run(document,{runOnly:{type:'tag',values:['wcag2a','wcag2aa']}}));
   assert.deepEqual(result.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.target)})),[]);
   const name=`${kind}-${lang}-${theme}-${width}.png`;await page.screenshot({path:new URL(name,output).pathname,fullPage:true});evidence.screenshots.push(name);assert.deepEqual(errors,[]);
  }
  await ctx.close();
 }
 passed('three pages: both languages, light/dark, 390/1440px, image load, axe WCAG2A/AA and overflow');
 const nojs=await browser.newContext({javaScriptEnabled:false,viewport:{width:390,height:844}});const np=await nojs.newPage();await np.goto(base+'/download');assert.equal(await np.locator('[data-download-platform]').count(),3);assert.equal(await np.locator('#platform-filter').isVisible(),false);await np.goto(base+'/articles/reading-notes');assert.match(await np.locator('.prose').innerText(),/记录/);await nojs.close();passed('JavaScript disabled: content and all store links usable');
 for(const [ua,want] of [['Mozilla/5.0 (iPhone; CPU iPhone OS 18_0)','ios'],['Mozilla/5.0 (Linux; Android 15)','android'],['Mozilla/5.0 (OpenHarmony 5.0)','harmony'],['Mozilla/5.0 Desktop','all']]){
  const ctx=await browser.newContext({userAgent:ua});const p=await ctx.newPage();await p.goto(base+'/download');await p.waitForLoadState('networkidle');assert.equal(await p.locator('#platform-filter [aria-pressed=true]').getAttribute('data-platform'),want);assert.match(p.url(),/\/download$/);await p.locator('[data-platform=harmony]').click();assert.equal(await p.locator('.is-recommended').getAttribute('data-download-platform'),'harmony');assert.equal(await p.locator('[data-download-platform]:visible').count(),3);await ctx.close();
 }
 passed('UA simulation only: recommendation/manual override, all links retained, no automatic navigation');
 const ctx=await browser.newContext({userAgent:'Mozilla/5.0 (iPhone) MicroMessenger',viewport:{width:390,height:844}});const p=await ctx.newPage();await p.goto(base+'/download');assert.equal(await p.locator('#wechat-note').isVisible(),true);await p.goto(base+'/articles/reading-notes');await p.evaluate(()=>document.documentElement.style.zoom='2');assert.equal(await p.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true);await p.screenshot({path:new URL('article-200-percent.png',output).pathname,fullPage:true});evidence.screenshots.push('article-200-percent.png');await p.keyboard.press('Tab');assert.equal(await p.evaluate(()=>document.activeElement.className),'skip-link');await ctx.close();passed('WeChat hint simulation, 200% zoom and keyboard skip link');
 }
 // Admin uses the actual configured backend; secrets remain in the private fixture file.
 const admin=await browser.newContext({viewport:{width:1440,height:1000},locale:'zh-CN',reducedMotion:'reduce'});const ap=await admin.newPage();ap.setDefaultTimeout(15000);ap.setDefaultNavigationTimeout(20000);ap.on("pageerror",e=>console.log("Admin pageerror: "+e.message));ap.on('requestfailed',r=>console.log('REQUEST FAILED '+r.url()+' '+r.failure()?.errorText));
 await admin.addInitScript(({tenant,app})=>localStorage.setItem('ak.admin.app-selection.v1',JSON.stringify({[tenant]:app})),{tenant:c.tenant_id,app:c.app_id});
 await ap.goto(ui+'/login');console.log('Login loaded');await ap.getByLabel(/^(账号|Account)$/).fill(c.email);await ap.getByLabel(/^(密码|Password)$/).fill(c.password);await ap.getByRole('button',{name:/^(登\s*录|Sign in)$/}).click();await ap.waitForURL(/\/dashboard(?:\?|$)/).catch(async e=>{console.log((await ap.locator('body').innerText()).slice(-1600));throw e});
 await ap.getByText(/^(应用管理|Applications)$/).last().click();await ap.getByRole('button',{name:/H5 Preview App/}).waitFor();
 await ap.locator('.ant-table-content,.ant-table-body').evaluateAll(xs=>xs.forEach(x=>{x.scrollLeft=x.scrollWidth}));await ap.getByRole('button',{name:/H5 Preview App/}).click();await ap.getByRole('menuitem',{name:/发行页配置|Public download page/}).press('Enter');
 const dialog=ap.getByRole('dialog');await dialog.getByRole('switch').first().waitFor();


 for(const locale of ['zh-CN','en-US']){
  if(await ap.evaluate(()=>document.documentElement.lang)!==locale){await dialog.locator('.ant-drawer-close').click();await dialog.waitFor({state:'hidden'});await ap.getByRole('button',{name:/语言|language/}).click();await ap.getByRole('menuitem',{name:locale==='zh-CN'?'简体中文':'English',exact:true}).press('Enter');await ap.locator('.ant-table-content,.ant-table-body').evaluateAll(xs=>xs.forEach(x=>{x.scrollLeft=x.scrollWidth}));await ap.getByRole('button',{name:/H5 Preview App/}).click();await ap.getByRole('menuitem',{name:/发行页配置|Public download page/}).press('Enter')}
  const filename='admin-public-web-'+locale+'.png';await dialog.getByRole('switch').first().waitFor();await ap.screenshot({path:new URL(filename,output).pathname});evidence.screenshots.push(filename);await dialog.locator('.ant-drawer-body').evaluate(x=>{x.scrollTop=x.scrollHeight});const bottom='admin-public-web-'+locale+'-downloads.png';await ap.screenshot({path:new URL(bottom,output).pathname});evidence.screenshots.push(bottom);await dialog.locator('.ant-drawer-body').evaluate(x=>{x.scrollTop=0});
 }
 const editName=dialog.locator('input').first();const originalName=await editName.inputValue();await editName.fill(originalName+' E2E');
 // A labelled injected conflict checks draft retention; actual concurrent writers are tested in PostgreSQL integration tests.
 await ap.route('**/public-web-config',route=>{return route.fulfill({status:409,contentType:'application/json',body:JSON.stringify({error:{code:'APP.CONFLICT',message:'Conflict',message_key:'errors.common.unknown'},request_id:'e2e-injected'})})},{times:1});
 await dialog.getByRole('button',{name:/^(保\s*存|Save)$/}).click();await dialog.getByRole('alert').filter({hasText:/冲突|其他|已更新|changed|conflict|updated|refresh/i}).waitFor().catch(async e=>{console.log('Dialog: '+(await dialog.innerText()));throw e});assert.equal(await editName.inputValue(),originalName+' E2E');await editName.fill(originalName);
 passed('Admin retains unsaved fields on injected HTTP 409');
 // Actual save with the optimistic version supplied by the backend.
 await dialog.locator('button[type=submit]').click().catch(async e=>{console.log('Save failure: '+(await ap.locator('body').innerText()).slice(-2500));throw e});await dialog.getByRole('alert').filter({hasText:/已保存|saved/i}).waitFor();
 passed('Admin configuration opens in both languages and saves against real backend');
 await admin.close();passed('actual Admin login and App management route');
}finally{await fs.writeFile(new URL(process.env.AK_H5_ADMIN_ONLY?'admin-evidence.json':'evidence.json',output),JSON.stringify(evidence,null,2)+'\n');await browser.close()}
