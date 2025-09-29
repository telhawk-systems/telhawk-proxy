import type { Detector } from "./types";
export const functionToStringDetector: Detector = {
  id: "fn_to_string",
  run: () => {
    let sample = "";
    try { sample = Function.prototype.toString.call(() => {}); } catch {}
    const patched = /puppeteer|webdriver|selenium|cdp/i.test(sample);
    return { id: "fn_to_string", score: patched ? 1 : 0, details: { sample: sample.slice(0, 60) } };
  }
};
