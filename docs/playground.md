# Decision playground

The embedded playground at `/go-system-one` provides a browser form for the `POST /v1/decision` API. It uses the same loopback service and does not add a second inference endpoint.

## Appearance

The page follows the neutral OKLCH palette, typography and controls used by the ported llama-ui. System text is used for labels and prose; monospace text is limited to structured values and code-like output.

Appearance follows the operating-system `prefers-color-scheme` setting. The page has no theme toggle, local-storage key or persisted theme override.

### Light desktop

![Go System One playground in light mode on desktop](images/go-system-one-light-desktop.png)

- OS preference: light
- viewport and output: 1440 × 1100
- device pixel ratio: 1
- SHA-256: `20efa66f80f963fefa40175b1756dac9caa03fb6d134a0da8a3c5d1c3a736631`

### Dark mobile

![Go System One playground in dark mode on mobile](images/go-system-one-dark-mobile.png)

- OS preference: dark
- viewport: 390 × 844
- full-page output: 390 × 1588
- device pixel ratio: 1
- content width: 370 px, without horizontal overflow
- SHA-256: `6aeb50d478ad3b683adf0bd044e177364c7f68ab054ccb47044b1b1413735317`

## Standalone browser checks

The repository has a model-free Playwright project under `browser/`. Its loopback-only Go fixture serves the real embedded page and synthetic decision responses; no checkpoint is loaded.

```sh
PLAYWRIGHT_BROWSERS_PATH=/workspace/.cache/ms-playwright make browser-test
```

The tests submit two contexts, check rendered decisions and selected probabilities, exercise light and dark OS preferences, reject mobile horizontal overflow, and verify status/method boundaries. CI runs the same Chromium suite. Candidate-by-candidate probability assertions will accompany the API/UI update that exposes complete distributions.

## Capture provenance

The page source was imported from [`go-pherence@8629232b14440f4a9aa06cfb6d6003c1302c8cb9`](https://github.com/rcarmo/go-pherence/commit/8629232b14440f4a9aa06cfb6d6003c1302c8cb9). Screenshots used the loopback-only synthetic browser fixture. No model or user data was loaded.

For each colour scheme, Playwright opened `/go-system-one`, clicked **Run decision**, waited for the first `.decision` result and captured the full page. Source validation passed:

```sh
cd webui/frontend
bun x playwright test --config playwright.go.config.ts --grep 'Go System One playground'
go test ./webui
go vet ./webui
git diff --check
```

The upstream browser suite passed all three scenarios used for the original hand-off. The standalone Playwright harness covers this page independently without carrying the Svelte/chat frontend.
