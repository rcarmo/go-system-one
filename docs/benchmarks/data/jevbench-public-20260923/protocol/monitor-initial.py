import subprocess,time,json,sys,os,signal
from pathlib import Path
root=Path('/workspace/tmp/gso-jevbench')
server=int(sys.argv[1]); runner_file=Path(sys.argv[2])
with (root/'gpu-telemetry.jsonl').open('a') as f:
    while True:
        try:
            text=subprocess.check_output(['nvidia-smi','-i','0','--query-gpu=temperature.gpu,memory.used,clocks.sm','--format=csv,noheader,nounits'],text=True,timeout=5)
            t,m,c=map(int,text.strip().split(','));f.write(json.dumps({'ts':time.time(),'temperature_c':t,'memory_mib':m,'sm_mhz':c})+'\n');f.flush()
            if t<83:
                time.sleep(.5);continue
            reason=f'thermal guard {t}C'
        except Exception as e: reason=f'telemetry failure: {e}'
        (root/'ABORTED').write_text(reason+'\n')
        os.kill(server,signal.SIGTERM)
        try: os.kill(int(runner_file.read_text()),signal.SIGTERM)
        except (FileNotFoundError,ProcessLookupError): pass
        sys.exit(70)
