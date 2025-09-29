import type { Detector } from "./types";

export const webglDetector: Detector = {
  id: "webgl",
  run: () => {
    let vendor = "";
    let renderer = "";
    let unmaskedVendor = "";
    let unmaskedRenderer = "";
    let version = "";
    let shadingLanguageVersion = "";
    let suspicious = false;

    try {
      const canvas = document.createElement("canvas");
      const gl: any = canvas.getContext("webgl") || canvas.getContext("experimental-webgl");
      
      if (gl) {
        // Get basic WebGL info
        vendor = gl.getParameter(gl.VENDOR) || "";
        renderer = gl.getParameter(gl.RENDERER) || "";
        version = gl.getParameter(gl.VERSION) || "";
        shadingLanguageVersion = gl.getParameter(gl.SHADING_LANGUAGE_VERSION) || "";
        
        // Try to get unmasked vendor/renderer for more detailed analysis
        const debugInfo = gl.getExtension("WEBGL_debug_renderer_info");
        if (debugInfo) {
          unmaskedVendor = gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL) || "";
          unmaskedRenderer = gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) || "";
        }
        
        // Check for automation signatures in all collected data
        const combined = (vendor + renderer + unmaskedVendor + unmaskedRenderer).toLowerCase();
        
        suspicious = /swiftshader|llvmpipe|headless|angle|software|mesa/i.test(combined) ||
                    vendor === "" ||
                    renderer === "" ||
                    combined.includes("google inc.") && combined.includes("angle");
      } else {
        suspicious = true; // No WebGL support is very suspicious
      }
    } catch {
      suspicious = true;
    }

    return { 
      id: "webgl", 
      score: suspicious ? 2 : 0, 
      details: { 
        vendor,
        renderer,
        unmaskedVendor,
        unmaskedRenderer,
        version,
        shadingLanguageVersion,
        suspicious,
        available: vendor !== "" || renderer !== ""
      } 
    };
  }
};