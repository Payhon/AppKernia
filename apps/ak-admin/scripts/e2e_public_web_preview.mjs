// Run against an isolated published H5 fixture. Never visit real stores/install APKs.
import fs from 'node:fs/promises';
import assert from 'node:assert/strict';
import {createRequire} from 'node:module';
const {chromium}=await import(process.env.AK_PLAYWRIGHT_MODULE || 'playwright');
const fixture=JSON.parse(await fs.readFile(process.env.AK_H5_E2E_FIXTURE,'utf8'));
const api=process.env.AK_E2E_API_URL || 'http://localhost:18080';
const ui=process.env.AK_E2E_BASE_URL || 'http://localhost:14173';
const base=`${api}/h5/apps/${fixture.app_id}`;
const output=new URL('../../../output/playwright/public-web-preview/',import.meta.url);
await fs.mkdir(output,{recursive:true});
const evidence={checks:[],screenshots:[],realDevice:false};
const pass=(name)=>{evidence.checks.push(name);console.log('PASS '+name)};
const browser=await chromium.launch({headless:true});
const require=createRequire(import.meta.url);
const axe=await fs.readFile(require.resolve('axe-core/axe.min.js'),'utf8');
const screenshot=async(page,name)=>{await page.screenshot({path:new URL(name,output).pathname});if(!evidence.screenshots.includes(name))evidence.screenshots.push(name)};
try {
 const context=await browser.newContext({viewport:{width:1440,height:1000},locale:'zh-CN',reducedMotion:'reduce'});
 await context.grantPermissions(['clipboard-read','clipboard-write'],{origin:ui});
 await context.addInitScript(({tenant,app})=>localStorage.setItem('ak.admin.app-selection.v1',JSON.stringify({[tenant]:app})),{tenant:fixture.tenant_id,app:fixture.app_id});
 const page=await context.newPage();page.setDefaultTimeout(15000);page.setDefaultNavigationTimeout(20000);
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto(ui+'/login');
 await page.getByLabel(/^(账号|Account)$/).fill(fixture.email);
 await page.getByLabel(/^(密码|Password)$/).fill(fixture.password);
 await page.getByRole('button',{name:/^(登\s*录|Sign in)$/}).click();
 await page.waitForURL(/\/dashboard(?:\?|$)/);
 pass('actual Admin login against isolated backend');
 const navigate=async(href)=>page.evaluate(async href=>{const {router}=await import('/src/app/router.tsx');const url=new URL(href,location.origin);await router.navigate({to:url.pathname,search:Object.fromEntries(url.searchParams)})},href);
 const modal=()=>page.locator('.ak-public-preview-modal');
 const ready=async()=>{await modal().waitFor();await modal().locator('.ak-phone-content[aria-busy=false]').waitFor();await modal().locator('iframe').contentFrame().locator('body[data-preview-origin]').waitFor();};
 const close=async()=>{await modal().getByRole('button',{name:/关闭预览|Close preview/}).click();await modal().waitFor({state:'hidden'});};
 const articleRow=()=>page.locator('tr').filter({has:page.getByText('/reading-notes',{exact:true})});
 const menuVisible=async()=>page.waitForFunction(()=>{const n=document.querySelector('.ak-article-actions-dropdown');if(!n)return false;const r=n.getBoundingClientRect();return r.left>=0 && r.top>=0 && r.right<=innerWidth && r.bottom<=innerHeight && Number(getComputedStyle(n).opacity)===1});
 const openArticle=async()=>{
  await navigate(`/app/content/articles?app_id=${fixture.app_id}`);
  console.log('Article route opened');await articleRow().locator('[data-article-actions]').click().catch(async error=>{console.log('Page errors: '+JSON.stringify(errors));console.log('Article surface: '+(await page.locator('body').innerText()).replaceAll(fixture.email,'fixture-user'));await screenshot(page,'debug-article.png');throw error});console.log('Article menu opened');
  const items=page.locator('.ant-dropdown:visible').getByRole('menuitem');await items.first().waitFor();
  await menuVisible();
  assert.equal(await items.count(),6);
  assert.equal(await items.locator('.anticon').count(),6);
  await screenshot(page,'article-menu-'+await page.locator('html').getAttribute('lang')+'.png');console.log('Menu rect '+JSON.stringify(await page.locator('.ant-dropdown:visible').boundingBox()));
  await page.getByRole('menuitem',{name:/手机预览|Mobile preview/}).press('Enter');console.log('Preview selected');await ready();console.log('Article ready');
 };
 const openDownload=async()=>{
  await navigate('/app/applications');
  await page.getByRole('button',{name:/H5 Preview App/}).click();
  await page.getByRole('menuitem',{name:/发行页配置|Public download page/}).press('Enter');
  await page.getByRole('dialog').getByRole('button',{name:/手机预览|Mobile preview/}).click();await ready();
 };
 const openDocument=async()=>{
  await navigate(`/app/content/pages?app_id=${fixture.app_id}`);
  await page.getByRole('button',{name:/手机预览|Mobile preview/}).first().click();await ready();
 };
 const setLocale=async(locale)=>{
  if(await page.locator('html').getAttribute('lang')===locale)return;
  await page.getByRole('button',{name:/语言|language/}).click();
  await page.getByRole('menuitem',{name:locale==='zh-CN'?'简体中文':'English',exact:true}).press('Enter');
  await page.locator(`html[lang="${locale}"]`).waitFor();
 };
 for(const locale of ['zh-CN','en-US']) {
  await setLocale(locale);
  for(const [kind,open] of [['article',openArticle],['download',openDownload],['document',openDocument]]) {
   await open();
   const frame=modal().locator('iframe').contentFrame();
   assert.equal(await frame.locator('html').getAttribute('lang'),locale);
   assert.equal(await frame.locator('body').getAttribute('data-preview-origin'),ui);
   for(const width of [1440,375]) {
    console.log('Layout '+kind+' '+locale+' '+width);await page.setViewportSize({width,height:width===375?812:1000});
    await page.waitForFunction(()=>{const r=document.querySelector('.ak-phone-device')?.getBoundingClientRect();return r && r.x>=0 && r.y>=0 && r.right<=innerWidth && r.bottom<=innerHeight}).catch(async error=>{console.log(await page.evaluate(()=>['.ak-public-preview-modal','.ak-public-preview-modal .ant-modal-container','.ak-public-preview-modal .ant-modal-header','.ak-public-preview-modal .ant-modal-body','.ak-public-preview-stage','.ak-phone-device'].map(s=>{const n=document.querySelector(s);return {s,rect:n?.getBoundingClientRect().toJSON(),height:n?.clientHeight,width:n?.clientWidth,style:n?[getComputedStyle(n).height,getComputedStyle(n).boxSizing,getComputedStyle(n).flex,getComputedStyle(n).display,getComputedStyle(n).minHeight,getComputedStyle(n).padding,getComputedStyle(n).overflow]:[]}})));await screenshot(page,'debug-layout.png');throw error});
    const shell=await modal().locator('.ak-phone-device').boundingBox();const viewport=page.viewportSize();
    await screenshot(page,`${kind}-${locale}-${width}.png`);
    assert(shell && shell.x>=0 && shell.y>=0 && shell.x+shell.width<=viewport.width && shell.y+shell.height<=viewport.height,JSON.stringify({shell,viewport}));
    const island=await modal().locator('.ak-phone-island').boundingBox();const content=await modal().locator('iframe').boundingBox();
    assert(island.y+island.height<=content.y);
    assert.equal(await frame.locator('html').evaluate(node=>node.scrollWidth<=node.clientWidth),true);
   }
   await page.setViewportSize({width:1440,height:1000});
   // Analyze dialog only: Drawer remains open underneath the download modal.
   await page.evaluate(axe);
   const result=await page.evaluate(async()=>await axe.run(document.querySelector('.ak-public-preview-modal'),{runOnly:{type:'tag',values:['wcag2a','wcag2aa']}}));
   assert.deepEqual(result.violations.map(v=>({id:v.id,nodes:v.nodes.map(n=>n.target)})),[]);
   if(kind==='article') {
    await frame.locator('body').evaluate(node=>window.scrollTo(0,node.scrollHeight));
    await frame.locator('body').press('Escape');await modal().waitFor({state:'hidden'});
    await page.waitForFunction(()=>document.activeElement?.hasAttribute('data-article-actions'));
   } else await close();
   if(kind==='download') await page.locator('.ant-drawer-close').click();
  }
  await navigate(`/app/content/articles?app_id=${fixture.app_id}`);await page.setViewportSize({width:375,height:812});await articleRow().locator('[data-article-actions]').click();await menuVisible();await screenshot(page,`article-menu-${locale}-375.png`);await page.keyboard.press('Escape');await page.setViewportSize({width:1440,height:1000});
  pass(`${locale}: all three entrypoints, 1440/375px screenshots, sizing, safe area, axe and Escape focus return`);
 }
 await openArticle();
 let frame=modal().locator('iframe').contentFrame();
 await modal().getByRole('button',{name:'Copy public link',exact:true}).click();
 assert.equal(await page.evaluate(()=>navigator.clipboard.readText()),`${base}/articles/reading-notes?lang=en-US`);
 const oldFrame=await modal().locator('iframe').elementHandle();
 await modal().getByRole('button',{name:'Refresh preview',exact:true}).click();await ready();
 assert.equal(await oldFrame.evaluate(node=>node.isConnected),false);
 assert.equal(await modal().locator('iframe').getAttribute('src'),`${base}/articles/reading-notes?lang=en-US`);
 await page.evaluate(()=>{navigator.clipboard.writeText=async()=>{throw new Error('injected clipboard denial')}});
 await modal().getByRole('button',{name:'Copy public link',exact:true}).click();
 await modal().getByRole('textbox',{name:'Copy public link',exact:true}).waitFor();
 assert.equal(await modal().getByRole('textbox').inputValue(),`${base}/articles/reading-notes?lang=en-US`);
 await screenshot(page,'copy-denied-en-US.png');
 pass('actual clipboard copy, frame replacement on refresh, injected clipboard denial fallback');
 await close();
 await openArticle();await navigate(`/app/content/articles?app_id=${fixture.switch_app_id}`);await modal().waitFor({state:'hidden'});assert.equal(await page.locator('.ak-phone-content iframe').count(),0);
 pass('changing the App context destroys the active iframe before any new content');
 await openDownload();frame=modal().locator('iframe').contentFrame();
 assert.equal(await frame.locator('#platform-filter [aria-pressed=true]').getAttribute('data-platform'),'all');
 await frame.locator('[data-platform=ios]').click();
 assert.equal(await frame.locator('#platform-filter [aria-pressed=true]').getAttribute('data-platform'),'ios');
 const store=frame.locator('.download-options a').first();const storeURL=await store.getAttribute('href');
 await context.route(storeURL,route=>route.fulfill({contentType:'text/html',body:'<!doctype html><title>Controlled store fixture</title>'}));
 const next=context.waitForEvent('page');await store.click();const popup=await next;await popup.waitForLoadState();
 assert.equal(popup.url(),storeURL);assert.equal(await popup.evaluate(()=>window.opener),null);await popup.close();
 assert.equal(await modal().locator('iframe').getAttribute('src'),`${base}/download?lang=en-US`);
 // APK target behavior is tested with a labelled inserted link, not an installation.
 await frame.locator('body').evaluate((body,href)=>{const a=document.createElement('a');a.href=href;a.textContent='Controlled APK fixture';a.id='e2e-apk';body.append(a)},base+'/apk');
 await context.route(base+'/apk',route=>route.fulfill({contentType:'text/html',body:'<!doctype html><title>Controlled APK target</title>'}));
 const apkNext=context.waitForEvent('page');await frame.locator('#e2e-apk').click();const apkPopup=await apkNext;await apkPopup.waitForLoadState();assert.equal(await apkPopup.evaluate(()=>window.opener),null);await apkPopup.close();
 await frame.locator('.legal-links a').first().click();await modal().locator('iframe').contentFrame().locator('body[data-preview-kind=document]').waitFor();await ready();
 assert.equal(await modal().locator('iframe').contentFrame().locator('body').getAttribute('data-preview-kind'),'document');
 pass('real browser platform retained; manual iOS selection; controlled store/APK new contexts with no opener; internal document navigation');
 await close();await page.locator('.ant-drawer-close').click();
 await navigate(`/app/content/articles?app_id=${fixture.app_id}`);await articleRow().locator('[data-article-actions]').click();
 await menuVisible();await screenshot(page,'article-menu-en-US.png');
 const publicTab=context.waitForEvent('page');await page.getByRole('menuitem',{name:'View public page',exact:true}).press('Enter');const publicPage=await publicTab;await publicPage.waitForLoadState();assert.equal(publicPage.url(),`${base}/articles/reading-notes?lang=en-US`);assert.equal(await publicPage.evaluate(()=>window.opener),null);await publicPage.close();
 pass('article public view supports keyboard activation and opens without an opener');
 const draft=page.locator('tr').filter({has:page.getByText('/draft-note',{exact:true})});await draft.locator('[data-article-actions]').click();await page.locator('.ant-dropdown:visible').getByRole('menuitem').first().waitFor();
 assert.equal(await page.getByRole('menuitem',{name:'Mobile preview',exact:true}).count(),0);
 await page.getByRole('menuitem',{name:'Delete',exact:true}).click().catch(async error=>{console.log('Draft menu: '+await page.locator('.ant-dropdown:visible').innerText());console.log(await page.locator('.ant-dropdown:visible').evaluate(n=>({rect:n.getBoundingClientRect().toJSON(),style:n.getAttribute('style'),viewport:{width:innerWidth,height:innerHeight},body:document.body.getBoundingClientRect().toJSON(),parent:n.parentElement.className})));await screenshot(page,'debug-menu.png');throw error});
 await page.getByRole('dialog').getByRole('button',{name:/Cancel/}).click();
 pass('draft has no public actions and deletion still requires confirmation');
 // Exercise the real server error page and a controlled missing handshake in both languages.
 const html=await (await fetch(base+'/download')).text();const previewScript=html.match(/<script type="module" src="([^"]+)"><\/script>/g).at(-1).match(/src="([^"]+)"/)[1];
 for(const locale of ['zh-CN','en-US']) {
  await setLocale(locale);
  await page.route(`${base}/articles/reading-notes*`,async route=>route.fulfill({response:await route.fetch({url:base+'/articles/missing-preview?lang='+locale})}));
  await openArticle();assert.equal(await modal().locator('iframe').contentFrame().locator('body').getAttribute('data-preview-kind'),'error');await screenshot(page,`unavailable-${locale}.png`);await close();await page.unroute(`${base}/articles/reading-notes*`);
  await page.route(api+previewScript,route=>route.fulfill({contentType:'text/javascript',body:'// Injected absent handshake'}));
  await navigate(`/app/content/articles?app_id=${fixture.app_id}`);await articleRow().locator('[data-article-actions]').click();await menuVisible();await page.getByRole('menuitem',{name:/手机预览|Mobile preview/}).click();
  await modal().getByRole('alert').waitFor({timeout:14000});await screenshot(page,`timeout-${locale}.png`);await close();await page.unroute(api+previewScript);
 }
 pass('bilingual actual server unavailable pages and injected missing-handshake timeout fallback');
 const denied=await context.newPage();
 await denied.route('http://127.0.0.1:19999/',route=>route.fulfill({contentType:'text/html',body:`<!doctype html><title>Untrusted parent fixture</title><iframe src="${base}/download"></iframe>`}));
 await denied.goto('http://127.0.0.1:19999/');await denied.waitForLoadState('networkidle');assert.equal(await denied.locator('iframe').contentFrame().locator('body[data-preview-origin]').count(),0);await denied.close();
 pass('browser rejects embedding from an untrusted parent origin');
 await openArticle();await page.emulateMedia({colorScheme:'dark'});await screenshot(page,'article-en-US-dark.png');await page.emulateMedia({colorScheme:'light'});
 await page.evaluate(()=>{document.documentElement.style.zoom='2';window.dispatchEvent(new Event('resize'))});await page.waitForTimeout(200);await screenshot(page,'article-en-US-200-percent.png');
 assert.equal(await page.evaluate(()=>{const r=document.querySelector('.ak-public-preview-modal').getBoundingClientRect();return r.left>=0 && r.right<=innerWidth && r.top>=0 && r.bottom<=innerHeight}),true);
 await page.evaluate(()=>{document.documentElement.style.zoom='';window.dispatchEvent(new Event('resize'))});
 await modal().locator('iframe').contentFrame().locator('html').evaluate(node=>node.style.zoom='2');
 assert.equal(await modal().locator('iframe').contentFrame().locator('html').evaluate(node=>node.scrollWidth<=node.clientWidth),true);
 await screenshot(page,'article-en-US-text-200-percent.png');await close();
 pass('dark H5 preview, 200% CSS zoom within modal bounds and 200% article text');
 assert.deepEqual(errors,[]);await context.close();
} finally {
 await fs.writeFile(new URL('evidence.json',output),JSON.stringify(evidence,null,2)+'\n');await browser.close();
}
