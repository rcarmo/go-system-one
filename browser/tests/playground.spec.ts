import { test, expect } from '@playwright/test';

async function assertNoOverflow(page: import('@playwright/test').Page) {
	expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
}

test('submits bounded decisions and follows the OS colour scheme', async ({ page }) => {
	const errors: string[] = [];
	page.on('pageerror', (error) => errors.push(error.message));
	await page.emulateMedia({ colorScheme: 'light' });
	await page.goto('/go-system-one');
	await expect(page.getByRole('heading', { name: 'Decision playground', exact: true })).toBeVisible();
	await expect(page.locator('#status')).toContainText('ready');
	await page.getByRole('button', { name: 'Run decision' }).click();
	await expect(page.locator('.decision')).toHaveCount(2);
	await expect(page.locator('.decision').first()).toContainText('"urgent": true');
	await expect(page.locator('.decision').nth(1)).toContainText('"urgent": false');
	await expect(page.locator('.prob')).toHaveCount(2);
	await expect(page.locator('.prob').first()).toHaveText('87.50%');
	await expect(page.locator('#timings')).toContainText('total: 3.50 ms');
	await assertNoOverflow(page);

	const light = await page.evaluate(() => ({
		colourScheme: getComputedStyle(document.documentElement).colorScheme,
		font: getComputedStyle(document.body).fontFamily,
		textareaFont: getComputedStyle(document.querySelector('textarea')!).fontFamily,
		panelRadius: getComputedStyle(document.querySelector('.panel')!).borderRadius,
		background: getComputedStyle(document.body).backgroundColor
	}));
	expect(light.colourScheme).toBe('light');
	expect(light.font).toContain('ui-sans-serif');
	expect(light.textareaFont).toContain('ui-monospace');
	expect(light.panelRadius).toBe('14px');

	await page.setViewportSize({ width: 390, height: 844 });
	await page.emulateMedia({ colorScheme: 'dark' });
	await page.goto('/go-system-one');
	await page.getByRole('button', { name: 'Run decision' }).click();
	await expect(page.locator('.decision')).toHaveCount(2);
	await assertNoOverflow(page);
	expect(await page.evaluate(() => getComputedStyle(document.documentElement).colorScheme)).toBe('dark');
	expect(await page.evaluate(() => getComputedStyle(document.body).backgroundColor)).not.toBe(light.background);
	expect(await page.locator('.grid').evaluate((element) => getComputedStyle(element).gridTemplateColumns.split(' ').length)).toBe(1);
	expect(errors).toEqual([]);
});

test('keeps status and method boundaries intact', async ({ request }) => {
	const status = await request.get('/go-system-one/v1/status');
	expect(status.ok()).toBe(true);
	expect(await status.json()).toMatchObject({
		model: 'fixture-model',
		backend: 'fixture',
		device: 'synthetic',
		busy: false
	});
	const method = await request.post('/go-system-one/v1/status');
	expect(method.status()).toBe(405);
	expect(method.headers().allow).toBe('GET, HEAD');
});
