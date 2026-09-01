// Run against an isolated seeded database; no production data or real contact details.
// AK_FEEDBACK_E2E_CREDENTIALS is a private JSON fixture file created by the test harness.
import fs from 'node:fs/promises'
import assert from 'node:assert/strict'
import crypto from 'node:crypto'
const { chromium } = await import(process.env.AK_PLAYWRIGHT_MODULE || 'playwright')
const c = JSON.parse(await fs.readFile(process.env.AK_FEEDBACK_E2E_CREDENTIALS, 'utf8'))
const api = process.env.AK_E2E_API_URL || 'http://localhost:8082'
const ui = process.env.AK_E2E_BASE_URL || 'http://localhost:4175'
const output = new URL('../../../output/playwright/help-feedback/', import.meta.url)
await fs.mkdir(output, { recursive: true })
const evidence = { checks: [], screenshots: [] }
function passed(name) { evidence.checks.push(name); console.log('PASS '+name) }
async function mobile(path, body, key, token) {
  return fetch(api+'/api/v1'+path, { method: body ? 'POST' : 'GET', headers: { 'Content-Type':'application/json', 'Accept-Language':'zh-CN', 'X-AppID':c.app_id,'X-AK-Device-Key':crypto.randomUUID(), ...(token?{Authorization:'Bearer '+token}:{}),...(key?{'Idempotency-Key':key}:{}) }, ...(body?{body:JSON.stringify(body)}:{}) })
}
const login = await mobile('/auth/login/password', {email:c.mobile_email,password:c.password})
assert.equal(login.status,200);const token=(await login.json()).data.access_token
const png=Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=','base64')
const upload = await mobile('/me/feedback-uploads',{original_name:'test.png',media_type:'image/png',size_bytes:png.length},null,token)
assert.equal(upload.status,201);const session=(await upload.json()).data
const form=new FormData();form.set('file',new Blob([png],{type:'image/png'}),'test.png')
const image=await fetch(api+'/api/v1'+session.upload_url,{method:'POST',headers:{Authorization:'Bearer '+token,'X-AppID':c.app_id},body:form})
assert.equal(image.status,200);const fileID=(await image.json()).data.file_id
const key=crypto.randomUUID(),marker='Feedback E2E '+key.slice(0,8)
const body={description:marker+'\n无法在帮助页面找到设置说明。 Please keep the original description and all replies. '+ '长文本测试 Long text. '.repeat(12),contact:'qa@example.test',platform:'ios',app_version:'0.1.0',file_ids:[fileID]}
const submitted=await mobile('/me/feedbacks',body,key,token);assert.equal(submitted.status,201);const item=(await submitted.json()).data
const replay=await mobile('/me/feedbacks',body,key,token);assert.equal((await replay.json()).data.id,item.id)
passed('image upload and idempotent mobile submission')
const unauth=await fetch(api+'/api/v1/me/feedbacks/'+item.id+'/attachments/'+fileID+'/content',{headers:{'X-AppID':c.app_id}});assert.equal(unauth.status,401)
passed('anonymous private image access denied')
const browser=await chromium.launch({headless:true});const context=await browser.newContext({reducedMotion:'reduce',viewport:{width:1440,height:1000},locale:'zh-CN'});const page=await context.newPage();page.setDefaultTimeout(15000)
try {
  await context.addInitScript(({tenant,app})=>localStorage.setItem('ak.admin.app-selection.v1',JSON.stringify({[tenant]:app})),{tenant:c.tenant_id,app:c.app_id})
  await page.goto(ui+'/login')
  await page.getByLabel(/^(账号|Account)$/).fill(c.email)
  await page.getByLabel(/^(密码|Password)$/).fill(c.password)
  await page.getByRole('button',{name:/^(登\s*录|Sign in)$/}).click()
  await page.waitForURL(/\/dashboard(?:\?|$)/);console.log('Admin login ready')
  await page.getByText(/^(问题反馈|Feedback)$/).last().click();console.log('Feedback route opened')
  await page.getByPlaceholder(/搜索问题描述|Search descriptions/).fill(marker)
  await page.getByPlaceholder(/搜索问题描述|Search descriptions/).press('Enter')
  const row=page.locator('tr').filter({hasText:marker})
  await row.getByRole('button',{name:/查\s*看|View/}).click()
  await page.getByRole('dialog').waitFor()
  const field=page.getByRole('textbox',{name:/^(回复内容|Reply)$/})
  await field.fill('问题已确认，已补充操作说明。 The help content has been updated.')
  // A simulated transport failure must keep the in-memory reply. It is labelled separately from real backend checks.
  const replyPath='**/feedbacks/'+item.id+'/replies'
  await page.route(replyPath,route=>route.abort('failed'),{times:1})
  await page.getByRole('button',{name:/发送回复|Send reply/}).click()
  await page.getByRole('alert').filter({hasText:/失败|failed/i}).waitFor()
  assert.match(await field.inputValue(),/The help content/)
  passed('admin failed reply retains draft (injected transport failure)')
  await page.getByRole('button',{name:/发送回复|Send reply/}).click()
  await page.getByText('问题已确认，已补充操作说明。 The help content has been updated.',{exact:true}).waitFor()
  await page.getByRole('button',{name:/标记为已解决|Mark as Resolved/}).click()
  await page.getByRole('dialog').getByText(/^(已解决|Resolved)$/).first().waitFor()
  const result=await mobile('/me/feedbacks/'+item.id,null,null,token);const data=(await result.json()).data
  assert.equal(data.status,'resolved');assert.equal(data.replies.length,1)
  const privateImage=await mobile('/me/feedbacks/'+item.id+'/attachments/'+fileID+'/content',null,null,token);assert.equal(privateImage.status,200);assert.equal(privateImage.headers.get('cache-control'),'private, no-store')
  passed('admin reply and resolve -> mobile reads reply and private screenshot')
  for(const locale of ['zh-CN','en-US']) for(const theme of ['light','dark']) {
    await page.locator('.ant-drawer-close').click()
    if(await page.evaluate(()=>document.documentElement.lang)!==locale){
      await page.getByRole('button',{name:/语言|language/}).click()
      await page.getByRole('menuitem',{name:locale==='zh-CN'?'简体中文':'English',exact:true}).press('Enter')
    }
    assert.equal(await page.evaluate(()=>document.documentElement.lang),locale)
    await page.emulateMedia({colorScheme:theme})
    await page.locator('tr').filter({hasText:marker}).getByRole('button',{name:/查\s*看|View/}).click()
    await page.getByRole('dialog').getByText('qa@example.test',{exact:true}).waitFor()
    const filename='admin-detail-'+locale+'-system-'+theme+'.png'
    await page.screenshot({path:new URL(filename,output).pathname,fullPage:true});evidence.screenshots.push(filename)
    assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true)
  }
  passed('admin bilingual/system-theme long-text screenshots and page overflow')
  await page.setViewportSize({width:768,height:1024})
  assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true)
  await page.screenshot({path:new URL('admin-detail-en-US-768.png',output).pathname,fullPage:true});evidence.screenshots.push('admin-detail-en-US-768.png')
  passed('768px drawer has no horizontal page overflow')
} catch(error) {
  console.log('Inputs: '+JSON.stringify(await page.locator('input').evaluateAll(xs=>xs.map(x=>({type:x.type,role:x.getAttribute('role'),label:x.getAttribute('aria-label'),placeholder:x.placeholder})))));console.log('UI failure at '+page.url()+'; text: '+(await page.locator('body').innerText()).slice(-2200));
  await page.screenshot({path:new URL('admin-failure.png',output).pathname,fullPage:true});throw error
} finally {
  await fs.writeFile(new URL('evidence.json',output),JSON.stringify(evidence,null,2)+'\n')
  await browser.close()
}
