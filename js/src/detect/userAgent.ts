import type { Detector } from "./types";

export const uaDetector: Detector = {
  id: "ua",
  run: () => {
    let ua = "";
    let platform = "";
    let userAgentData: any = null;
    let oscpu = "";
    let appVersion = "";
    let appName = "";
    let vendor = "";
    let suspicious = false;

    try {
      ua = (typeof navigator !== "undefined" && navigator.userAgent) ? navigator.userAgent : "";
      platform = (typeof navigator !== "undefined" && (navigator as any).platform) || "";
      oscpu = (typeof navigator !== "undefined" && (navigator as any).oscpu) || "";
      appVersion = (typeof navigator !== "undefined" && navigator.appVersion) || "";
      appName = (typeof navigator !== "undefined" && navigator.appName) || "";
      vendor = (typeof navigator !== "undefined" && (navigator as any).vendor) || "";
      
      // Collect modern User-Agent Client Hints if available
      if (typeof navigator !== "undefined" && (navigator as any).userAgentData) {
        userAgentData = {
          brands: (navigator as any).userAgentData.brands || [],
          mobile: (navigator as any).userAgentData.mobile,
          platform: (navigator as any).userAgentData.platform || ""
        };
      }

      // Check for automation tool signatures
      const uaLower = ua.toLowerCase();
      const platformLower = platform.toLowerCase();
      
      suspicious = /headlesschrome|phantomjs|puppeteer|playwright|selenium|webdriver|chromedriver|automation/i.test(ua) ||
                  // Platform inconsistencies  
                  (platformLower.includes("win") && !uaLower.includes("windows")) ||
                  (platformLower.includes("mac") && uaLower.includes("windows")) ||
                  (platformLower.includes("linux") && uaLower.includes("windows")) ||
                  // Empty or minimal UA
                  ua.length < 20 ||
                  // Missing expected fields
                  (ua && !platform);
                  
    } catch {}

    return { 
      id: "ua", 
      score: suspicious ? 2 : 0, 
      details: { 
        ua,
        platform,
        oscpu,
        appVersion,
        appName,
        vendor,
        userAgentData,
        suspicious,
        uaLength: ua.length
      } 
    };
  }
};