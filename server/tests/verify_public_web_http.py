"""Destructive checks ONLY against the dedicated H5 fixture database, never a live tenant.
The fixture APK is a ZIP test payload, not an installable application.
"""
import io,json,os,subprocess,time,uuid,zipfile
from pathlib import Path
from urllib.request import Request,urlopen,build_opener,HTTPRedirectHandler
from urllib.error import HTTPError
from urllib.parse import urlparse,parse_qs,urlencode

fixture=json.loads(Path(os.environ['AK_H5_E2E_FIXTURE']).read_text())
app,tenant=(str(uuid.UUID(fixture[k])) for k in ('app_id','tenant_id'))
database=os.environ.get('AK_H5_DATABASE','ak_h5_verify')
if not database.startswith('ak_h5_'):raise RuntimeError('H5 tests require an isolated ak_h5_* database')
container=os.environ.get('AK_H5_PG_CONTAINER','appkernia-news-demo-postgres-1')
base=os.environ.get('AK_E2E_API_URL','http://localhost:18080')
if urlparse(base).hostname not in ('localhost','127.0.0.1'):raise RuntimeError('Only loopback test servers are allowed')

def sql(value):
 r=subprocess.run(['docker','exec','-i',container,'psql','-U','appkernia','-d',database,'-At','-v','ON_ERROR_STOP=1'],input=value,text=True,capture_output=True)
 if r.returncode:raise RuntimeError(r.stderr)
 return r.stdout.strip()
assert sql(f"SELECT code FROM iam.tenants WHERE id='{tenant}'")=='h5-e2e'
class NoRedirect(HTTPRedirectHandler):
 def redirect_request(self,*args,**kwargs):return None
opener=build_opener(NoRedirect)
def get(path,headers=None):
 try:r=opener.open(Request(base+path,headers=headers or {}),timeout=15)
 except HTTPError as e:r=e
 with r:return r.status,r.headers,r.read()
