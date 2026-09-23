"""Upstream Runner/TypeSafe adapter, with only an untimed thermal prepare hook.
No task, request, prediction, scorer or probability transformation is changed.
"""
import sys, os, json, time, subprocess, datetime
from pathlib import Path
sys.path.insert(0, '/workspace/tmp/gso-jevbench/v1.4.0')
from jevbench.adapters.typesafe import TypeSafeAdapter
from jevbench.runner import Runner
from jevbench.budget import Ledger
from jevbench.tasks import load_jsonl, dataset_hash
from jevbench.summarize import summarize
root=Path('/workspace/tmp/gso-jevbench-v140')
suite=Path('/workspace/tmp/gso-jevbench/v1.4.0')
tier=sys.argv[1]
assert tier in ('easy','original','hard')
chunk=sys.argv[2]
limit=int(sys.argv[3]) if len(sys.argv)>3 else 24
assert chunk.isalnum() and limit>0
out=root/'run'/(tier+'-'+chunk)
out.mkdir(parents=True,exist_ok=False)
if list(root.glob('chunks/**/ABORTED')): raise RuntimeError('Previous thermal failure requires review')
def gpu():
    s=subprocess.check_output(['nvidia-smi','-i','0','--query-gpu=temperature.gpu,memory.used,clocks.sm','--format=csv,noheader,nounits'],text=True)
    t,m,c=map(int,s.strip().split(','))
    return {'temperature_c':t,'memory_mib':m,'sm_mhz':c}
class CooledTypeSafe(TypeSafeAdapter):
    def prepare(self,task):
        started=time.perf_counter()
        for _ in range(600):
            before=gpu()
            if before['temperature_c'] <= 55:
                before=gpu()
                if before['temperature_c'] <= 55: break
            time.sleep(1)
        else: raise RuntimeError('cooldown timeout')
        with (out/'thermal-starts.jsonl').open('a') as f:
            f.write(json.dumps({'task_id':task.id,'utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),'wait_s':time.perf_counter()-started,**before})+'\n')
        # Upstream Runner catches prepare exceptions; hard exit avoids continuing
        # inference after a failed thermal precondition.
    cost_basis='local_gpu_compute_cost_unmeasured'
# Ensure failures in thermal preparation cannot be ignored by upstream Runner.
original_prepare=CooledTypeSafe.prepare
def guarded_prepare(self,task):
    try: original_prepare(self,task)
    except BaseException as e:
        print('THERMAL PREP FAILED',repr(e),file=sys.stderr,flush=True)
        os._exit(70)
CooledTypeSafe.prepare=guarded_prepare
full_tasks=load_jsonl(str(suite/'datasets'/'public'/(tier+'.jsonl')))
previous=[]
for directory in sorted((root/'run').iterdir()):
    if directory==out or not directory.is_dir(): continue
    m=directory/'manifest.json'; results=directory/'results.jsonl'
    if m.exists() and results.exists() and json.loads(m.read_text()).get('tier')==tier:
        previous.extend(json.loads(line) for line in results.read_text().splitlines() if line.strip())
ids=[r['task_id'] for r in previous]
assert len(ids)==len(set(ids)), 'duplicate prior records'
assert set(ids)<=set(t.id for t in full_tasks), 'unknown prior task'
# A saved HTTP refusal/outage stop is not circumvented by resuming another chunk.
if any(r.get('status_code') in (401,403,429) for r in previous): raise RuntimeError('prior stop status')
ordered=[next(r for r in previous if r['task_id']==t.id) for t in full_tasks if t.id in ids]
errors=0
for r in ordered:
    errors=errors+1 if not r['ok'] and r['status_code']!=422 else 0
    if errors>=3: raise RuntimeError('prior infrastructure stop')
tasks=[t for t in full_tasks if t.id not in ids][:limit]
if not tasks: raise RuntimeError('no unattempted items')
adapter=CooledTypeSafe(endpoint='http://127.0.0.1:18087',model='gemma-4-12b-it-go-system-one',key_env='',price_input_per_m=None,price_output_per_m=None)
ledger=Ledger(str(out/'ledger.jsonl'),cap_usd=1)
runner=Runner(adapter,ledger,raw_dir=str(out/'raw'),default_reserve_usd=0)
started=datetime.datetime.now(datetime.timezone.utc).isoformat()
manifest={'suite_revision':'2fa63fa3226cb369795525ed011800f57dcbd894','service_revision':'b18ee0d4748bac436999aa72c000e06406c3cce6','tier':tier,'n_planned':len(tasks),'binary_sha256':__import__('hashlib').sha256((root/'server').read_bytes()).hexdigest(),'dataset_hash':dataset_hash(tasks),'adapter':'typesafe','endpoint':adapter.endpoint,'model':adapter.model,'python':sys.version,'started_utc':started,'protocol':'Unmodified TypeSafeAdapter.run and Runner scoring; prepare only waits for <=55C before every request. External telemetry aborts at >=83C. No warmup, retries, truncation, prompt tuning, concurrency, costs or latency adjustments.'}
(out/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
# Keep upstream Runner.run_task/scoring; carry consecutive-error state across
# deliberately bounded chunks. Save each row exclusively and fsync as upstream.
records=[]
with (out/'results.jsonl').open('x') as stream:
    for t in tasks:
        r=runner.run_task(t); records.append(r)
        stream.write(json.dumps(r,allow_nan=False)+'\n');stream.flush();os.fsync(stream.fileno())
        print(t.id, r['status'], 'HTTP', r['status_code'], flush=True)
        errors=errors+1 if not r['ok'] and r['status_code']!=422 else 0
        if r['status_code'] in (401,403,429) or errors>=3:
            print('STOP: upstream access/rate limit or consecutive infrastructure errors',flush=True)
            break
summary=summarize(tasks,records)
(out/'summary.json').write_text(json.dumps(summary,indent=2)+'\n')
manifest.update(n_attempted=len(records),finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),complete=len(records)==len(tasks))
(out/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
print(json.dumps({k:summary[k] for k in ('n_planned','n_attempted','n_correct','accuracy','schema_validity','operational_success','latency','complete')},indent=2),flush=True)
if len(records)!=len(tasks): sys.exit(3)
