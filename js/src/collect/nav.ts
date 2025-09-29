export const readNav = () => {
  if (typeof navigator === "undefined") return {};
  
  const nav: any = {};
  
  try {
    nav.ua = navigator.userAgent || "";
    nav.lang = navigator.language || "";
    nav.langs = (navigator.languages || []).slice(0, 10); // Get more languages for analysis
    nav.plat = (navigator as any).platform || "";
    nav.hc = (navigator as any).hardwareConcurrency ?? null;
    nav.vendor = (navigator as any).vendor || "";
    nav.appName = navigator.appName || "";
    nav.appVersion = navigator.appVersion || "";
    nav.oscpu = (navigator as any).oscpu || "";
    nav.buildID = (navigator as any).buildID || "";
    nav.product = (navigator as any).product || "";
    nav.productSub = (navigator as any).productSub || "";
    nav.maxTouchPoints = (navigator as any).maxTouchPoints ?? null;
    nav.cookieEnabled = navigator.cookieEnabled ?? null;
    nav.onLine = navigator.onLine ?? null;
    nav.doNotTrack = navigator.doNotTrack || "";
    nav.globalPrivacyControl = (navigator as any).globalPrivacyControl ?? null;
    
    // Check for automation-specific properties
    nav.webdriver = (navigator as any).webdriver ?? null;
    nav.automation = (navigator as any).automation ?? null;
    nav.permissions = typeof (navigator as any).permissions !== "undefined";
    nav.bluetooth = typeof (navigator as any).bluetooth !== "undefined";
    nav.credentials = typeof (navigator as any).credentials !== "undefined";
    nav.mediaDevices = typeof (navigator as any).mediaDevices !== "undefined";
    nav.serviceWorker = typeof (navigator as any).serviceWorker !== "undefined";
    
    // Collect plugin info (for security analysis)
    nav.pluginCount = navigator.plugins?.length ?? 0;
    nav.mimeTypeCount = navigator.mimeTypes?.length ?? 0;
    
  } catch (e) {
    nav.error = String(e);
  }
  
  return nav;
};
