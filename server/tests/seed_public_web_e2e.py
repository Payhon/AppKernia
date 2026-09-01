"""Create or reuse a local H5 browser fixture after core seeding in an isolated database.
Uses AK_H5_PASSWORD_FILE and writes a private AK_H5_E2E_FIXTURE JSON, never logs credentials.
"""
import subprocess,json,uuid,os,struct,zlib
from pathlib import Path

database=os.environ.get('AK_H5_DATABASE','ak_h5_verify')
if not database.startswith('ak_h5_'):raise RuntimeError('Only isolated ak_h5_* databases are allowed')
container=os.environ.get('AK_H5_PG_CONTAINER','appkernia-news-demo-postgres-1')
fixture_path=Path(os.environ.get('AK_H5_E2E_FIXTURE','/tmp/ak-h5-fixture.json'))
object_root=Path(os.environ.get('AK_H5_OBJECT_DIR','/tmp/ak-h5-objects'))
def query(sql):
 p=subprocess.run(['docker','exec','-i',container,'psql','-U','appkernia','-d',database,'-At','-v','ON_ERROR_STOP=1'],input=sql,text=True,capture_output=True)
 if p.returncode: raise RuntimeError(p.stderr)
 return p.stdout.strip()
def q(s):return "'"+str(s).replace("'","''")+"'"
ids=json.loads(query("SELECT json_build_object('tenant_id',t.id,'app_id',a.id,'user_id',u.id) FROM iam.tenants t JOIN app.applications a ON a.tenant_id=t.id AND a.is_default JOIN iam.users u ON u.email='h5-admin@example.test' WHERE t.code='h5-e2e';"))
a,t,u=ids['app_id'],ids['tenant_id'],ids['user_id']
def add_switch_app(values):
 # Same-tenant context change fixture, never an invalid or cross-tenant App ID.
 query(f"INSERT INTO app.applications(tenant_id,code,name,owner_tenant_id,creator_user_id) VALUES('{t}','h5-preview-switch','Preview switch fixture','{t}','{u}') ON CONFLICT (tenant_id,code) DO NOTHING;")
 values['switch_app_id']=query(f"SELECT id FROM app.applications WHERE tenant_id='{t}' AND code='h5-preview-switch';")
def write_fixture(values):
 fd=os.open(fixture_path,os.O_CREAT|os.O_TRUNC|os.O_WRONLY,0o600)
 try:
  os.fchmod(fd,0o600)
  os.write(fd,json.dumps(values).encode())
 finally: os.close(fd)
if fixture_path.exists():
 old=json.loads(fixture_path.read_text())
 if old.get('app_id')==a and query(f"SELECT count(*) FROM app.application_public_web_configs WHERE app_id='{a}'")=='1':
  add_switch_app(old);write_fixture(old)
  print('existing isolated fixture reused');raise SystemExit(0)
 raise RuntimeError('Fixture file belongs to another database; choose another output path')
ids['email']='h5-admin@example.test';ids['password']=Path(os.environ.get('AK_H5_PASSWORD_FILE','/tmp/ak-h5-password')).read_text()
# Deterministic fixture images made from PNG primitives, no external or private images.
def png(w,h):
 data=b''.join(b'\0'+b''.join(bytes((235-int(y/h*25),240-int(x/w*12),250,255)) for x in range(w)) for y in range(h))
 def chunk(typ,data):return struct.pack('>I',len(data))+typ+data+struct.pack('>I',zlib.crc32(typ+data)&0xffffffff)
 return b'\x89PNG\r\n\x1a\n'+chunk(b'IHDR',struct.pack('>IIBBBBB',w,h,8,6,0,0,0))+chunk(b'IDAT',zlib.compress(data))+chunk(b'IEND',b'')
queries=[]
for kind,w,h in [('icon',128,128),('screenshot',360,720),('article',900,480)]:
 fid=str(uuid.uuid4());ids[kind+'_file_id']=fid;key=t+'/h5-fixture/'+fid+'.png';body=png(w,h);path=object_root/key;path.parent.mkdir(parents=True,exist_ok=True);path.write_bytes(body)
 queries.append(f"INSERT INTO storage.files(id,tenant_id,provider,bucket_name,object_key,original_name,media_type,extension,size_bytes,status,scan_status) VALUES('{fid}','{t}','local','appkernia-local','{key}','fixture.png','image/png','png',{len(body)},'ready','clean');")
