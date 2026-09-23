"""Aggregate and verify a completed, single-revision public run; no inference."""
import json,sys,hashlib,collections,datetime
from pathlib import Path
root=Path('/workspace/tmp/gso-jevbench-v140'); suite=Path('/workspace/tmp/gso-jevbench/v1.4.0')
sys.path.insert(0,str(suite))
from jevbench.tasks import load_jsonl,dataset_hash
from jevbench.adapters.base import build_question
from jevbench.scoring import score_task
from jevbench.summarize import summarize,metric
from jevbench.metrics import latency_summary
from jevbench.composite_v13 import tvd
sha=lambda b:hashlib.sha256(b).hexdigest()
output=root/'report';output.mkdir(exist_ok=True)
tiers={k:load_jsonl(str(suite/'datasets/public'/f'{k}.jsonl')) for k in ['easy','original','hard']}
tasks=[t for ts in tiers.values() for t in ts];by={t.id:t for t in tasks}
binary=sha((root/'server').read_bytes()); records={}; raws={}; chunks=[]; starts=[]
for d in sorted((root/'run').iterdir()):
 if not (d/'results.jsonl').exists():continue
 manifest=json.loads((d/'manifest.json').read_text()); rs=[json.loads(s) for s in (d/'results.jsonl').read_text().splitlines()]
 assert manifest['suite_revision']=='2fa63fa3226cb369795525ed011800f57dcbd894'
 assert manifest['service_revision']=='b18ee0d4748bac436999aa72c000e06406c3cce6'
 assert manifest['binary_sha256']==binary
 assert not manifest.get('diagnostic_continue_http400',False)
 thermal=[json.loads(s) for s in (d/'thermal-starts.jsonl').read_text().splitlines()]
 assert len(thermal)==len(rs)==len(list((d/'raw').glob('*.json')))
 for r,start in zip(rs,thermal):
  tid=r['task_id'];assert tid not in records and tid==start['task_id'];assert start['temperature_c']<=55
  t=by[tid];data=(d/'raw'/(sha(tid.encode())+'.json')).read_bytes();assert sha(data)==r['raw_sha256'];raw=json.loads(data)
  assert raw['request']=={'state':t.state,'model':'gemma-4-12b-it-go-system-one','questions':{'decision':build_question(t)}}
  assert raw['http_status']==r['status_code']==200 and r['ok'] and r['cost_usd'] is None
  ans=raw['response']['answers']['decision'];p={'yes':ans['noul'],'no':1-ans['noul']} if t.question['type']=='noul' else ans['probabilities']
  assert p==r['probs_as_returned'];sc=score_task(p,t)
  for k in ['correct','predicted','valid','strict_valid','renormalized','probs']:assert sc[k]==r[k],(tid,k)
  assert sc['strict_valid'] and not sc['renormalized']
  records[tid]=dict(r,source_chunk=d.name,tier=manifest['tier'],question_type=t.question['type'])
  raws[tid]={'task_id':tid,'source_chunk':d.name,'raw_sha256':r['raw_sha256'],'evidence':raw}
  starts.append(dict(start,source_chunk=d.name))
 chunks.append({'directory':d.name,'manifest':manifest,'durable_records':len(rs),'results_sha256':sha((d/'results.jsonl').read_bytes()),'interrupted':not (d/'summary.json').exists(),'first_request':rs[0]['task_id'] if rs else None})
assert set(records)==set(by)
rs=[records[t.id] for t in tasks]
result={'schema':'go-system-one-jevbench-v140-public-v1','scope':'All 231 public items, one service revision; no diagnostic continuation; not the official sealed-set score','official_jevbench_score':None,'rank':None,'usd_per_1000_decisions':None,'per_tier':{},'by_type':{},'all_public':summarize(tasks,rs)}
for tier,ts in tiers.items():
 subset=[records[t.id] for t in ts];s=summarize(ts,subset)
 s['first_requests']=[{'task_id':c['first_request'],'latency_s':records[c['first_request']]['latency_s']}for c in chunks if c['manifest']['tier']==tier and c['first_request']]
 s['latency_excluding_first_requests']=latency_summary([r['latency_s']for r in subset if r['task_id']not in{x['task_id']for x in s['first_requests']}])
 result['per_tier'][tier]=s
