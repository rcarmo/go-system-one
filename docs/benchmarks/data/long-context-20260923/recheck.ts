import {readFileSync,writeFileSync,existsSync} from 'node:fs';
import {createHash} from 'node:crypto';
const base='/workspace/tmp/gso-jevbench/report/';
const old=readFileSync(base+'raw-evidence.jsonl','utf8').trim().split('\n').map(JSON.parse).filter(r=>r.evidence.http_status===400);
const offset=Number(process.argv[2]),count=Number(process.argv[3]);
const dest=`/workspace/tmp/gso-long-context/recheck-${offset}.json`;if(existsSync(dest))throw Error('exists');
const report:any={base_revision:'c3f6ffd92aacd87cf6bd4cd7c91a9196b58475ec',candidate:'uncommitted long-context fix; exact diff hash retained separately',binary_sha256:createHash('sha256').update(readFileSync('/workspace/tmp/gso-long-context/server-final')).digest('hex'),method:'Previously rejected inputs only; unchanged exact requests; one call each; cool <=55C before each, abort >=83C; no labelled tuning or input truncation; client HTTP latency',started_utc:new Date().toISOString(),cases:[]};
function gpu(){const s=Bun.spawnSync(['nvidia-smi','--query-gpu=temperature.gpu,memory.used','--format=csv,noheader,nounits']).stdout.toString().trim().split(',').map(Number);return {temperature_c:s[0],memory_mib:s[1]}}
for(const c of old.slice(offset,offset+count)){
 for(let j=0;j<600&&gpu().temperature_c>55;j++)await Bun.sleep(1000);const before=gpu();if(before.temperature_c>55)throw Error('cooldown');
 let peak=before.memory_mib,maxT=before.temperature_c;const abort=new AbortController();
 const poll=setInterval(()=>{const s=gpu();peak=Math.max(peak,s.memory_mib);maxT=Math.max(maxT,s.temperature_c);if(maxT>=83)abort.abort()},500);
 const timeout=setTimeout(()=>abort.abort(),120000);try{const start=performance.now();const r=await fetch('http://127.0.0.1:18088/v1/systemone',{method:'POST',body:JSON.stringify(c.evidence.request),headers:{'Content-Type':'application/json'},signal:abort.signal});const text=await r.text(),http_ms=performance.now()-start;let response;try{response=JSON.parse(text)}catch{response=text};report.cases.push({task_id:c.task_id,request:c.evidence.request,status:r.status,response,http_ms,before,max_temperature_c:maxT,sampled_peak_memory_mib:peak});writeFileSync(dest,JSON.stringify(report,null,2)+'\n');console.log(c.task_id,r.status,http_ms.toFixed(0));}finally{clearInterval(poll);clearTimeout(timeout)}
}
