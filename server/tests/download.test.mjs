import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
const source = await readFile(new URL('../resource/static/js/download.js',import.meta.url),'utf8');
const {detectPlatform} = await import('data:text/javascript;base64,'+Buffer.from(source).toString('base64'));
for(const [name,nav,want] of [
 ['iPhone',{userAgent:'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X)'},'ios'],
 ['iPad',{userAgent:'Mozilla/5.0 (iPad; CPU OS 18_0)'},'ios'],
 ['iPad desktop mode',{userAgent:'Macintosh',platform:'MacIntel',maxTouchPoints:5},'ios'],
 ['Android',{userAgent:'Mozilla/5.0 (Linux; Android 15; Pixel 9)'},'android'],
 ['explicit Harmony',{userAgent:'Mozilla/5.0 (Linux; Android 12; HarmonyOS)'},'harmony'],
 ['OpenHarmony',{userAgent:'Mozilla/5.0 (OpenHarmony 5.0)'},'harmony'],
 ['ambiguous Huawei',{userAgent:'Mozilla/5.0 (HUAWEI; Mobile)'},'all'],
 ['ambiguous Huawei Android UA',{userAgent:'Mozilla/5.0 (Linux; Android 12; HUAWEI)'},'all'],
 ['desktop Mac',{platform:'MacIntel',maxTouchPoints:0,userAgent:'Macintosh'},'all'],
 ['unknown',{},'all'],
]) test(name,()=>assert.equal(detectPlatform(nav),want));