for kind in ['choice','noul','score']:result['by_type'][kind]=metric([t for t in tasks if t.question['type']==kind],rs)
pairs=[]
for t in tiers['hard']:
 gold=t.provenance.get('gold_probs')
 if gold is not None:pairs.append({'task_id':t.id,'tvd':tvd(records[t.id]['probs'],gold,t.labels)})
result['hard_gold_distributions']={'n':len(pairs),'mean_tvd':sum(p['tvd']for p in pairs)/len(pairs),'per_task':pairs}
result['chunk_provenance']=chunks
telemetry=[dict(json.loads(s),source_log=str(p.relative_to(root))) for p in sorted((root/'chunks').glob('*/gpu-telemetry.jsonl')) for s in p.read_text().splitlines()]
assert max(t['temperature_c']for t in telemetry)<83
result['thermal']={'max_sampled_temperature_c':max(t['temperature_c']for t in telemetry),'max_sampled_device_memory_mib':max(t['memory_mib']for t in telemetry),'max_start_temperature_c':max(t['temperature_c']for t in starts),'cooldown_total_s':sum(t['wait_s']for t in starts),'samples':len(telemetry)}
result['datasets']={tier:{'n':len(ts),'canonical_hash':dataset_hash(ts),'file_sha256':sha((suite/'datasets/public'/f'{tier}.jsonl').read_bytes())}for tier,ts in tiers.items()}
result['dataset_hash']=dataset_hash(tasks)
result['provenance']={'service_revision':'b18ee0d4748bac436999aa72c000e06406c3cce6','suite_revision':'2fa63fa3226cb369795525ed011800f57dcbd894','suite_tag':'v1.4.0','binary_sha256':binary,'model_sha256':'90fd944d227e9d9b68e7e2c7d5b57b79d4c66ed521b0919fbbd932cf834f6f8e','device':'NVIDIA GeForce RTX 3060 12 GB','driver':'580.173.02','python':sys.version,'packing':'default automatic 512 rows; existing per-entry serial fallback','upstream_tests':'104 passed, 5 subtests passed','started_utc':datetime.datetime.fromtimestamp(min(r['ts']for r in rs),datetime.timezone.utc).isoformat(),'finished_utc':datetime.datetime.fromtimestamp(max(r['ts']for r in rs),datetime.timezone.utc).isoformat()}
# Compare to provisional old+recheck by task, not just headline totals.
old=[json.loads(s)for s in Path('/workspace/projects/go-system-one/docs/benchmarks/data/jevbench-public-20260923/records.jsonl').read_text().splitlines()]
prior={r['task_id']:r for r in old}
for c in json.loads(Path('/workspace/projects/go-system-one/docs/benchmarks/data/long-context-20260923/recheck.json').read_text())['cases']:prior[c['task_id']]=c['scored']
result['previous_mixed_revision_comparison']={'winner_changes':sum(r['predicted']!=prior[r['task_id']]['predicted']for r in rs),'correctness_changes':sum(r['correct']!=prior[r['task_id']]['correct']for r in rs),'max_probability_delta':max(abs(p-prior[r['task_id']]['probs'][k])for r in rs for k,p in r['probs'].items()),'note':'Historical comparison only; no old measurement enters this new result.'}
for name,values in [('records.jsonl',rs),('raw-evidence.jsonl',[raws[t.id]for t in tasks]),('thermal-starts.jsonl',starts),('gpu-telemetry.jsonl',telemetry)]:
 (output/name).write_text(''.join(json.dumps(v,ensure_ascii=False,allow_nan=False)+'\n'for v in values))
(output/'summary.json').write_text(json.dumps(result,indent=2,allow_nan=False)+'\n')
for tier,s in result['per_tier'].items():print(tier,{k:s[k]for k in ['n_attempted','n_correct','n_valid','accuracy','brier_mean','ordinal_mae','latency']},'ECE',s['ece']['ece'])
print('pooled',result['all_public']['n_correct'],result['all_public']['accuracy'],result['all_public']['latency']);print('thermal',result['thermal']);print('gold',result['hard_gold_distributions']['mean_tvd']);print('prior',result['previous_mixed_revision_comparison'])
