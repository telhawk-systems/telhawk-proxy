import { defaultConfig, type PixelConfig } from "./config";
import { readNav } from "./collect/nav";
import { readScreen } from "./collect/screen";
import { readDoc } from "./collect/doc";
import { readPerf } from "./collect/perf";
import { runDetectors } from "./detect";
import { toPayload } from "./api/payload";
import { pickEndpoint } from "./api/routes";
import { sendBeaconOrFetch } from "./transport/beacon";

export function init(cfg: Partial<PixelConfig> = {}) {
  const conf = { ...defaultConfig, ...cfg };
  try {
    const env = { nav: readNav(), screen: readScreen(), doc: readDoc(), perf: readPerf() };
    queueMicrotask(async () => {
      const det = await runDetectors();
      const payload = toPayload({ env, detectors: det.results, score: det.score, bucket: det.bucket });
      await sendBeaconOrFetch(JSON.stringify(payload), pickEndpoint(conf));
    });
  } catch { /* never break the page */ }
}
