"""Offline evidence audit using the pinned upstream JevBench implementation.
Usage: python audit.py --suite /path/to/jevbench
No inference or HTTP requests are made.
"""
import argparse
import hashlib
import json
import subprocess
import sys
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument('--suite', required=True, type=Path)
args = parser.parse_args()
root = Path(__file__).resolve().parent
suite = args.suite.resolve()
assert subprocess.check_output(['git', '-C', str(suite), 'rev-parse', 'HEAD'], text=True).strip() == 'f79a1cab94ab9a5879383b7ef9ee1805b9dc2d84'
sys.path.insert(0, str(suite))
from jevbench.tasks import load_jsonl, dataset_hash
from jevbench.adapters.base import build_question
from jevbench.scoring import score_task
from jevbench.summarize import summarize

load_lines = lambda name: [json.loads(line) for line in (root / name).read_text().splitlines()]
records = load_lines('records.jsonl')
raw = load_lines('raw-evidence.jsonl')
lengths = load_lines('token-lengths.jsonl')
starts = load_lines('thermal-starts.jsonl')
telemetry = load_lines('gpu-telemetry.jsonl')
summary = json.loads((root / 'summary.json').read_text())
tiers = {tier: load_jsonl(str(suite / 'datasets' / 'public' / (tier + '.jsonl'))) for tier in ('easy', 'original', 'hard')}
tasks = [t for ts in tiers.values() for t in ts]
by = {t.id: t for t in tasks}
assert len(records) == len(raw) == len(lengths) == len(starts) == len(tasks) == 231
assert [r['task_id'] for r in records] == [t.id for t in tasks]
assert len({r['task_id'] for r in records}) == 231
assert {r['task_id'] for r in starts} == set(by)
assert max(r['temperature_c'] for r in starts) <= 55
assert max(r['temperature_c'] for r in telemetry) < 83
raw_by = {r['task_id']: r for r in raw}
length_by = {r['task_id']: r for r in lengths}
assert set(raw_by) == set(length_by) == set(by)
for r in records:
    t = by[r['task_id']]
    evidence = raw_by[t.id]['evidence']
    encoded = json.dumps(evidence, ensure_ascii=False, allow_nan=False).encode()
    assert hashlib.sha256(encoded).hexdigest() == r['raw_sha256'] == raw_by[t.id]['raw_sha256'], t.id
    assert evidence['request'] == {'state': t.state, 'model': 'gemma-4-12b-it-go-system-one', 'questions': {'decision': build_question(t)}}
    assert r['cost_usd'] is None
    assert evidence['http_status'] == r['status_code']
    assert length_by[t.id]['exceeds_nvidia_2048_limit'] == (r['status_code'] == 400)
    if r['ok']:
        a = evidence['response']['answers']['decision']
        p = {'yes': a['noul'], 'no': 1 - a['noul']} if t.question['type'] == 'noul' else a['probabilities']
        assert p == r['probs_as_returned']
        score = score_task(p, t)
        for key in ('valid', 'strict_valid', 'renormalized', 'correct', 'predicted', 'probs'):
            assert score[key] == r[key], (t.id, key)
    else:
        assert r['status_code'] == 400 and not r['valid'] and not r['correct']
        assert evidence['response'].strip() == 'NVIDIA shared prefill: invalid NVIDIA Gemma4 prefill'
for tier, ts in tiers.items():
    assert dataset_hash(ts) == summary['datasets'][tier]['canonical_hash']
    got = summarize(ts, [r for r in records if r['task_id'] in {t.id for t in ts}])
    for key, value in got.items():
        assert summary['per_tier'][tier][key] == value, (tier, key)
assert dataset_hash(tasks) == summary['dataset_hash']
assert sum(r['correct'] for r in records) == 169
assert sum(not r['ok'] for r in records) == 36
stopped = [r for r in records if r['source_chunk'] == 'hard-chunk1']
assert len(stopped) == 3 and all(r['status_code'] == 400 for r in stopped)
assert summarize(tiers['hard'], stopped) == {k: v for k, v in summary['hard_original_stop'].items() if k != 'reason'}
assert summary['official_jevbench_score'] is None and summary['rank'] is None
print('Verified 231 unique attempts, all raw hashes/requests, upstream scores/aggregates, context failures and original hard stop.')
