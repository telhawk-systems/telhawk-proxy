import type { Detector } from "./types";

export const apiMatrixDetector: Detector = {
  id: "api_matrix",
  run: () => {
    let apis = {};
    let inconsistencies: string[] = [];

    try {
      // Check modern Web API availability
      apis = {
        bluetooth: 'bluetooth' in navigator,
        credentials: 'credentials' in navigator,
        serviceWorker: 'serviceWorker' in navigator,
        share: 'share' in navigator,
        wakeLock: 'wakeLock' in navigator,
        usb: 'usb' in navigator,
        serial: 'serial' in navigator,
        hid: 'hid' in navigator,
        webNFC: 'nfc' in navigator,
        webLocks: 'locks' in navigator,
        presentation: 'presentation' in navigator,
        gamepad: 'getGamepads' in navigator,
        maxTouchPoints: navigator.maxTouchPoints || 0
      };

      const ua = navigator.userAgent.toLowerCase();
      const isMobile = /mobile|android|iphone|ipad|ipod/.test(ua);
      const isChrome = /chrome/.test(ua) && !/edge|edg/.test(ua);
      const isFirefox = /firefox/.test(ua);
      const isSafari = /safari/.test(ua) && !/chrome/.test(ua);

      // Collect inconsistencies for analysis (no scoring)
      
      // Mobile device claiming no touch support
      if (isMobile && navigator.maxTouchPoints === 0) {
        inconsistencies.push("mobile_no_touch");
      }

      // Chrome missing expected APIs
      if (isChrome && !(apis as any).serviceWorker) {
        inconsistencies.push("chrome_no_sw");
      }

      // Mobile device with desktop-only APIs
      if (isMobile && ((apis as any).usb || (apis as any).serial || (apis as any).hid)) {
        inconsistencies.push("mobile_desktop_apis");
      }

      // Desktop claiming mobile APIs
      if (!isMobile && (apis as any).webNFC) {
        inconsistencies.push("desktop_mobile_apis");
      }

      // Firefox with Chrome-specific APIs
      if (isFirefox && ((apis as any).bluetooth || (apis as any).usb || (apis as any).serial)) {
        inconsistencies.push("firefox_chrome_apis");
      }

      // Safari with non-Safari APIs
      if (isSafari && ((apis as any).bluetooth || (apis as any).usb || (apis as any).serial || (apis as any).hid)) {
        inconsistencies.push("safari_chrome_apis");
      }

    } catch (e) {
      inconsistencies.push("detection_error");
    }

    return { 
      id: "api_matrix", 
      score: 0, // No scoring, just raw data
      details: { 
        apis,
        inconsistencies,
        userAgent: {
          isMobile: /mobile|android|iphone|ipad|ipod/.test(navigator.userAgent.toLowerCase()),
          isChrome: /chrome/.test(navigator.userAgent.toLowerCase()) && !/edge|edg/.test(navigator.userAgent.toLowerCase()),
          isFirefox: /firefox/.test(navigator.userAgent.toLowerCase()),
          isSafari: /safari/.test(navigator.userAgent.toLowerCase()) && !/chrome/.test(navigator.userAgent.toLowerCase())
        }
      } 
    };
  }
};