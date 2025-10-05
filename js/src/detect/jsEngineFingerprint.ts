import type { Detector } from "./types";

export const jsEngineFingerprintDetector: Detector = {
  id: "js_engine",
  run: () => {
    let errorStackFormat = "";
    let functionToStringLength = 0;
    let performanceNowPrecision = 0;
    let mathConstantLength = 0;
    let regexUnicodeSupport = false;
    let arrayStringifyBehavior = "";
    let objectToStringResult = "";
    let suspicious = false;
    let engineSignatures: string[] = [];

    try {
      // Error stack format (V8, SpiderMonkey, JavaScriptCore have different formats)
      try {
        throw new Error("test");
      } catch (e: any) {
        errorStackFormat = (e.stack || "").slice(0, 50);
        
        // Check for automation tool signatures in stack traces
        if (errorStackFormat.includes("puppeteer") || 
            errorStackFormat.includes("webdriver") ||
            errorStackFormat.includes("selenium")) {
          engineSignatures.push("automation_in_stack");
          suspicious = true;
        }
      }

      // Function.toString behavior
      functionToStringLength = Function.prototype.toString.toString().length;
      
      // V8: ~29, SpiderMonkey: ~37, JavaScriptCore: ~35 (approximate)
      if (functionToStringLength < 20 || functionToStringLength > 50) {
        engineSignatures.push("unusual_function_toString_length");
      }

      // Performance.now() precision varies by engine and privacy settings
      performanceNowPrecision = performance.now() % 1;
      
      // Math constant precision (engines may differ)
      mathConstantLength = Math.PI.toString().length;
      if (mathConstantLength !== 17) { // Standard JS should be 17 chars
        engineSignatures.push("unusual_math_pi_precision");
      }

      // Regex Unicode support
      try {
        regexUnicodeSupport = /\u{1F4A9}/u.test('💩');
        if (!regexUnicodeSupport) {
          engineSignatures.push("no_unicode_regex");
        }
      } catch {
        engineSignatures.push("unicode_regex_error");
      }

      // Array stringify behavior differences
      try {
        const arr = [undefined, null, 1];
        arrayStringifyBehavior = JSON.stringify(arr);
        // Should be "[null,null,1]" in standard browsers
        if (arrayStringifyBehavior !== "[null,null,1]") {
          engineSignatures.push("unusual_array_stringify");
        }
      } catch {
        engineSignatures.push("array_stringify_error");
      }

      // Object.prototype.toString behavior
      objectToStringResult = Object.prototype.toString.call(window);
      // Should be "[object Window]" in browsers
      if (!objectToStringResult.includes("Window") && typeof window !== "undefined") {
        engineSignatures.push("unusual_window_toString");
      }

      // Check for engine-specific globals that shouldn't be there
      const v8Globals = ['%DebugPrint', '%GetOptimizationStatus'];
      const spiderMonkeyGlobals = ['uneval', 'toSource'];
      const nodeGlobals = ['global', 'process', 'Buffer'];
      
      v8Globals.forEach(global => {
        if ((window as any)[global] !== undefined) {
          engineSignatures.push("v8_debug_global");
          suspicious = true;
        }
      });

      spiderMonkeyGlobals.forEach(global => {
        if ((window as any)[global] !== undefined) {
          engineSignatures.push("spidermonkey_global");
        }
      });

      nodeGlobals.forEach(global => {
        if ((window as any)[global] !== undefined) {
          engineSignatures.push("node_global");
          suspicious = true;
        }
      });

      // Check for automation-modified prototypes
      const originalToString = Function.prototype.toString;
      try {
        if (originalToString.toString().includes("native code") === false) {
          engineSignatures.push("modified_function_prototype");
          suspicious = true;
        }
      } catch {}

      // Overall suspicion
      if (engineSignatures.length > 2) {
        suspicious = true;
      }

    } catch (e) {
      suspicious = true;
      engineSignatures.push("detection_error");
    }

    return { 
      id: "js_engine", 
      score: suspicious ? Math.min(3, engineSignatures.length) : 0, 
      details: { 
        errorStackFormat,
        functionToStringLength,
        performanceNowPrecision,
        mathConstantLength,
        regexUnicodeSupport,
        arrayStringifyBehavior,
        objectToStringResult,
        engineSignatures,
        suspicious,
        signatureCount: engineSignatures.length
      } 
    };
  }
};