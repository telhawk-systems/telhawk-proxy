import type { Detector } from "./types";
export const webglDetector: Detector = {
  id: "webgl",
  run: () => {
    try {
      const c = document.createElement("canvas");
      const gl: any = c.getContext("webgl") || c.getContext("experimental-webgl");
      if (!gl) return { id: "webgl", score: 0, details: { available: false } };
      const info = gl.getExtension("WEBGL_debug_renderer_info");
      const vendor = info ? gl.getParameter(info.UNMASKED_VENDOR_WEBGL) : null;
      const renderer = info ? gl.getParameter(info.UNMASKED_RENDERER_WEBGL) : null;
      const suspicious = /SwiftShader|Software|llvmpipe|Headless|ANGLE/i.test(String(vendor) + String(renderer));
      return { id: "webgl", score: suspicious ? 1 : 0, details: { vendor, renderer } };
    } catch { return { id: "webgl", score: 0 }; }
  }
};