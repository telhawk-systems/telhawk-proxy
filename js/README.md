# GoTrack JS Pixel

This is the **client-side tracking pixel library** written in TypeScript.  
It collects environment signals, runs bot-detection heuristics, and sends events to the GoTrack collector (`/px.gif` or `/collect`).

The structure may look like a lot of files, but each directory has a focused purpose:

---

## 📂 Project Layout

```js/
package.json # deps, build scripts
tsconfig.json # TS compiler options
rollup.config.mjs # bundler config (UMD + ESM builds)

src/
index.ts # entrypoint; boot + init()
config.ts             # runtime config schema + defaults

ids/                  # identifiers and consent
  session.ts          # sid/uid management
  consent.ts          # DNT / GPC / CMP hooks

transport/            # network sending logic
  beacon.ts           # sendBeacon (preferred)
  fetch.ts            # fetch/XHR fallback
  img.ts              # <img src> pixel fallback
  batch.ts            # batching and compression
  sign.ts             # optional HMAC signing

collect/              # passive environment signals
  nav.ts              # UA, lang, tz, platform
  screen.ts           # resolution, DPR, available rects
  perf.ts             # coarse nav/perf timings
  doc.ts              # referrer, visibility, focus
  input.ts            # pointer/keyboard entropy (rate-only)

detect/               # active bot-detection heuristics
  types.ts            # Detector interfaces
  registry.ts         # registry runner (aggregates detectors)
  index.ts            # exports `runDetectors()`
  webdriver.ts        # navigator.webdriver, Selenium globals
  plugins.ts          # plugins/mimeTypes empty check
  functionToString.ts # patched Function.toString() anomalies
  permissions.ts      # Permissions API quirks
  webgl.ts            # vendor/renderer anomalies (SwiftShader, ANGLE)
  userAgent.ts        # UA / platform consistency

utils/                # safe wrappers & helpers
  dom.ts              # DOM-safe access
  time.ts             # monotonic clock helpers
  rand.ts             # CSPRNG + jitter
  guards.ts           # schema & size guards
  utils.ts            # safe() / timeout()

api/                  # payload shaping & endpoint routing
  payload.ts          # normalize detectors/env into canonical JSON
  routes.ts           # endpoint chooser & path randomizer
  test/
unit/ # Jest-style unit tests
detect.spec.ts
e2e/ # Playwright headless vs headful checks
pixel.spec.ts

README.md # (this file)
```
---

## 🧩 Philosophy

- **Small core, pluggable parts.** Each detector is independent and scored. Registry aggregates into `{ score, bucket }`.
- **Privacy-aware.** Detectors return *booleans/flags*, not raw fingerprints (e.g., `"pluginsLen=0"`, not full plugin names).
- **Transport fallback ladder.** Try `sendBeacon → fetch → img`.
- **Non-breaking.** Pixel never throws; page should load even if detectors fail.
- **Explainable.** Payload contains structured detector results so server can apply rules.

---

## 🚦 Development

### 1. Install & build
```bash
cd js
npm install
npm run build    # runs rollup → dist/esm + dist/umd
```

### 2. Test

```bash
npm run test     # runs unit tests
npm run e2e      # runs Playwright tests (headless vs headful)
```

### 3. Run in a page
```html
<script src="/dist/pixel.umd.js"></script>
<script>
  GoTrackPixel.init({ endpoint: "https://your-collector/collect" });
</script>
```

### 🕵️‍♂️ Adding a Detector
* Create a new file in src/detect/ (e.g. canvas.ts).
* Export a Detector object:
```ts
import type { Detector } from "./types";
export const canvasDetector: Detector = {
  id: "canvas",
  run: () => {
    // probe here...
    return { id: "canvas", score: suspicious ? 1 : 0, details: {...} };
  }
}
```
### 📦 Payload Shape
Example payload sent to `/collect`:
```
{
  "ver": 1,
  "ts": 1696035000000,
  "env": {
    "ua": "...",
    "lang": "en-US",
    "screen": { "w": 1920, "h": 1080, "dpr": 2 }
  },
  "detectors": [
    { "id": "webdriver", "score": 2, "details": { "webdriver": true } },
    { "id": "plugins", "score": 1, "details": { "pluginsLen": 0 } }
  ],
  "score": 3,
  "bucket": "high"
}
```

### 🛣️ Roadmap
* More detectors (mobile emulation, timezone/locale mismatches, audio context quirks)

* Lightweight bundling for <script> drop-in (5–8 KB gzipped)

*  Optional “strict privacy” mode (drop IP, obey DNT fully)

* Demo dashboard with real detector scores

### ⚠️ Note
This repo is **educational & security-focused**.
Collected signals are coarse, non-invasive, and only used for **fraud/bot defense**.
