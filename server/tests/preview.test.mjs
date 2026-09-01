import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
const source=await readFile(new URL('../resource/static/js/preview.js',import.meta.url),'utf8');
const {keepInPreview,installPreview}=await import('data:text/javascript;base64,'+Buffer.from(source).toString('base64'));
const app='00000000-0000-4000-8000-000000000001';
const current=new URL(`https://public.example/h5/apps/${app}/download`);
test('only documents in the same App remain inside the preview',()=>{
 for(const path of [`/h5/apps/${app}/download#downloads`,`/h5/apps/${app}/pages/privacy?lang=en-US`,`/s/article?app_id=${app}`]) assert.equal(keepInPreview(new URL(path,current),current),true);
 for(const path of [`/h5/apps/${app}/apk`,`/h5/apps/${app}/packages/a/b`,`/h5/apps/${app}/download?format=qr`,'/admin-api/v1/apps','https://store.example/',`/h5/apps/00000000-0000-4000-8000-000000000002/download`]) assert.equal(keepInPreview(new URL(path,current),current),false);
});
function fixture(kind='download') {
 const sent=[];const win=new EventTarget();win.location={href:current.href};win.parent={postMessage:(data,origin)=>sent.push({data,origin})};
 const doc=new EventTarget();doc.body={dataset:{previewOrigin:'https://admin.example',previewKind:kind}};
 const receive=(origin,source,data)=>{const event=new Event('message');Object.assign(event,{origin,source,data});win.dispatchEvent(event)};
 const init={channel:'ak.public-web.preview.v1',type:'init',loadId:'test-123'};
 return {win,doc,sent,receive,init};
}
test('bridge rejects foreign parents, sources and malformed messages and cleans up',()=>{
 const f=fixture();const cleanup=installPreview(f.win,f.doc);
 f.receive('https://evil.example',f.win.parent,f.init);f.receive('https://admin.example',{},f.init);f.receive('https://admin.example',f.win.parent,{...f.init,loadId:{}});
 assert.equal(f.sent.length,0);
 f.receive('https://admin.example',f.win.parent,f.init);
 assert.deepEqual(f.sent,[{data:{...f.init,type:'ready'},origin:'https://admin.example'}]);
 cleanup();f.receive('https://admin.example',f.win.parent,f.init);assert.equal(f.sent.length,1);
});
test('error page reports unavailable and escape only works after trusted initialization',()=>{
 const f=fixture('error');installPreview(f.win,f.doc);
 const esc=()=>{const event=new Event('keydown',{cancelable:true});Object.assign(event,{key:'Escape'});f.doc.dispatchEvent(event);return event.defaultPrevented};
 assert.equal(esc(),false);f.receive('https://admin.example',f.win.parent,f.init);assert.equal(f.sent[0].data.type,'unavailable');assert.equal(esc(),true);assert.equal(f.sent[1].data.type,'close');
});
test('standalone page never installs preview behavior',()=>{
 const f=fixture();f.win.parent=f.win;installPreview(f.win,f.doc);f.receive('https://admin.example',f.win,f.init);assert.equal(f.sent.length,0);
});
