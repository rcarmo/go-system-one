// Run with Bun against an idle GPU and verified artifacts. The TypeSafe route
// has no handler timing field: report client HTTP latency, never mix chart units.
import { createHash } from 'node:crypto';
import { readFile, writeFile, mkdir, rename } from 'node:fs/promises';
import { resolve, dirname } from 'node:path';

const args = new Map<string, string>();
for (let i = 2; i < process.argv.length; i += 2) {
  if (!process.argv[i]?.startsWith('--') || !process.argv[i + 1]) throw Error('use --name value');
  args.set(process.argv[i].slice(2), process.argv[i + 1]);
}
const need = (name: string) => { const value = args.get(name); if (!value) throw Error(`--${name} required`); return value; };
const binary = resolve(need('binary')), output = resolve(need('out')), revision = need('revision');
const url = 'http://127.0.0.1:18086';
const hash = (value: string | Buffer) => createHash('sha256').update(value).digest('hex');
function gpu() {
  const p = Bun.spawnSync(['nvidia-smi', '-i', '0', '--query-gpu=name,driver_version,temperature.gpu,memory.used,clocks.sm', '--format=csv,noheader,nounits']);
  if (p.exitCode) throw Error(p.stderr.toString());
  const [name, driver, t, m, c] = p.stdout.toString().trim().split(',').map(s => s.trim());
  const temperature = Number(t), memory_mib = Number(m), sm_mhz = Number(c);
  if (![temperature, memory_mib, sm_mhz].every(Number.isFinite)) throw Error('invalid GPU telemetry');
  return { name, driver, temperature, memory_mib, sm_mhz };
}
async function cool() {
  for (let i = 0; i < 240; i++) { if (gpu().temperature <= 55) return; await Bun.sleep(1000); }
  throw Error('cooldown timeout');
}
const active = Bun.spawnSync(['nvidia-smi', '--query-compute-apps=pid', '--format=csv,noheader,nounits']);
if (active.exitCode || active.stdout.toString().trim()) throw Error('GPU busy or unavailable');
try { await fetch(url + '/go-system-one/v1/status', { signal: AbortSignal.timeout(500) }); throw Error('port occupied'); }
catch (e) { if (String(e).includes('occupied')) throw e; }
await mkdir(dirname(output), { recursive: true });
const command = [binary, '-model', need('model'), '-tokenizer-dir', need('tokenizer-dir'), '-listen', '127.0.0.1:18086'];
const server = Bun.spawn(command, { stdout: Bun.file(output + '.server.log'), stderr: Bun.file(output + '.server.log') });
let stopping = false;
async function stop() {
  if (stopping) return;
  stopping = true; server.kill('SIGTERM');
  const timer = setTimeout(() => server.kill('SIGKILL'), 35000);
  try { await server.exited; } finally { clearTimeout(timer); }
}
process.on('SIGTERM', () => { void stop().then(() => process.exit(143)); });
process.on('SIGINT', () => { void stop().then(() => process.exit(130)); });
const state = { ticket: 'Production services are down and customers cannot connect.' };
const noul = { type: 'noul', instructions: 'Does this need urgent handling?' };
const choice = { type: 'choice', instructions: 'Choose the team.', criteria: { 'technical operations': 'Production outages', 'billing support': 'Invoice questions' } };
const score = { type: 'score', instructions: 'Rate incident severity.', criteria: ['Routine request', 'Degraded service', 'Total outage'] };
const workloads = [
  { label: 'Noul', questions: { urgent: noul } },
  { label: 'Choice', questions: { route: choice } },
  { label: 'Score', questions: { severity: score } },
  { label: 'Noul + choice + score', questions: { urgent: noul, route: choice, severity: score } },
];
const report: any = {
  schema: 'go-system-one-typesafe-benchmark-v1', revision, binary_sha256: hash(await readFile(binary)),
  model_sha256: '90fd944d227e9d9b68e7e2c7d5b57b79d4c66ed521b0919fbbd932cf834f6f8e',
  started_utc: new Date().toISOString(), command, mode: 'automatic', device: gpu(),
  method: 'One case warmup and five warm client HTTP samples; cool to <=55C before every call; abort at >=83C; 500ms GPU sampling. HTTP interval is fetch through JSON decode, excludes telemetry queries and loading. One structured state per request, no labelled accuracy evaluation.', cases: [],
};
async function save() { await writeFile(output + '.tmp', JSON.stringify(report, null, 2) + '\n'); await rename(output + '.tmp', output); }
try {
  let ready = false;
  for (let i = 0; i < 240; i++) {
    if (server.exitCode !== null) throw Error('server exited');
    try { const r = await fetch(url + '/go-system-one/v1/status', { signal: AbortSignal.timeout(500) }); if (r.ok) { report.status = await r.json(); ready = true; break; } } catch {}
    await Bun.sleep(1000);
  }
  if (!ready) throw Error('readiness timeout');
  for (const workload of workloads) {
    const request = { state, questions: workload.questions };
    const cell: any = { label: workload.label, request, request_sha256: hash(JSON.stringify(request)), status: 'running', trials: [] };
    report.cases.push(cell); await save();
    for (let i = 0; i < 6; i++) {
      await cool(); const before = gpu(); let peak = before.memory_mib, maxT = before.temperature, reason = '';
      const controller = new AbortController();
      const monitor = setInterval(() => {
        try { const g = gpu(); peak = Math.max(peak, g.memory_mib); maxT = Math.max(maxT, g.temperature); if (g.temperature >= 83) { reason = '83C thermal guard'; controller.abort(); } }
        catch (e) { reason = String(e); controller.abort(); }
      }, 500);
      const timer = setTimeout(() => { reason = 'request timeout'; controller.abort(); }, 120000);
      try {
        const start = performance.now();
        const r = await fetch(url + '/v1/systemone', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(request), signal: controller.signal });
        if (!r.ok) throw Error(`HTTP ${r.status}: ${await r.text()}`);
        const response: any = await r.json(), http_ms = performance.now() - start;
        const after = gpu(); maxT = Math.max(maxT, after.temperature); peak = Math.max(peak, after.memory_mib);
        if (maxT >= 83) throw Error('83C thermal guard');
        if (!(http_ms > 0) || Object.keys(response.answers || {}).length !== Object.keys(request.questions).length) throw Error('invalid response');
        for (const [name, q] of Object.entries(request.questions) as [string, any][]) {
          const a = response.answers[name]; if (a?.type !== q.type) throw Error('wrong answer type');
          if (q.type === 'noul') { if (!Number.isFinite(a.noul) || a.noul < 0 || a.noul > 1 || 'confidence' in a) throw Error('invalid noul'); }
          else {
            const p = Object.values(a.probabilities || {}) as number[];
            if (!p.length || p.some(v => !Number.isFinite(v) || v < 0 || v > 1) || Math.abs(p.reduce((s, v) => s + v, 0) - 1) > 1e-9) throw Error('invalid distribution');
            if (q.type === 'choice' && !(a.choice in q.criteria)) throw Error('invalid choice');
            if (q.type === 'score' && Math.abs(a.score - Object.entries(a.probabilities).reduce((s, [k, v]) => s + Number(k) * Number(v), 0)) > 1e-9) throw Error('invalid expectation');
          }
        }
        const trial = { http_ms, before, after, max_temperature_c: maxT, sampled_peak_memory_mib: peak, response };
        if (i === 0) cell.warmup = trial; else cell.trials.push(trial);
        await save();
      } catch (e) { cell.status = 'aborted'; cell.reason = reason || String(e); await save(); throw e; }
      finally { clearInterval(monitor); clearTimeout(timer); }
    }
    const samples = cell.trials.map((t: any) => t.http_ms).sort((a: number, b: number) => a - b);
    Object.assign(cell, { status: 'complete', n: samples.length, samples, min: samples[0], median: samples[2], max: samples[4], p95: samples[4], p99: samples[4] });
    await save(); console.log(JSON.stringify({ case: cell.label, median_http_ms: cell.median, min: cell.min, max: cell.max }));
  }
} finally { await stop(); }
