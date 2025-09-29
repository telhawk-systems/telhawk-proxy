import type { Detector } from "./types";

export const functionToStringDetector: Detector = {
  id: "fn_to_string",
  run: () => {
    let nativeSample = "";
    let customSample = "";
    let toStringSample = "";
    let suspicious = false;

    try {
      // Test native function toString
      nativeSample = Function.prototype.toString.call(() => {});
      
      // Test if toString itself has been modified
      toStringSample = Function.prototype.toString.toString();
      
      // Test a built-in function
      customSample = Function.prototype.toString.call(Array.prototype.push);
      
      // Look for automation tool signatures in the output
      const combined = (nativeSample + toStringSample + customSample).toLowerCase();
      
      suspicious = /puppeteer|webdriver|selenium|cdp|chrome.?devtools|automation|phantom/i.test(combined) ||
                  nativeSample.includes("Illegal invocation") ||
                  nativeSample.length < 10 ||
                  !nativeSample.includes("native code") ||
                  toStringSample.includes("Illegal invocation");
                  
    } catch (e) {
      suspicious = true;
      nativeSample = `Error: ${e}`;
    }

    return { 
      id: "fn_to_string", 
      score: suspicious ? 2 : 0, 
      details: { 
        nativeSample: nativeSample.slice(0, 100),
        toStringSample: toStringSample.slice(0, 100), 
        customSample: customSample.slice(0, 100),
        suspicious
      } 
    };
  }
};
