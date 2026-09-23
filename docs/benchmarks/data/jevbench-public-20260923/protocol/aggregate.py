import json,sys,hashlib,collections,math
from pathlib import Path
root=Path('/workspace/tmp/gso-jevbench'); upstream=root/'upstream'
sys.path.insert(0,str(upstream))
from jevbench.tasks import load_jsonl,dataset_hash
from jevbench.adapters.base import build_question
from jevbench.scoring import score_task
from jevbench.summarize import summarize,metric
from jevbench.metrics import latency_summary
from jevbench.composite_v13 import tvd, calibration, chance_corrected_accuracy
sha=lambda b:hashlib.sha256(b).hexdigest()
output=root/'report';output.mkdir(exist_ok=True)
tiers={k:load_jsonl(str(upstream/'datasets/public'/f'{k}.jsonl')) for k in ['easy','original','hard']}
tasks=[t for ts in tiers.values() for t in ts];by={t.id:t for t in tasks}
records={};raw_records={};chunks=[];starts=[]
for d in sorted((root/'run').iterdir()):
 if not (d/'results.jsonl').exists():continue
 manifest=json.loads((d/'manifest.json').read_text()); lines=(d/'results.jsonl').read_text().splitlines(); rs=[json.loads(s) for s in lines]
 assert manifest['suite_revision']=='f79a1cab94ab9a5879383b7ef9ee1805b9dc2d84'
 assert manifest['service_revision']=='a4e49833c85c96b775f2f2240f129279001f3167'
 thermal=[json.loads(s) for s in (d/'thermal-starts.jsonl').read_text().splitlines()]
 assert len(thermal)==len(rs)==len(list((d/'raw').glob('*.json')))
 for r,start in zip(rs,thermal):
  tid=r['task_id']; assert tid not in records and tid==start['task_id']; assert start['temperature_c']<=55
  t=by[tid];rawpath=d/'raw'/(sha(tid.encode())+'.json');data=rawpath.read_bytes();assert sha(data)==r['raw_sha256'];raw=json.loads(data)
  assert raw['request']=={'state':t.state,'model':'gemma-4-12b-it-go-system-one','questions':{'decision':build_question(t)}}
  assert raw['http_status']==r['status_code'];assert r['cost_usd'] is None
  if r['ok']:
   ans=raw['response']['answers']['decision'];p={'yes':ans['noul'],'no':1-ans['noul']} if t.question['type']=='noul' else ans['probabilities']
   assert p==r['probs_as_returned'];recomputed=score_task(p,t)
   for k in ['correct','predicted','valid','strict_valid','renormalized','probs']:assert recomputed[k]==r[k],(tid,k)
  else: assert r['status_code']==400 and not r['correct'] and not r['valid']
  r=dict(r,source_chunk=d.name,tier=manifest['tier'],question_type=t.question['type'])
  records[tid]=r;raw_records[tid]={'task_id':tid,'source_chunk':d.name,'raw_sha256':r['raw_sha256'],'evidence':raw};starts.append(start)
 chunks.append({'directory':d.name,'manifest':manifest,'durable_records':len(rs),'results_sha256':sha((d/'results.jsonl').read_bytes()),'interrupted':not (d/'summary.json').exists(),'first_request':rs[0]['task_id'] if rs else None})
assert set(records)==set(by)
rs=[records[t.id] for t in tasks]
orderedraw=[raw_records[t.id] for t in tasks]
(output/'records.jsonl').write_text(''.join(json.dumps(r,allow_nan=False)+'\n' for r in rs))
(output/'raw-evidence.jsonl').write_text(''.join(json.dumps(r,allow_nan=False,ensure_ascii=False)+'\n' for r in orderedraw))
result={'scope':'231 public items; easy/original complete, hard supplemental diagnostic after upstream stop; NOT official ranked JevBench Score','official_jevbench_score':None,'rank':None,'usd_per_1000_decisions':None,'per_tier':{},'by_type':{}}
for tier,ts in tiers.items():
 subset=[records[t.id] for t in ts];s=summarize(ts,subset)
 s['status']='completed_public_tier' if tier!='hard' else 'supplemental_http400_continuation_diagnostic'
 s['latency_successes']=latency_summary([r['latency_s'] for r in subset if r['ok']])
 s['first_requests']=[{'task_id':c['first_request'],'latency_s':records[c['first_request']]['latency_s'],'status':records[c['first_request']]['status']} for c in chunks if c['manifest']['tier']==tier and c['first_request']]
 s['latency_excluding_first_requests']=latency_summary([r['latency_s'] for r in subset if r['task_id'] not in {x['task_id'] for x in s['first_requests']}])
 s['chance_accuracy_public']=sum(1/len(t.labels) for t in ts)/len(ts)
 s['chance_corrected_accuracy_public']=chance_corrected_accuracy(s['accuracy'],s['chance_accuracy_public'])
 result['per_tier'][tier]=s
