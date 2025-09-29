import type { Detector } from "./types";
export const webdriverDetector: Detector = {
  id: "webdriver",
  run: () => {
    let webdriver = false;
    try { webdriver = !!(navigator as any).webdriver; } catch {}
    return { id: "webdriver", score: webdriver ? 2 : 0, details: { webdriver }, reliable: webdriver };
  }
};
