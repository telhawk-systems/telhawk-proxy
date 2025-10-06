// Note: endpoint will default to window.GO_TRACK_URL or current page path
// This allows tracking data to be posted to any URL for ad-blocker evasion
const defaultConfig = {
    endpoint: undefined, // Will be determined by pickEndpoint()
    batchSize: 10,
    timeout: 5000
};

const readNav = () => {
    if (typeof navigator === "undefined")
        return {};
    const nav = {};
    try {
        nav.ua = navigator.userAgent || "";
        nav.lang = navigator.language || "";
        nav.langs = (navigator.languages || []).slice(0, 10); // Get more languages for analysis
        nav.plat = navigator.platform || "";
        nav.hc = navigator.hardwareConcurrency ?? null;
        nav.vendor = navigator.vendor || "";
        nav.appName = navigator.appName || "";
        nav.appVersion = navigator.appVersion || "";
        nav.oscpu = navigator.oscpu || "";
        nav.buildID = navigator.buildID || "";
        nav.product = navigator.product || "";
        nav.productSub = navigator.productSub || "";
        nav.maxTouchPoints = navigator.maxTouchPoints ?? null;
        nav.cookieEnabled = navigator.cookieEnabled ?? null;
        nav.onLine = navigator.onLine ?? null;
        nav.doNotTrack = navigator.doNotTrack || "";
        nav.globalPrivacyControl = navigator.globalPrivacyControl ?? null;
        // Check for automation-specific properties
        nav.webdriver = navigator.webdriver ?? null;
        nav.automation = navigator.automation ?? null;
        nav.permissions = typeof navigator.permissions !== "undefined";
        nav.bluetooth = typeof navigator.bluetooth !== "undefined";
        nav.credentials = typeof navigator.credentials !== "undefined";
        nav.mediaDevices = typeof navigator.mediaDevices !== "undefined";
        nav.serviceWorker = typeof navigator.serviceWorker !== "undefined";
        // Collect plugin info (for security analysis)
        nav.pluginCount = navigator.plugins?.length ?? 0;
        nav.mimeTypeCount = navigator.mimeTypes?.length ?? 0;
    }
    catch (e) {
        nav.error = String(e);
    }
    return nav;
};

const readScreen = () => {
    if (typeof window === "undefined")
        return {};
    const s = window.screen || {};
    const dpr = window.devicePixelRatio || 1;
    return { w: s.width, h: s.height, aw: s.availWidth, ah: s.availHeight, dpr };
};

const readDoc = () => {
    if (typeof document === "undefined")
        return {};
    return {
        ref: document.referrer || "",
        vis: document.visibilityState || "visible",
        hasFocus: !!document.hasFocus?.()
    };
};

const readPerf = () => {
    try {
        const t = performance.getEntriesByType?.("navigation")?.[0];
        if (!t)
            return {};
        return { ttfb: Math.max(0, t.responseStart - t.startTime), dom: Math.max(0, t.domContentLoadedEventEnd - t.startTime) };
    }
    catch {
        return {};
    }
};

let clicks = 0, keys = 0;
if (typeof window !== "undefined") {
    window.addEventListener?.("click", () => { clicks++; }, { passive: true });
    window.addEventListener?.("keydown", () => { keys++; }, { passive: true });
}
const readInputEntropy = () => ({ clicks, keys });

const rng = (len = 16) => {
    const a = new Uint8Array(len);
    (globalThis.crypto || {}).getRandomValues?.(a);
    return Array.from(a, b => b.toString(16).padStart(2, "0")).join("");
};

const KEY = "gt_sid";
const getSessionId = () => {
    try {
        const s = localStorage.getItem(KEY);
        if (s)
            return s;
        const v = rng(16);
        localStorage.setItem(KEY, v);
        return v;
    }
    catch {
        return rng(16);
    }
};


const runRegistry = async (detectors) => {
    const results = [];
    for (const d of detectors) {
        try {
            const r = await Promise.resolve(d.run());
            results.push({ ...r, score: (r.score ?? 0) * (d.weight ?? 1) });
        }
        catch {
            results.push({ id: d.id, score: 0, details: { error: true } });
        }
    }
    const score = results.reduce((s, r) => s + (r.score || 0), 0);
    const bucket = score >= 3 ? "high" : score >= 1 ? "med" : "low";
    return { results, score, bucket };
};

