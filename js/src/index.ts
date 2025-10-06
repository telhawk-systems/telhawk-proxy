import { defaultConfig, type PixelConfig } from "./config";
import { readNav } from "./collect/nav";
import { readScreen } from "./collect/screen";
import { readDoc } from "./collect/doc";
import { readPerf } from "./collect/perf";
import { readInputEntropy } from "./collect/input";
import { getSessionId } from "./ids/session";
import { readConsent } from "./ids/consent";
import { runDetectors } from "./detect";
import { toPayload } from "./api/payload";
import { pickEndpoint } from "./api/routes";
import { sendBeaconOrFetch } from "./transport/beacon";

export function init(cfg: Partial<PixelConfig> = {}) {
  const conf = { ...defaultConfig, ...cfg };
  try {
    const env = { 
      nav: readNav(), 
      screen: readScreen(), 
      doc: readDoc(), 
      perf: readPerf(),
      input: readInputEntropy(),
      session: { sid: getSessionId() },
      consent: readConsent() // Still collect for analysis, but don't block
    };
    
    queueMicrotask(async () => {
      const det = await runDetectors();
      const payload = toPayload({ env, detectors: det.results, score: det.score, bucket: det.bucket });
      await sendBeaconOrFetch(JSON.stringify(payload), pickEndpoint(conf), conf.secret);
    });
  } catch { /* never break the page */ }
}

// Auto-initialize if window exists and auto-init is not disabled
if (typeof window !== 'undefined' && !(window as any).GO_TRACK_NO_AUTO_INIT) {
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => init());
  } else {
    // Document already loaded
    init();
  }
}
