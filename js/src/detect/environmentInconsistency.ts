import type { Detector } from "./types";

export const environmentInconsistencyDetector: Detector = {
  id: "env_inconsistency",
  run: () => {
    let screenInconsistencies: string[] = [];
    let localeInconsistencies: string[] = [];
    let timingInconsistencies: string[] = [];
    let suspicious = false;

    try {
      // Screen/viewport inconsistencies
      const screen = window.screen;
      const devicePixelRatio = window.devicePixelRatio || 1;
      const innerWidth = window.innerWidth;
      const innerHeight = window.innerHeight;

      // Check screen dimension consistency
      if (screen.availWidth > screen.width) {
        screenInconsistencies.push("avail_width_larger");
      }
      
      if (screen.availHeight > screen.height) {
        screenInconsistencies.push("avail_height_larger");
      }

      // Check device pixel ratio consistency
      if (devicePixelRatio % 0.25 !== 0 || devicePixelRatio > 4 || devicePixelRatio < 0.5) {
        screenInconsistencies.push("unusual_dpr");
      }

      // Check orientation consistency
      if (screen.orientation) {
        const isLandscape = screen.width > screen.height;
        const orientationSaysLandscape = screen.orientation.type.includes('landscape');
        if (isLandscape !== orientationSaysLandscape) {
          screenInconsistencies.push("orientation_mismatch");
        }
      }

      // Viewport vs screen consistency
      if (innerWidth > screen.width || innerHeight > screen.height) {
        screenInconsistencies.push("viewport_larger_than_screen");
      }

      // Locale/timezone inconsistencies
      const now = new Date();
      const timezoneOffset = now.getTimezoneOffset();
      const languages = navigator.languages || [navigator.language];
      const primaryLang = navigator.language;
      
      let resolvedTimezone = "";
      try {
        resolvedTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
      } catch {}

      // Check language consistency
      if (languages.length === 0 || !primaryLang) {
        localeInconsistencies.push("missing_languages");
      }

      if (languages.length === 1 && primaryLang !== languages[0]) {
        localeInconsistencies.push("language_array_mismatch");
      }

      // Check timezone/language geographic consistency
      if (primaryLang && resolvedTimezone) {
        const langRegion = primaryLang.split('-')[1]?.toLowerCase();
        const timezone = resolvedTimezone.toLowerCase();
        
        // Some basic geographic inconsistency checks
        if (langRegion === 'us' && timezone.includes('europe')) {
          localeInconsistencies.push("us_lang_europe_tz");
        }
        if (langRegion === 'gb' && timezone.includes('america')) {
          localeInconsistencies.push("gb_lang_america_tz");
        }
        if (primaryLang.startsWith('zh') && timezone.includes('america') && !timezone.includes('los_angeles')) {
          localeInconsistencies.push("chinese_lang_america_tz");
        }
      }

      // Performance timing inconsistencies
      const perfNow = performance.now();
      const perfNowPrecision = perfNow % 1;
      
      // Check performance.now() precision (varies by browser and privacy settings)
      if (perfNowPrecision === 0 && perfNow > 100) {
        timingInconsistencies.push("perfect_timing_precision");
      }

      // Check Date vs performance timing consistency
      const dateNow = Date.now();
      const perfTimeOrigin = performance.timeOrigin || 0;
      const calculatedNow = perfTimeOrigin + perfNow;
      const timeDiff = Math.abs(dateNow - calculatedNow);
      
      if (timeDiff > 1000) { // More than 1 second difference
        timingInconsistencies.push("date_performance_mismatch");
      }

      // Overall suspicion calculation
      const totalInconsistencies = screenInconsistencies.length + localeInconsistencies.length + timingInconsistencies.length;
      suspicious = totalInconsistencies > 2;

    } catch (e) {
      suspicious = true;
    }

    return { 
      id: "env_inconsistency", 
      score: suspicious ? Math.min(3, screenInconsistencies.length + localeInconsistencies.length + timingInconsistencies.length) : 0, 
      details: { 
        screen: {
          width: window.screen.width,
          height: window.screen.height,
          availWidth: window.screen.availWidth,
          availHeight: window.screen.availHeight,
          devicePixelRatio: window.devicePixelRatio,
          innerWidth: window.innerWidth,
          innerHeight: window.innerHeight,
          inconsistencies: screenInconsistencies
        },
        locale: {
          language: navigator.language,
          languages: navigator.languages,
          timezoneOffset: new Date().getTimezoneOffset(),
          resolvedTimezone: (() => {
            try { return Intl.DateTimeFormat().resolvedOptions().timeZone; }
            catch { return ""; }
          })(),
          inconsistencies: localeInconsistencies
        },
        timing: {
          performanceNow: performance.now(),
          performanceNowPrecision: performance.now() % 1,
          timeOrigin: performance.timeOrigin || 0,
          inconsistencies: timingInconsistencies
        },
        suspicious,
        totalInconsistencies: screenInconsistencies.length + localeInconsistencies.length + timingInconsistencies.length
      } 
    };
  }
};