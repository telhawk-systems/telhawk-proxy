import type { Detector } from "./types";
export const pluginsDetector: Detector = {
  id: "plugins",
  run: () => {
    let p = 0, m = 0;
    try { p = navigator.plugins?.length ?? 0; } catch {}
    try { m = navigator.mimeTypes?.length ?? 0; } catch {}
    const suspicious = p === 0 && m === 0;
    return { id: "plugins", score: suspicious ? 1 : 0, details: { pluginsLen: p, mimeLen: m } };
  }
};