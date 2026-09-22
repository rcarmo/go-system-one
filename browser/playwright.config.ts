import { defineConfig } from '@playwright/test';

export default defineConfig({
	testDir: './tests',
	fullyParallel: false,
	workers: 1,
	use: {
		baseURL: 'http://127.0.0.1:18181',
		headless: true,
		launchOptions: { args: ['--no-sandbox'] }
	},
	webServer: {
		command: 'cd .. && WEBUI_BROWSER_TEST=1 go test ./webui -run ^TestBrowserServer$ -count=1 -timeout=5m',
		url: 'http://127.0.0.1:18181/go-system-one/v1/status',
		timeout: 120_000,
		reuseExistingServer: false
	}
});