for kind in ['choice','noul','score']:
 ts=[t for t in tasks if t.question['type']==kind];result['by_type'][kind]=metric(ts,rs)
hard=tiers['hard'];official=[r for r in rs if r['source_chunk']=='hard-chunk1']
result['hard_original_stop']=summarize(hard,official)
result['hard_original_stop']['reason']='Three consecutive HTTP 400 responses; original run stopped as upstream requires. 108 hard items were unattempted in this run.'
result['all_public_diagnostic']=summarize(tasks,rs)
pairs=[]
for t in hard:
 gold=t.provenance.get('gold_probs');r=records[t.id]
 if gold is not None and r['valid']: pairs.append({'task_id':t.id,'tvd':tvd(r['probs'],gold,t.labels)})
mean=sum(p['tvd'] for p in pairs)/len(pairs) if pairs else None
result['hard_gold_distribution_diagnostic']={'n_planned':sum('gold_probs' in t.provenance for t in hard),'n_valid':len(pairs),'mean_tvd':mean,'per_task':pairs,'calibration_axis_public_diagnostic':calibration(result['per_tier']['hard']['ece']['ece'],mean),'note':'Computed on valid public hard responses only; failed requests do not acquire invented probability distributions; not full-suite calibration.'}
result['failures']=dict(collections.Counter(r['error'] for r in rs if not r['ok']))
result['failure_ids']=[r['task_id'] for r in rs if not r['ok']]
result['chunk_provenance']=chunks
telemetry=[json.loads(s) for p in [root/'gpu-telemetry.jsonl',*sorted((root/'chunks').glob('*/gpu-telemetry.jsonl'))] for s in p.read_text().splitlines()]
assert max(t['temperature_c'] for t in telemetry)<83
result['thermal']={'max_sampled_temperature_c':max(t['temperature_c'] for t in telemetry),'max_sampled_device_memory_mib':max(t['memory_mib'] for t in telemetry),'max_start_temperature_c':max(t['temperature_c'] for t in starts),'cooldown_total_s':sum(t['wait_s'] for t in starts),'samples':len(telemetry),'note':'Device-wide memory sampled at 500ms; brief peaks may be missed; cooling and loading excluded from HTTP timing.'}
published=json.loads((upstream/'results/v1.2/jevbench-v1.2-per-task.json').read_text())
peers=[]
for key in ['jev-1.13.0','semif-qwen3.5-4b','winnow-12b','kev-0.6b']:
 s=published['systems'][key];row={'key':key,'display':s['display'],'source':'pinned upstream public_tasks outcome codes, not a local rerun','per_tier':{}}
 for tier,ts in tiers.items():
  codes=[s['public_tasks'][t.id][0] for t in ts];n=len(ts);row['per_tier'][tier]={'n':n,'correct':codes.count('c'),'wrong':codes.count('w'),'failed':codes.count('f'),'unattempted':codes.count('n'),'accuracy':codes.count('c')/n}
 peers.append(row)
result['same_public_id_comparators']=peers
result['datasets']={tier:{'n':len(ts),'canonical_hash':dataset_hash(ts),'file_sha256':sha((upstream/'datasets/public'/f'{tier}.jsonl').read_bytes())}for tier,ts in tiers.items()}
result['dataset_hash']=dataset_hash(tasks)
result['provenance']={'service_revision':'a4e49833c85c96b775f2f2240f129279001f3167','suite_revision':'f79a1cab94ab9a5879383b7ef9ee1805b9dc2d84','binary_sha256':sha((root/'server').read_bytes()),'model_sha256':'90fd944d227e9d9b68e7e2c7d5b57b79d4c66ed521b0919fbbd932cf834f6f8e','device':'NVIDIA GeForce RTX 3060 12 GB','driver':'580.173.02','python':sys.version,'status':json.loads((root/'status.json').read_text()),'packing':'default automatic (512 token rows, per-entry serial fallback)','upstream_tests':'97 passed, 5 subtests passed','no_model_or_prompt_changes':True}
(output/'summary.json').write_text(json.dumps(result,indent=2,allow_nan=False)+'\n')
for tier,s in result['per_tier'].items():print(tier,{k:s[k] for k in ['n_planned','n_attempted','n_correct','n_valid','accuracy','brier_mean','ordinal_mae','latency']},'ECE',s['ece']['ece'],'success latency',s['latency_successes'])
print('TOTAL',result['all_public_diagnostic']['n_correct'],result['all_public_diagnostic']['accuracy'])
print('thermal',result['thermal']);print('gold',result['hard_gold_distribution_diagnostic']);print('errors',result['failures']);print('peers',peers)
