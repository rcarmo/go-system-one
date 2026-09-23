import{readFileSync,writeFileSync}from'node:fs';import{createHash}from'node:crypto';
const request=JSON.parse(readFileSync('/workspace/projects/go-system-one/docs/benchmarks/request.json','utf8'));
const results=[];const dir='/workspace/tmp/gso-long-context/';const baseline='/workspace/tmp/gso-jevbench/server';
function gpu(){const s=Bun.spawnSync(['nvidia-smi','--query-gpu=temperature.gpu,memory.used','--format=csv,noheader,nounits']);if(s.exitCode)throw Error('telemetry');const [t,m]=s.stdout.toString().trim().split(',').map(Number);return {temperature_c:t,memory_mib:m}}
async function cool(){for(let i=0;i<600;i++){if(gpu().temperature_c<=55)return;await Bun.sleep(1000)}throw Error('cooldown')}
for(const [label,binary]of [['baseline',baseline],['candidate',dir+'server-final'],['candidate',dir+'server-final'],['baseline',baseline]]){
 const server=Bun.spawn([binary,'-model','/tmp/qev-gemma4-12b/gemma-4-12b-it-UD-Q4_K_XL.gguf','-tokenizer-dir','/tmp/qev-gemma4-12b/tokenizer','-listen','127.0.0.1:18089'],{stdout:Bun.file(dir+'short-'+results.length+'.log'),stderr:Bun.file(dir+'short-'+results.length+'.log')});
 try{
 let ready=false;for(let i=0;i<180;i++){try{const r=await fetch('http://127.0.0.1:18089/go-system-one/v1/status');if(r.ok){ready=true;break}}catch{}if(server.exitCode!==null)throw Error('exited');await Bun.sleep(1000)}if(!ready)throw Error('not ready');
 await cool();const before=gpu();const samples=[];let warmup;for(let i=0;i<21;i++){
 if(gpu().temperature_c>=83)throw Error('guard');const r=await fetch('http://127.0.0.1:18089/v1/decision',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(request)});if(!r.ok)throw Error(await r.text());const response=await r.json();if(i===0)warmup=response;else samples.push(response);}
 const times=samples.map(x=>x.timings.total_ms).sort((a,b)=>a-b);const report={label,binary_sha256:createHash('sha256').update(readFileSync(binary)).digest('hex'),request,warmup,samples,median_ms:(times[9]+times[10])/2,min_ms:times[0],max_ms:times.at(-1),before,after:gpu()};results.push(report);writeFileSync(dir+'short-check.json',JSON.stringify({method:'A/B/B/A, one warmup then20 uninterrupted handler samples; each block starts <=55C. Single boolean regression check only, not chart refresh.',runs:results},null,2));console.log(label,report.median_ms);
 }finally{server.kill('SIGTERM');await server.exited;}
}
