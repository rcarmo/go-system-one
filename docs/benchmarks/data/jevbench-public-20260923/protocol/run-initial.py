"""Upstream Runner/TypeSafe adapter, with only an untimed thermal prepare hook.
No task, request, prediction, scorer or probability transformation is changed.
"""
import sys, os, json, time, subprocess, datetime
from pathlib import Path
sys.path.insert(0, '/workspace/tmp/gso-jevbench/upstream')
from jevbench.adapters.typesafe import TypeSafeAdapter
from jevbench.runner import Runner
from jevbench.budget import Ledger
from jevbench.tasks import load_jsonl, dataset_hash
from jevbench.summarize import summarize
root=Path('/workspace/tmp/gso-jevbench')
tier=sys.argv[1]
assert tier in ('easy','original','hard')
out=root/'run'/tier
out.mkdir(parents=True,exist_ok=True)
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
tasks=load_jsonl(str(root/'upstream'/'datasets'/'public'/(tier+'.jsonl')))
adapter=CooledTypeSafe(endpoint='http://127.0.0.1:18087',model='gemma-4-12b-it-go-system-one',key_env='',price_input_per_m=None,price_output_per_m=None)
ledger=Ledger(str(out/'ledger.jsonl'),cap_usd=1)
runner=Runner(adapter,ledger,raw_dir=str(out/'raw'),default_reserve_usd=0)
started=datetime.datetime.now(datetime.timezone.utc).isoformat()
manifest={'suite_revision':'f79a1cab94ab9a5879383b7ef9ee1805b9dc2d84','service_revision':'a4e49833c85c96b775f2f2240f129279001f3167','tier':tier,'n_planned':len(tasks),'dataset_hash':dataset_hash(tasks),'adapter':'typesafe','endpoint':adapter.endpoint,'model':adapter.model,'python':sys.version,'started_utc':started,'protocol':'Unmodified TypeSafeAdapter.run and Runner scoring; prepare only waits for <=55C before every request. External telemetry aborts at >=83C. No warmup, retries, truncation, prompt tuning, concurrency, costs or latency adjustments.'}
(out/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
records=runner.run_all(tasks,results_path=str(out/'results.jsonl'))
summary=summarize(tasks,records)
(out/'summary.json').write_text(json.dumps(summary,indent=2)+'\n')
manifest.update(n_attempted=len(records),finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),complete=len(records)==len(tasks))
(out/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
print(json.dumps({k:summary[k] for k in ('n_planned','n_attempted','n_correct','accuracy','schema_validity','operational_success','latency','complete')},indent=2),flush=True)
if len(records)!=len(tasks): sys.exit(3)
