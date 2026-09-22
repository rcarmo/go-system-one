// Run with Bun. Verified artifacts and one idle NVIDIA GPU are required.
// This collector uses loopback HTTP, three warm samples and a thermal guard.
import { createHash } from 'node:crypto';
import { mkdir, readFile, writeFile, rename } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';

const args = new Map<string,string>();
for(let i=2;i<process.argv.length;i+=2) {
 if(!process.argv[i]?.startsWith('--')||!process.argv[i+1])throw Error('use --name value arguments');
 args.set(process.argv[i].slice(2),process.argv[i+1]);
}
const required=(name:string)=>{const v=args.get(name);if(!v)throw Error(`--${name} required`);return v};
const binary=resolve(required('binary')), model=required('model'), tokenizer=required('tokenizer-dir');
const revision=required('revision'), output=resolve(required('out')), mode=required('mode');
if(!['automatic','serial'].includes(mode))throw Error('mode must be automatic or serial');
const sizes=(args.get('sizes')||'1,10,25,50,100').split(',').map(Number);
if(sizes.some(n=>!Number.isInteger(n)||n<1||n>256))throw Error('invalid sizes');
const root=resolve(import.meta.dir,'..');
const cohortBytes=await readFile(resolve(root,'docs/benchmarks/multifield-cohort.json'));
const cohort=JSON.parse(cohortBytes.toString());
const base=cohort.requests[0];
const url='http://127.0.0.1:18084';
function gpu(){
 const p=Bun.spawnSync(['nvidia-smi','-i','0','--query-gpu=name,driver_version,temperature.gpu,memory.used,clocks.sm','--format=csv,noheader,nounits']);
 if(p.exitCode)throw Error(p.stderr.toString());
 const [name,driver,t,m,clock]=p.stdout.toString().trim().split(',').map(x=>x.trim());
 const temperature=Number(t),memory_mib=Number(m),sm_mhz=Number(clock);
 if(!Number.isFinite(temperature)||!Number.isFinite(memory_mib))throw Error('invalid device query');
 return {name,driver,temperature,memory_mib,sm_mhz};
}
const active=Bun.spawnSync(['nvidia-smi','--query-compute-apps=pid','--format=csv,noheader,nounits']);
if(active.exitCode||active.stdout.toString().trim())throw Error('GPU busy or unavailable');
try {await fetch(url+'/go-system-one/v1/status',{signal:AbortSignal.timeout(500)});throw Error('benchmark port occupied');}catch(e){if(String(e).includes('occupied'))throw e;}
await mkdir(dirname(output),{recursive:true});
const command=[binary,'-model',model,'-tokenizer-dir',tokenizer,'-listen','127.0.0.1:18084'];
if(mode==='serial')command.push('-packed-token-rows','0'); // automatic deliberately omits the flag
const server=Bun.spawn(command,{stdout:Bun.file(output+'.server.log'),stderr:Bun.file(output+'.server.log')});
let stopping=false;
const stop=async()=>{if(stopping)return;stopping=true;server.kill('SIGTERM');const timer=setTimeout(()=>server.kill('SIGKILL'),35000);try{await server.exited;}finally{clearTimeout(timer);}};
process.on('SIGTERM',()=>{void stop().then(()=>process.exit(143));});
process.on('SIGINT',()=>{void stop().then(()=>process.exit(130));});
const report:any={schema:'go-system-one-batch-sweep-v1',revision,mode,started_utc:new Date().toISOString(),binary_sha256:createHash('sha256').update(await readFile(binary)).digest('hex'),cohort_sha256:createHash('sha256').update(cohortBytes).digest('hex'),model_sha256:'90fd944d227e9d9b68e7e2c7d5b57b79d4c66ed521b0919fbbd932cf834f6f8e',method:'One same-size warmup; three measured warm handler samples; idle GPU; cool to <=55C before each; abort at >=83C; 500ms temperature/memory samples; schema: boolean plus 3-way multi-token enum. Contexts cycle the frozen cohort with unique ticket suffixes. No labelled accuracy evaluation.',device:gpu(),cases:[]};
const save=async()=>{await writeFile(output+'.tmp',JSON.stringify(report,null,2)+'\n');await rename(output+'.tmp',output)};
try {
 let ready=false;
 for(let i=0;i<240;i++) {
  if(server.exitCode!==null)throw Error('server exited before readiness');
  try{const r=await fetch(url+'/go-system-one/v1/status',{signal:AbortSignal.timeout(500)});if(r.ok){ready=true;break;}}catch{}
  await Bun.sleep(1000);
 }
 if(!ready)throw Error('readiness timeout');
 for(const size of sizes){
  const request={...base,contexts:Array.from({length:size},(_,i)=>`${base.contexts[i%base.contexts.length]} Ticket ${i+1}.`)};
  const cell:any={size,request,status:'running',samples:[],warmup:null};report.cases.push(cell);await save();
  let expected:string|undefined;
  for(let i=0;i<4;i++) {
   let cool=false;for(let j=0;j<240;j++){if(gpu().temperature<=55){cool=true;break;}await Bun.sleep(1000);}if(!cool)throw Error('cooldown timeout');
   const before=gpu();let peak=before.memory_mib,maxTemp=before.temperature,guard='';
   const controller=new AbortController();
   const monitor=setInterval(()=>{try{const g=gpu();peak=Math.max(peak,g.memory_mib);maxTemp=Math.max(maxTemp,g.temperature);if(g.temperature>=83){guard='thermal guard 83C';controller.abort();}}catch(e){guard=String(e);controller.abort();}},500);
   const timeout=setTimeout(()=>{guard='request timeout 240s';controller.abort();},240000);
   const start=performance.now();let body:any;
   try {
    const r=await fetch(url+'/v1/decision',{method:'POST',body:JSON.stringify(request),headers:{'Content-Type':'application/json'},signal:controller.signal});
    if(!r.ok)throw Error(`HTTP ${r.status}: ${await r.text()}`);
    body=await r.json();
   } catch(e) {
    cell.status='aborted';cell.reason=guard||String(e);cell.aborted_sample={warmup:i===0,elapsed_ms:performance.now()-start,max_temperature_c:maxTemp,sampled_peak_memory_mib:peak};
    await save();console.log(JSON.stringify({mode,size,status:cell.status,reason:cell.reason}));
    throw Error(cell.reason);
   } finally {clearInterval(monitor);clearTimeout(timeout);}
   const wall=performance.now()-start,after=gpu();peak=Math.max(peak,after.memory_mib);maxTemp=Math.max(maxTemp,after.temperature);
   if(body.results?.length!==size||!Number.isFinite(body.timings.total_ms)||body.timings.total_ms<=0)throw Error('invalid response');
   for(const r of body.results)for(const f of Object.values(r.fields) as any[]){if(!f.candidates?.length||f.candidates.some((c:any)=>!Number.isFinite(c.probability)||c.probability<0||c.probability>1)||Math.abs(f.candidates.reduce((s:number,c:any)=>s+c.probability,0)-1)>1e-9)throw Error('invalid probability distribution');}
   const decisions=JSON.stringify(body.results.map((r:any)=>r.decision));if(expected!==undefined&&expected!==decisions)throw Error('decision drift between repeats');expected=decisions;
   const sample={handler_ms:body.timings.total_ms,wall_ms:wall,before,after,max_temperature_c:maxTemp,sampled_peak_memory_mib:peak};
   if(i===0)cell.warmup=sample;else cell.samples.push(sample);
   cell.results=body.results;await save();
  }
  const times=cell.samples.map((s:any)=>s.handler_ms).sort((a:number,b:number)=>a-b);
  cell.status='complete';cell.min_ms=times[0];cell.median_ms=times[1];cell.max_ms=times[2];cell.entries_per_second=1000*size/times[1];
  await save();console.log(JSON.stringify({mode,size,median_ms:cell.median_ms,entries_per_second:cell.entries_per_second}));
 }
}finally{await stop();}