const pluginsDetector = {
    id: "plugins",
    run: () => {
        let plugins = [];
        let mimeTypes = [];
        let pluginsLen = 0;
        let mimeLen = 0;
        try {
            pluginsLen = navigator.plugins?.length ?? 0;
            mimeLen = navigator.mimeTypes?.length ?? 0;
            // Collect plugin names for security analysis
            if (navigator.plugins) {
                for (let i = 0; i < navigator.plugins.length; i++) {
                    const plugin = navigator.plugins[i];
                    if (plugin?.name) {
                        plugins.push(plugin.name);
                    }
                }
            }
            // Collect mime types for security analysis
            if (navigator.mimeTypes) {
                for (let i = 0; i < navigator.mimeTypes.length; i++) {
                    const mime = navigator.mimeTypes[i];
                    if (mime?.type) {
                        mimeTypes.push(mime.type);
                    }
                }
            }
        }
        catch { }
        const suspicious = pluginsLen === 0 && mimeLen === 0;
        const score = suspicious ? 2 : (pluginsLen < 3 ? 1 : 0); // More aggressive scoring
        return {
            id: "plugins",
            score,
            details: {
                pluginsLen,
                mimeLen,
                plugins: plugins.slice(0, 20), // Limit to first 20 to avoid huge payloads
                mimeTypes: mimeTypes.slice(0, 20),
                suspicious
            }
        };
    }
};

