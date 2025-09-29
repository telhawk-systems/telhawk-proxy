import type { Detector } from "./types";
export const uaDetector: Detector = {
  id: "ua",
  run: () => {
    const ua = (typeof navigator !== "undefined" && navigator.userAgent) ? navigator.userAgent : "";
    const platform = (typeof navigator !== "undefined" && (navigator as any).platform) || "";
    const suspicious = /HeadlessChrome|PhantomJS|Puppeteer|Playwright/i.test(ua);
    return { id: "ua", score: suspicious ? 1 : 0, details: { ua, platform } };
  }
};