queries.append(f"UPDATE app.applications SET name='H5 Preview App',icon_file_id='{ids['icon_file_id']}' WHERE id='{a}';")
queries.append(f"INSERT INTO app.application_assets(tenant_id,app_id,asset_type,file_id,position) VALUES('{t}','{a}','screenshot','{ids['screenshot_file_id']}',0);")
queries.append(f"INSERT INTO app.application_public_web_configs(tenant_id,app_id,enabled,apk_enabled,promotion_enabled) VALUES('{t}','{a}',true,false,true);")
for loc,name,intro,promo_title,promo_description,promo_button in [
 ('zh-CN','拾光笔记','记录生活中的每一个灵感。\n轻松整理笔记，让重要的事触手可及。','在拾光笔记中继续阅读','保存灵感，随时回到重要的内容。','下载应用'),
 ('en-US','Daylight Notes','A quiet place for your ideas.\nCapture moments, organize notes, and keep what matters close.','Keep reading in Daylight Notes','Save ideas and return to what matters whenever you like.','Get the app'),
]:
 queries.append(f"INSERT INTO app.application_public_web_translations(tenant_id,app_id,locale,name,introduction,promotion_title,promotion_description,promotion_button_label) VALUES('{t}','{a}',{q(loc)},{q(name)},{q(intro)},{q(promo_title)},{q(promo_description)},{q(promo_button)});")
for platform,name,url in [('ios','App Store','https://apps.apple.com/'),('android','Google Play','https://play.google.com/store/apps'),('harmony','AppGallery','https://appgallery.huawei.com/')]:
 queries.append(f"INSERT INTO app.application_store_listings(tenant_id,app_id,name,scheme,enabled,priority,platform,web_url) VALUES('{t}','{a}',{q(name)},'fixture://',true,10,{q(platform)},{q(url)});")
for slug,status in [('reading-notes','published'),('draft-note','draft')]:
 aid=str(uuid.uuid4());ids[slug+'_id']=aid
 queries.append(f"INSERT INTO content.articles(id,tenant_id,app_id,slug,status,published_at,cover_file_id) VALUES('{aid}','{t}','{a}','{slug}','{status}',now(),'{ids['article_file_id']}');")
 for loc,title,summary in [('zh-CN','给生活留一点记录的空间','从一段文字开始，保存日常的灵感与发现。'),('en-US','Make a little room for everyday stories','Start with a few words. Keep your discoveries and small moments close.')]:
  text=('记录是一种温柔的整理。无需华丽的形式，让每一个想法都能被认真保存。' if loc=='zh-CN' else 'A note can be a gentle way to make sense of the day. Give each idea enough space to become something meaningful.')
  body=text+'\n\n## '+('从这里开始' if loc=='zh-CN' else 'A thoughtful beginning')+'\n\n'+text+'\n\n> '+text+'\n\n- '+text+'\n- '+text+'\n\n![Landscape](/api/v1/public/content/assets/'+ids['article_file_id']+')\n\n| A | B |\n| --- | --- |\n| One | Two |\n\n```go\nfmt.Println("'+('你好' if loc=='zh-CN' else 'Hello')+'")\n```\n\n'+('LongUnbrokenWord'*25)+'\n\n[Link](https://example.test/)\n\n<script>alert(1)</script>\n\n<img src=x onerror=alert(1) hx-get="/admin-api/v1/users">'
  queries.append(f"INSERT INTO content.article_translations(article_id,locale,title,summary,body_format,body) VALUES('{aid}',{q(loc)},{q(title)},{q(summary)},'markdown',{q(json.dumps(body,ensure_ascii=False))}::jsonb);")
page=query(f"SELECT id FROM content.pages WHERE app_id='{a}' AND slug='privacy-policy';")
if not page:
 page=str(uuid.uuid4());queries.append(f"INSERT INTO content.pages(id,app_id,tenant_id,slug,page_type) VALUES('{page}','{a}','{t}','privacy-policy','privacy-policy');")
rev=str(uuid.uuid4());queries.append(f"INSERT INTO content.page_revisions(id,page_id,app_id,tenant_id,revision_number,content_hash,status,published_at) VALUES('{rev}','{page}','{a}','{t}',1,decode(repeat('ab',32),'hex'),'published',now());")
for loc,title,body in [('zh-CN','隐私政策','## 关于这份说明\n\n这是浏览器验证用的示例内容，不是正式法律文本。\n\n我们重视您对个人信息的选择。'),('en-US','Privacy policy','## About this notice\n\nThis is a browser test fixture, not a legal document.\n\nYour choices about personal information matter.')]:
 queries.append(f"INSERT INTO content.page_revision_translations(revision_id,locale,title,body_format,body) VALUES('{rev}',{q(loc)},{q(title)},'markdown',{q(json.dumps(body,ensure_ascii=False))}::jsonb);")
queries.append(f"UPDATE content.pages SET current_revision_id='{rev}',status='published' WHERE id='{page}';")
query('BEGIN;\n'+'\n'.join(queries)+'\nCOMMIT;')
add_switch_app(ids);write_fixture(ids)
print('isolated fixture ready')