const uaDetector = {
    id: "ua",
    run: () => {
        let ua = "";
        let platform = "";
        let userAgentData = null;
        let oscpu = "";
        let appVersion = "";
        let appName = "";
        let vendor = "";
        let suspicious = false;
        try {
            ua = (typeof navigator !== "undefined" && navigator.userAgent) ? navigator.userAgent : "";
            platform = (typeof navigator !== "undefined" && navigator.platform) || "";
            oscpu = (typeof navigator !== "undefined" && navigator.oscpu) || "";
            appVersion = (typeof navigator !== "undefined" && navigator.appVersion) || "";
            appName = (typeof navigator !== "undefined" && navigator.appName) || "";
            vendor = (typeof navigator !== "undefined" && navigator.vendor) || "";
            // Collect modern User-Agent Client Hints if available
            if (typeof navigator !== "undefined" && navigator.userAgentData) {
                userAgentData = {
                    brands: navigator.userAgentData.brands || [],
                    mobile: navigator.userAgentData.mobile,
                    platform: navigator.userAgentData.platform || ""
                };
            }
            // Check for automation tool signatures
            const uaLower = ua.toLowerCase();
            const platformLower = platform.toLowerCase();
            suspicious = !!(/headlesschrome|phantomjs|puppeteer|playwright|selenium|webdriver|chromedriver|automation/i.test(ua) ||
                // Platform inconsistencies  
                (platformLower.includes("win") && !uaLower.includes("windows")) ||
                (platformLower.includes("mac") && uaLower.includes("windows")) ||
                (platformLower.includes("linux") && uaLower.includes("windows")) ||
                // Empty or minimal UA
                ua.length < 20 ||
                // Missing expected fields
                (ua && !platform));
        }
        catch { }
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

const permissionsDetector = {
    id: "permissions",
    run: async () => {
        const permissions = {};
        let suspicious = false;
        let available = false;
        try {
            if (typeof navigator !== "undefined" && navigator.permissions?.query) {
                available = true;
                // Test multiple permissions for security analysis
                const permissionsToTest = [
                    'camera', 'microphone', 'geolocation', 'notifications',
                    'persistent-storage', 'push', 'midi'
                ];
                for (const perm of permissionsToTest) {
                    try {
                        const result = await navigator.permissions.query({ name: perm });
                        permissions[perm] = result.state;
                        // In headless/automation, permissions often behave strangely
                        if (result.state === "prompt" && perm === "camera") {
                            suspicious = true; // Camera should rarely be in prompt state
                        }
                    }
                    catch (e) {
                        permissions[perm] = `error: ${e}`;
                    }
                }
                // Check for automation-specific permission behaviors
                if (Object.values(permissions).every(state => state === "denied")) {
                    suspicious = true; // All permissions denied is suspicious
                }
            }
        }
        catch (e) {
            permissions.error = String(e);
        }
        return {
            id: "permissions",
            score: suspicious ? 1 : 0,
            details: {
                available,
                permissions,
                suspicious,
                permissionCount: Object.keys(permissions).length
            }
        };
    }
};

const webglDetector = {
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
            const gl = canvas.getContext("webgl") || canvas.getContext("experimental-webgl");
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
            }
            else {
                suspicious = true; // No WebGL support is very suspicious
            }
        }
        catch {
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

const webdriverDetector = {
    id: "webdriver",
    run: () => {
        let webdriver = false;
        let automation = false;
        let chromeRuntime = false;
        let seleniumGlobals = false;
        let phantomGlobals = false;
        let suspicious = false;
        const globals = [];
        try {
            // Check navigator.webdriver
            webdriver = !!navigator.webdriver;
            // Check for automation flags
            automation = !!navigator.automation;
            // Check for Chrome runtime (missing in headless)
            chromeRuntime = typeof window.chrome !== "undefined" &&
                typeof window.chrome.runtime !== "undefined";
            // Check for Selenium global variables
            const seleniumVars = ['__selenium_unwrapped', '__webdriver_evaluate', '__selenium_evaluate',
                '__fxdriver_unwrapped', '__driver_evaluate', '__webdriver_script_fn'];
            seleniumVars.forEach(varName => {
                if (window[varName] !== undefined) {
                    seleniumGlobals = true;
                    globals.push(varName);
                }
            });
            // Check for PhantomJS globals
            const phantomVars = ['__phantomas', '_phantom', 'callPhantom'];
            phantomVars.forEach(varName => {
                if (window[varName] !== undefined) {
                    phantomGlobals = true;
                    globals.push(varName);
                }
            });
            // Check for other automation indicators
            const otherIndicators = ['cdc_adoQpoasnfa76pfcZLmcfl_Array', 'cdc_adoQpoasnfa76pfcZLmcfl_Promise',
                'cdc_adoQpoasnfa76pfcZLmcfl_Symbol', '$chrome_asyncScriptInfo', '$cdc_asdjflasutopfhvcZLmcfl_'];
            otherIndicators.forEach(indicator => {
                if (window[indicator] !== undefined) {
                    globals.push(indicator);
                }
            });
            suspicious = webdriver || automation || seleniumGlobals || phantomGlobals ||
                globals.length > 0 ||
                // Missing chrome runtime in Chrome is suspicious
                (navigator.userAgent.includes("Chrome") && !chromeRuntime);
        }
        catch { }
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

const functionToStringDetector = {
    id: "fn_to_string",
    run: () => {
        let nativeSample = "";
        let customSample = "";
        let toStringSample = "";
        let suspicious = false;
        try {
            // Test native function toString
            nativeSample = Function.prototype.toString.call(() => { });
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
        }
        catch (e) {
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

const audioContextDetector = {
    id: "audio_context",
    run: () => {
        let sampleRate = 0;
        let maxChannelCount = 0;
        let numberOfInputs = 0;
        let numberOfOutputs = 0;
        let state = "";
        let baseLatency = 0;
        let outputLatency = 0;
        let available = false;
        try {
            const AudioContextClass = window.AudioContext || window.webkitAudioContext;
            if (!AudioContextClass) {
                return createResult();
            }
            const ctx = new AudioContextClass();
            available = true;
            // Collect audio context properties
            sampleRate = ctx.sampleRate || 0;
            maxChannelCount = ctx.destination?.maxChannelCount || 0;
            numberOfInputs = ctx.destination?.numberOfInputs || 0;
            numberOfOutputs = ctx.destination?.numberOfOutputs || 0;
            state = ctx.state || "";
            baseLatency = ctx.baseLatency || 0;
            outputLatency = ctx.outputLatency || 0;
            // Clean up
            ctx.close?.();
        }
        catch (e) {
            // Error details captured in available flag
        }
        function createResult() {
            return {
                id: "audio_context",
                score: 0, // No scoring, just raw data
                details: {
                    available,
                    sampleRate,
                    maxChannelCount,
                    numberOfInputs,
                    numberOfOutputs,
                    state,
                    baseLatency,
                    outputLatency
                }
            };
        }
        return createResult();
    }
};

const apiMatrixDetector = {
    id: "api_matrix",
    run: () => {
        let apis = {};
        let inconsistencies = [];
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
            if (isChrome && !apis.serviceWorker) {
                inconsistencies.push("chrome_no_sw");
            }
            // Mobile device with desktop-only APIs
            if (isMobile && (apis.usb || apis.serial || apis.hid)) {
                inconsistencies.push("mobile_desktop_apis");
            }
            // Desktop claiming mobile APIs
            if (!isMobile && apis.webNFC) {
                inconsistencies.push("desktop_mobile_apis");
            }
            // Firefox with Chrome-specific APIs
            if (isFirefox && (apis.bluetooth || apis.usb || apis.serial)) {
                inconsistencies.push("firefox_chrome_apis");
            }
            // Safari with non-Safari APIs
            if (isSafari && (apis.bluetooth || apis.usb || apis.serial || apis.hid)) {
                inconsistencies.push("safari_chrome_apis");
            }
        }
        catch (e) {
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

const environmentInconsistencyDetector = {
    id: "env_inconsistency",
    run: () => {
        let screenInconsistencies = [];
        let localeInconsistencies = [];
        let timingInconsistencies = [];
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
            }
            catch { }
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
        }
        catch (e) {
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
                        try {
                            return Intl.DateTimeFormat().resolvedOptions().timeZone;
                        }
                        catch {
                            return "";
                        }
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

const jsEngineFingerprintDetector = {
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
        let engineSignatures = [];
        try {
            // Error stack format (V8, SpiderMonkey, JavaScriptCore have different formats)
            try {
                throw new Error("test");
            }
            catch (e) {
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
            }
            catch {
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
            }
            catch {
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
                if (window[global] !== undefined) {
                    engineSignatures.push("v8_debug_global");
                    suspicious = true;
                }
            });
            spiderMonkeyGlobals.forEach(global => {
                if (window[global] !== undefined) {
                    engineSignatures.push("spidermonkey_global");
                }
            });
            nodeGlobals.forEach(global => {
                if (window[global] !== undefined) {
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
            }
            catch { }
            // Overall suspicion
            if (engineSignatures.length > 2) {
                suspicious = true;
            }
        }
        catch (e) {
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

const detectors = [
    pluginsDetector,
    uaDetector,
    permissionsDetector,
    webglDetector,
    webdriverDetector,
    functionToStringDetector,
    audioContextDetector,
    apiMatrixDetector,
    environmentInconsistencyDetector,
    jsEngineFingerprintDetector
];
const runDetectors = () => runRegistry(detectors);

// Generate a simple UUID-like string for browsers without crypto.randomUUID
const generateId = () => {
    if (typeof crypto !== 'undefined' && crypto.randomUUID) {
        return crypto.randomUUID();
    }
    // Simple fallback
    return 'evt_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
};
const toPayload = (data) => {
    const payload = {
        event_id: generateId(),
        ts: new Date().toISOString(),
        type: "pageview",
    };
    if (data.env) {
        // URL information
        if (typeof location !== 'undefined') {
            payload.url = {
                referrer: document?.referrer || data.env.doc?.referrer || undefined,
                referrer_hostname: document?.referrer ? new URL(document.referrer).hostname : undefined,
                raw_query: location.search || undefined,
            };
            payload.route = {
                domain: location.hostname || undefined,
                path: location.pathname || undefined,
                title: document?.title || undefined,
                protocol: location.protocol?.replace(':', '') || undefined,
            };
        }
        // Device information from collected environment
        payload.device = {};
        if (data.env.nav) {
            payload.device.ua = data.env.nav.ua;
            payload.device.language = data.env.nav.lang;
            payload.device.languages = data.env.nav.langs;
            payload.device.tz = data.env.nav.tz;
            payload.device.tz_offset_minutes = data.env.nav.tzOffset;
            payload.device.hardware_concurrency = data.env.nav.hardwareConcurrency;
            payload.device.cookie_enabled = data.env.nav.cookieEnabled;
            payload.device.storage_available = data.env.nav.storageAvailable;
        }
        if (data.env.screen) {
            payload.device.viewport_w = data.env.screen.w;
            payload.device.viewport_h = data.env.screen.h;
            payload.device.device_pixel_ratio = data.env.screen.dpr;
            payload.device.prefers_color_scheme = data.env.screen.colorScheme;
            // Screen info
            if (data.env.screen.screenW && data.env.screen.screenH) {
                payload.device.screens = [{
                        width: data.env.screen.screenW,
                        height: data.env.screen.screenH,
                        availWidth: data.env.screen.availW,
                        availHeight: data.env.screen.availH,
                        colorDepth: data.env.screen.colorDepth,
                        pixelDepth: data.env.screen.pixelDepth,
                    }];
            }
        }
        // Session information
        if (data.env.session) {
            payload.session = {
                session_id: data.env.session.sid,
            };
        }
    }
    // Bot detection results
    if (data.score !== undefined || (data.detectors && data.detectors.length > 0)) {
        payload.server = {
            bot_score: data.score || 0,
            bot_reasons: (data.detectors || []).map((d) => d.id).filter(Boolean),
        };
    }
    return payload;
};

// Get the tracking endpoint from window.GO_TRACK_URL or default to current page
// This allows posting to any URL to avoid ad-blocker detection
const getDefaultEndpoint = () => {
    // Check if window.GO_TRACK_URL is set
    if (typeof window !== 'undefined' && window.GO_TRACK_URL) {
        return window.GO_TRACK_URL;
    }
    // Default to current page location (harder to block)
    if (typeof window !== 'undefined' && window.location) {
        return window.location.pathname;
    }
    // Fallback to /collect
    return "/collect";
};
const pickEndpoint = (cfg) => cfg.endpoint || getDefaultEndpoint();

const sign = async (body, secret) => {
    if (!secret || typeof globalThis.crypto === "undefined" || !globalThis.crypto.subtle) {
        return null; // No signing if no secret or no crypto support
    }
    try {
        const encoder = new TextEncoder();
        const key = await globalThis.crypto.subtle.importKey("raw", encoder.encode(secret), { name: "HMAC", hash: "SHA-256" }, false, ["sign"]);
        const signature = await globalThis.crypto.subtle.sign("HMAC", key, encoder.encode(body));
        const hashArray = Array.from(new Uint8Array(signature));
        return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
    }
    catch {
        return null; // Return null on any error
    }
};

const fetchSend = async (body, endpoint, secret) => {
    const headers = {
        "Content-Type": "application/json",
        // Always add marker header to identify this as a GoTrack request
        // If HMAC is enabled, the /hmac.js script will replace this with real signature
        "X-GoTrack-HMAC": "tracking"
    };
    // If secret is provided directly, generate HMAC here (legacy support)
    if (secret) {
        const signature = await sign(body, secret);
        if (signature) {
            headers["X-GoTrack-HMAC"] = signature;
        }
    }
    await fetch(endpoint, {
        method: "POST",
        keepalive: true,
        headers,
        body
    });
};

const imgSend = (params, endpoint = "/px.gif") => {
    const q = new URLSearchParams();
    Object.entries(params).forEach(([k, v]) => q.set(k, String(v)));
    const i = new Image(1, 1);
    i.referrerPolicy = "no-referrer";
    // Ensure endpoint supports query params
    const separator = endpoint.includes('?') ? '&' : '?';
    i.src = `${endpoint}${separator}${q.toString()}`;
};

const sendBeaconOrFetch = async (body, endpoint, secret) => {
    // Use fetch with proper HMAC signing
    // The secret is passed through from the config
    try {
        await fetchSend(body, endpoint, secret);
        return;
    }
    catch {
        // Final fallback to img pixel (no HMAC support here)
        try {
            const data = JSON.parse(body);
            imgSend(data, endpoint);
        }
        catch {
            // If JSON parsing fails, send minimal data
            imgSend({ error: "fallback" }, endpoint);
        }
    }
};

function init(cfg = {}) {
    const conf = { ...defaultConfig, ...cfg };
    try {
        const env = {
            nav: readNav(),
            screen: readScreen(),
            doc: readDoc(),
            perf: readPerf(),
            input: readInputEntropy(),
            session: { sid: getSessionId() }
        };
        queueMicrotask(async () => {
            const det = await runDetectors();
            const payload = toPayload({ env, detectors: det.results, score: det.score, bucket: det.bucket });
            await sendBeaconOrFetch(JSON.stringify(payload), pickEndpoint(conf), conf.secret);
        });
    }
    catch { /* never break the page */ }
}
// Auto-initialize if window exists and auto-init is not disabled
if (typeof window !== 'undefined' && !window.GO_TRACK_NO_AUTO_INIT) {
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => init());
    }
    else {
        // Document already loaded
        init();
    }
}

export { init };
//# sourceMappingURL=pixel.esm.js.map
