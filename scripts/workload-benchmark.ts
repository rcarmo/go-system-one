// Bun collector for an already-running, verified, loopback Go System One server.
// Save exact requests/results alongside handler timing; never rewrite history.
import { readFile, writeFile, mkdir, rename } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { createHash } from 'node:crypto';

const args=new Map<string,string>();
for(let i=2;i<process.argv.length;i+=2){if(!process.argv[i]?.startsWith('--')||!process.argv[i+1])throw Error('use --name value');args.set(process.argv[i].slice(2),process.argv[i+1]);}
const need=(k:string)=>{const v=args.get(k);if(!v)throw Error(`--${k} required`);return v};
const output=resolve(need('out')), revision=need('revision'), binary=resolve(need('binary'));
const url=new URL(args.get('url')||'http://127.0.0.1:18085');
if(url.protocol!=='http:'||!['127.0.0.1','localhost','[::1]'].includes(url.hostname))throw Error('loopback HTTP only');
const root=resolve(import.meta.dir,'..');
const baseline=JSON.parse(await readFile(resolve(root,'docs/benchmarks/request.json'),'utf8'));
const other=JSON.parse(await readFile(resolve(root,'docs/benchmarks/multifield-cohort.json'),'utf8'));
const status=await fetch(new URL('/go-system-one/v1/status',url));if(!status.ok)throw Error('server not ready');
function device(){const p=Bun.spawnSync(['nvidia-smi','-i','0','--query-gpu=temperature.gpu,memory.used,clocks.sm','--format=csv,noheader,nounits']);if(p.exitCode)throw Error(p.stderr.toString());const [temperature,memory_mib,sm_mhz]=p.stdout.toString().trim().split(',').map(Number);if(!Number.isFinite(temperature)||!Number.isFinite(memory_mib))throw Error('invalid device data');return {temperature,memory_mib,sm_mhz};}
async function cool(){for(let i=0;i<240;i++){if(device().temperature<=55)return;await Bun.sleep(1000);}throw Error('cooldown timeout');}
async function call(request:any){
 const controller=new AbortController();let reason='';const before=device();let peak=before.memory_mib,maxTemp=before.temperature;
 const monitor=setInterval(()=>{try{const d=device();peak=Math.max(peak,d.memory_mib);maxTemp=Math.max(maxTemp,d.temperature);if(d.temperature>=83){reason='83C thermal guard';controller.abort();}}catch(e){reason=String(e);controller.abort();}},500);
 const timeout=setTimeout(()=>{reason='request timeout';controller.abort();},120000),start=performance.now();
 try{
  const r=await fetch(new URL('/v1/decision',url),{method:'POST',body:JSON.stringify(request),headers:{'Content-Type':'application/json'},signal:controller.signal});
  if(!r.ok)throw Error(`HTTP ${r.status}: ${await r.text()}`);
  const response:any=await r.json();const after=device();
  if(response.results?.length!==request.contexts.length||!Number.isFinite(response.timings.total_ms)||response.timings.total_ms<=0)throw Error('invalid response');
  return {handler_ms:response.timings.total_ms,wall_ms:performance.now()-start,before,after,max_temperature_c:Math.max(maxTemp,after.temperature),sampled_peak_memory_mib:Math.max(peak,after.memory_mib),response};
 }catch(e){throw Error(reason||String(e));}finally{clearInterval(monitor);clearTimeout(timeout);}
}
async function save(path:string,data:any){await mkdir(dirname(path),{recursive:true});await writeFile(path+'.tmp',JSON.stringify(data,null,2)+'\n');await rename(path+'.tmp',path);}
const sortedStats=(samples:number[])=>{const s=[...samples].sort((a,b)=>a-b),n=s.length;return {n,min:s[0],median:n%2?s[(n-1)/2]:(s[n/2-1]+s[n/2])/2,p95:s[Math.ceil(.95*n)-1],p99:s[Math.ceil(.99*n)-1],max:s[n-1],samples:s};};
const provenance={revision,binary_sha256:createHash('sha256').update(await readFile(binary)).digest('hex'),model_sha256:'90fd944d227e9d9b68e7e2c7d5b57b79d4c66ed521b0919fbbd932cf834f6f8e',device:'NVIDIA GeForce RTX 3060 12 GB',driver:'580.173.02',status:await status.json()};
await cool();const warmup=await call(baseline),trials=[];let decisions:string|undefined;
for(let i=0;i<100;i++){
 const t=await call(baseline);const d=JSON.stringify(t.response.results.map((r:any)=>r.decision));
 if(decisions!==undefined&&decisions!==d)throw Error('decision drift');decisions=d;trials.push(t);
}
const distribution={...provenance,method:'One warm-up then 100 uninterrupted warm HTTP requests after cooling to <=55C; 83C guard; handler samples exclude loading.',request:baseline,request_sha256:createHash('sha256').update(JSON.stringify(baseline)).digest('hex'),warmup,...sortedStats(trials.map(t=>t.handler_ms)),decisions:JSON.parse(decisions!),trials};
await save(output+'/warm-latency.json',distribution);console.log(JSON.stringify({case:'warm100',median:distribution.median,p95:distribution.p95,p99:distribution.p99}));
const enumRequest={...baseline,schema:{priority:{type:'enum',description:'Priority level',choices:['critical incident','critical warning','routine maintenance','routine monitoring']}}};
const schema4={};for(let i=0;i<4;i++)(schema4 as any)[`urgent${i+1}`]={type:'boolean',description:'Does this need urgent handling?'};
const cacheMiss={...baseline,schema:{urgent:{type:'boolean',description:'Is this urgent?'}}};
const cases=[
 {label:'Boolean cache hit',request:baseline},
 {label:'Cache disabled',request:{...baseline,cache_prompt:false}},
 {label:'Cache miss',request:baseline,before:cacheMiss},
 {label:'Short context',request:{...baseline,contexts:['All services are healthy.']}},
 {label:'Long context',request:{...baseline,contexts:['A routine status report follows. '.repeat(60)+'All production services are now unavailable and customers cannot connect.']}},
 {label:'Multi-token enum tree',request:enumRequest},
 {label:'Multi-token enum greedy',request:{...enumRequest,mode:'auto',tree_max:2}},
 {label:'Four distinct contexts',request:{...baseline,contexts:other.requests[0].contexts.slice(0,4)}},
 {label:'Four boolean fields',request:{...baseline,schema:schema4}},
 {label:'Boolean + enum',request:{...other.requests[0],contexts:baseline.contexts}}
];
const report:any={...provenance,method:'One case warm-up; five warm HTTP handler samples per case. Cool to <=55C before each measured request. Cache-miss case replaces schema before each call. 83C guard. New workloads: compare via these exact request bodies, not historical names.',cases:[]};
for(const c of cases){
 await cool();if(c.before)await call(c.before);await call(c.request);
 const samples=[];let previous:string|undefined;
 for(let i=0;i<5;i++){
  await cool();if(c.before)await call(c.before);
  const t=await call(c.request),d=JSON.stringify(t.response.results.map((r:any)=>r.decision));if(previous!==undefined&&previous!==d)throw Error('decision drift');previous=d;samples.push(t);
 }
 const stats=sortedStats(samples.map(s=>s.handler_ms));report.cases.push({...c,...stats,trials:samples});await save(output+'/workloads.json',report);console.log(JSON.stringify({case:c.label,median:stats.median,min:stats.min,max:stats.max}));
}