prefix='/h5/apps/'+app
checks=[]
def passed(s):checks.append(s);print('PASS '+s)
file,release=str(uuid.uuid4()),str(uuid.uuid4())
archive=io.BytesIO()
with zipfile.ZipFile(archive,'w') as z:z.writestr('AndroidManifest.xml','TEST FIXTURE ONLY')
payload=archive.getvalue();key=tenant+'/h5-fixture/'+file+'.apk'
object_root=Path(os.environ.get('AK_H5_OBJECT_DIR','/tmp/ak-h5-objects'));path=object_root/key;path.parent.mkdir(parents=True,exist_ok=True);path.write_bytes(payload)
try:
 sql(f"""BEGIN;
 INSERT INTO storage.files(id,tenant_id,provider,bucket_name,object_key,original_name,media_type,extension,size_bytes,status,scan_status)
 VALUES('{file}','{tenant}','local','appkernia-local','{key}','fixture.apk','application/vnd.android.package-archive','apk',{len(payload)},'ready','clean');
 INSERT INTO sys.mobile_releases(id,tenant_id,app_id,platform,current_version,minimum_version,release_notes,active,version,package_type,package_file_id,ever_published_at)
 VALUES('{release}','{tenant}','{app}','android','1.0.0','1.0.0','{{}}',true,'1.0.0','native_app','{file}',now());
 INSERT INTO sys.mobile_release_targets(tenant_id,app_id,release_id,package_type,platform) VALUES('{tenant}','{app}','{release}','native_app','android');
 INSERT INTO sys.mobile_release_publications(tenant_id,app_id,release_id,package_type,platform) VALUES('{tenant}','{app}','{release}','native_app','android');
 UPDATE app.application_public_web_configs SET apk_enabled=true WHERE app_id='{app}';COMMIT;""")
 status,headers,_=get(prefix+'/apk?url=https://evil.example');assert status==302,status
 target=headers['Location'];assert target.startswith(prefix+'/packages/');assert headers['Cache-Control']=='no-store'
 status,headers,data=get(target);assert status==200,status;assert data==payload;assert headers['Cache-Control']=='no-store'
 passed('signed APK resource returns fixture bytes; arbitrary url parameter cannot redirect')
 query=parse_qs(urlparse(target).query);query['expires']=[str(int(time.time())-1)]
 assert get(urlparse(target).path+'?'+urlencode(query,doseq=True))[0]==404
 assert get(target+'x')[0]==404
 assert get(target.replace(app,str(uuid.uuid4())))[0]==404
 old='/api/v1/public/app-version/download/'+release+'/'+file+'?'+urlparse(target).query
 assert get(old,{'X-AppID':app})[0] != 200
 passed('expired/tampered/cross-App and mobile-route signature replay rejected')
 sql(f"UPDATE storage.files SET scan_status='infected' WHERE id='{file}';")
 assert get(target)[0]==404;assert get(prefix+'/apk')[0]==404
 sql(f"UPDATE storage.files SET scan_status='clean' WHERE id='{file}';UPDATE app.application_public_web_configs SET apk_enabled=false WHERE app_id='{app}';")
 assert get(target)[0]==404;assert get(prefix+'/apk')[0]==404
 sql(f"UPDATE app.application_public_web_configs SET apk_enabled=true WHERE app_id='{app}';DELETE FROM sys.mobile_release_publications WHERE release_id='{release}';")
 assert get(target)[0]==404;assert get(prefix+'/apk')[0]==404
 passed('scan rejection, APK switch and publication withdrawal invalidate subsequent requests')
 article=str(uuid.UUID(fixture['reading-notes_id']));image=str(uuid.UUID(fixture['article_file_id']))
 sql(f"UPDATE content.articles SET status='archived' WHERE id='{article}';")
 assert get(prefix+'/articles/reading-notes')[0]==404;assert get(prefix+'/assets/'+image)[0]==404
 sql(f"UPDATE content.articles SET status='published' WHERE id='{article}';UPDATE app.applications SET status='disabled' WHERE id='{app}';")
 assert get(prefix+'/articles/reading-notes')[0]==404;assert get(prefix+'/download')[0]==404;assert get(prefix+'/pages/privacy-policy')[0]==404
 passed('article withdrawal removes article/image access; inactive App blocks all pages')
 sql(f"UPDATE app.applications SET status='active' WHERE id='{app}';")
 before=sql(f"SELECT count(*) FROM app.legal_consents WHERE app_id='{app}'")
 assert get(prefix+'/pages/privacy-policy')[0]==200
 assert before==sql(f"SELECT count(*) FROM app.legal_consents WHERE app_id='{app}'")
 code,headers,body=get(prefix+'/download?lang=en-US',{'Host':'evil.example','X-Forwarded-Host':'evil.example','Accept-Language':'zh-CN'})
 assert code==200;assert headers['Content-Language']=='en-US';assert b'evil.example' not in body
 passed('policy reading does not consent; trusted canonical ignores Host and explicit language wins')
 code,_,body=get('/api/v1/public/content/items/reading-notes',{'X-AppID':app,'Accept-Language':'en-US'})
 assert code==200;assert json.loads(body)['data']['share_url']==base+prefix+'/articles/reading-notes?lang=en-US'
 passed('Mobile public detail returns the trusted localized H5 share URL')
finally:
 sql(f"""DELETE FROM sys.mobile_release_publications WHERE release_id='{release}';DELETE FROM sys.mobile_releases WHERE id='{release}';DELETE FROM storage.files WHERE id='{file}';
 UPDATE app.application_public_web_configs SET apk_enabled=false WHERE app_id='{app}';UPDATE app.applications SET status='active' WHERE id='{app}';
 UPDATE content.articles SET status='published' WHERE app_id='{app}' AND slug='reading-notes';""")
 path.unlink(missing_ok=True)
 Path('output/playwright/public-web/http-evidence.json').write_text(json.dumps({'checks':checks,'installableAPK':False},ensure_ascii=False,indent=2)+'\n')
