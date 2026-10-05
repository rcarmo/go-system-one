import { defineConfig } from '@playwright/test';
import { execFileSync } from 'node:child_process';

// Direct `bun x playwright test` uses the same environment as Make.
const env = JSON.parse(execFileSync('bash', ['../scripts/project-env.sh', 'node', '-e', 'console.log(JSON.stringify(process.env))'], { encoding: 'utf8' }));
for (const key of ['PROJECT_RUN_DIR', 'TMPDIR', 'TMP', 'TEMP', 'PLAYWRIGHT_BROWSERS_PATH', 'PROFILE_ROOT']) process.env[key] = env[key];

export default defineConfig({
	testDir: './tests',
	outputDir: `${process.env.PROJECT_RUN_DIR}/playwright`,
	reporter: [['list']],
	fullyParallel: false,
	workers: 1,
	use: {
		baseURL: 'http://127.0.0.1:18181',
		headless: true,
		launchOptions: { args: ['--no-sandbox'] }
	},
	webServer: {
		command: 'cd .. && exec ./scripts/browser-server.sh',
		gracefulShutdown: { signal: 'SIGTERM', timeout: 30_000 },
		url: 'http://127.0.0.1:18181/go-system-one/v1/status',
		timeout: 120_000,
		reuseExistingServer: false
	}
});
