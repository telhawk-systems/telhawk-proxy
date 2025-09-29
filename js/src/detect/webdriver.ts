import type { Detector } from "./types";

export const webdriverDetector: Detector = {
  id: "webdriver",
  run: () => {
    let webdriver = false;
    let automation = false;
    let chromeRuntime = false;
    let seleniumGlobals = false;
    let phantomGlobals = false;
    let suspicious = false;
    const globals: string[] = [];

    try {
      // Check navigator.webdriver
      webdriver = !!(navigator as any).webdriver;
      
      // Check for automation flags
      automation = !!(navigator as any).automation;
      
      // Check for Chrome runtime (missing in headless)
      chromeRuntime = typeof (window as any).chrome !== "undefined" && 
                     typeof (window as any).chrome.runtime !== "undefined";
      
      // Check for Selenium global variables
      const seleniumVars = ['__selenium_unwrapped', '__webdriver_evaluate', '__selenium_evaluate', 
                           '__fxdriver_unwrapped', '__driver_evaluate', '__webdriver_script_fn'];
      
      seleniumVars.forEach(varName => {
        if ((window as any)[varName] !== undefined) {
          seleniumGlobals = true;
          globals.push(varName);
        }
      });
      
      // Check for PhantomJS globals
      const phantomVars = ['__phantomas', '_phantom', 'callPhantom'];
      phantomVars.forEach(varName => {
        if ((window as any)[varName] !== undefined) {
          phantomGlobals = true;
          globals.push(varName);
        }
      });
      
      // Check for other automation indicators
      const otherIndicators = ['cdc_adoQpoasnfa76pfcZLmcfl_Array', 'cdc_adoQpoasnfa76pfcZLmcfl_Promise',
                              'cdc_adoQpoasnfa76pfcZLmcfl_Symbol', '$chrome_asyncScriptInfo', '$cdc_asdjflasutopfhvcZLmcfl_'];
      
      otherIndicators.forEach(indicator => {
        if ((window as any)[indicator] !== undefined) {
          globals.push(indicator);
        }
      });
      
      suspicious = webdriver || automation || seleniumGlobals || phantomGlobals || 
                  globals.length > 0 || 
                  // Missing chrome runtime in Chrome is suspicious
                  (navigator.userAgent.includes("Chrome") && !chromeRuntime);
      
    } catch {}

    return { 
      id: "webdriver", 
      score: suspicious ? 3 : 0, // High score for automation detection
      details: { 
        webdriver, 
        automation,
        chromeRuntime,
        seleniumGlobals,
        phantomGlobals,
        globals: globals.slice(0, 10), // Limit to prevent huge payloads
        suspicious
      },
      reliable: webdriver || automation || globals.length > 0
    };
  }
};
