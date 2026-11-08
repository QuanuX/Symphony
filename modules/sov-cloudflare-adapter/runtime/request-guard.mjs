// Receipt-owned guard for cf 1.0.0-beta.28. No caller-selected JavaScript.
import {readFileSync, writeSync} from 'node:fs';
const expected = JSON.parse(readFileSync(new URL('./request.json', import.meta.url), 'utf8'));
const fetchOnce = globalThis.fetch;
let attempted = 0;
const emit = value => { const b=Buffer.from(JSON.stringify(value)); const h=Buffer.alloc(4);h.writeUInt32BE(b.length);writeSync(3,h);writeSync(3,b); };
const stable = x => JSON.stringify(x, (_,v) => v && !Array.isArray(v) && typeof v==='object' ? Object.fromEntries(Object.keys(v).sort().map(k=>[k,v[k]])) : v);
const urlKey = value => {const u=new URL(value);u.searchParams.sort();return u.href;};
globalThis.fetch = async (input, init={}) => {
 const url=typeof input==='string'||input instanceof URL?String(input):input.url;
 const method=(init.method??input.method??'GET').toUpperCase();
 if (typeof expected.deadline_unix_ms!=='string'||!/^\d+$/.test(expected.deadline_unix_ms)||Date.now()>=Number(expected.deadline_unix_ms)) throw new Error('Use expired');
 if (attempted!==0 || urlKey(url)!==urlKey(expected.url) || method!==expected.method || new URL(url).origin!=='https://api.cloudflare.com') throw new Error('Request boundary refused');
 let body=init.body;
 if (expected.bodyKind==='json') {
  if (typeof body!=='string'||stable(JSON.parse(body))!==stable(expected.body)) throw new Error('Body boundary refused');
 } else if (body!==undefined&&body!==null) throw new Error('Unexpected body');
 attempted++;
 emit({stage:'dispatch_started',attempts:attempted});
 const response=await fetchOnce(input,{...init,redirect:'error'});
 const reader=response.body?.getReader();let chunks=[],size=0;
 if(reader) for(;;){const {done,value}=await reader.read();if(done)break;size+=value.length;if(size>2097152){await reader.cancel();throw new Error('Response bound');}chunks.push(value);}
 const bytes=Buffer.concat(chunks.map(c=>Buffer.from(c)),size);
 // Error bodies, headers and diagnostic text never leave the private child.
 emit({stage:'response_received',attempts:attempted,status:response.status,body:response.ok?bytes.toString('utf8'):null});
 return new Response([204,205,304].includes(response.status)?null:bytes,{status:response.status,headers:response.headers});
};
process.on('exit',()=>emit({stage:'exited',attempts:attempted}));
