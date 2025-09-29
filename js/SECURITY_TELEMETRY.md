# Security Telemetry Collection Summary

## 🔍 **Comprehensive Data Collection for Fraud/Bot Detection**

### **🔌 Plugin Analysis (`pluginsDetector`)**
- **Plugin names** - Full list of installed browser plugins  
- **MIME types** - Complete list of supported MIME types
- **Plugin counts** - Number of plugins vs MIME types for consistency checks
- **Suspicious patterns** - Empty plugin arrays (common in headless browsers)

### **🎮 WebGL Fingerprinting (`webglDetector`)**  
- **Vendor info** - Graphics card vendor (Intel, NVIDIA, AMD)
- **Renderer details** - Specific GPU model and driver info
- **Unmasked data** - Attempts to get real hardware info bypassing privacy protections
- **Version strings** - WebGL and shading language versions
- **Automation signatures** - SwiftShader, ANGLE, Mesa (software rendering)

### **🌐 User Agent Analysis (`uaDetector`)**
- **Full user agent string** - Complete browser/OS identification
- **Platform details** - OS platform, CPU architecture 
- **App metadata** - Browser name, version, vendor info
- **Client hints** - Modern UA client hints when available
- **Consistency checks** - Platform vs UA string validation
- **Automation detection** - Headless, Selenium, Puppeteer signatures

### **🔧 WebDriver Detection (`webdriverDetector`)**
- **WebDriver flags** - navigator.webdriver property
- **Automation properties** - navigator.automation and similar
- **Global variables** - Selenium, PhantomJS, ChromeDriver globals
- **Chrome runtime** - Checks for missing chrome.runtime in headless
- **CDC indicators** - Chrome DevTools Command signatures

### **🔧 Function Tampering (`functionToStringDetector`)**
- **Native function samples** - toString() output from clean functions
- **Built-in function tests** - Array.push and other native methods
- **Tamper detection** - Modified Function.prototype.toString
- **Automation signatures** - Puppeteer, Selenium method modification

### **🔐 Permissions Analysis (`permissionsDetector`)**
- **Multiple permissions** - Camera, microphone, geolocation, notifications
- **Permission states** - Granted, denied, prompt for each
- **Behavioral patterns** - All-denied states (common in automation)
- **API availability** - Whether Permissions API exists

### **🧭 Navigator Fingerprinting (`readNav`)**
- **Complete navigator object** - All available properties and methods
- **Language preferences** - Primary language + full language list  
- **Hardware details** - CPU cores, touch points, memory hints
- **Feature detection** - Available APIs (bluetooth, credentials, etc.)
- **Automation flags** - webdriver, automation properties
- **Privacy settings** - DNT, GPC (collected but not honored)

### **📱 Environment Context**
- **Screen details** - Resolution, DPR, available space
- **Document state** - Referrer, visibility, focus status  
- **Performance timing** - TTFB, DOM load times
- **Input entropy** - Click/keystroke counts for human behavior
- **Session tracking** - Persistent session IDs

## 🎯 **Security-First Scoring**

Higher scores for more suspicious indicators:
- **WebDriver detected**: +3 points (very high confidence)
- **Empty plugins**: +2 points (headless browsers)
- **Software rendering**: +2 points (automation VMs)
- **UA inconsistencies**: +2 points (spoofed environments)
- **Function tampering**: +2 points (automation tool injection)

## 📦 **Rich Payload Example**

```json
{
  "ver": 1,
  "ts": 1696035000000,
  "env": {
    "nav": {
      "ua": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36...",
      "plugins": ["Chrome PDF Plugin", "Native Client", ...],
      "mimeTypes": ["application/pdf", "application/x-nacl", ...],
      "webdriver": false,
      "vendor": "Google Inc.",
      "platform": "Win32"
    },
    "screen": { "w": 1920, "h": 1080, "dpr": 1.5 }
  },
  "detectors": [
    {
      "id": "webdriver",
      "score": 0,
      "details": {
        "webdriver": false,
        "automation": false,
        "globals": [],
        "chromeRuntime": true
      }
    },
    {
      "id": "plugins", 
      "score": 0,
      "details": {
        "plugins": ["Chrome PDF Plugin", "Native Client"],
        "mimeTypes": ["application/pdf", "application/x-nacl"],
        "pluginsLen": 2,
        "mimeLen": 2
      }
    },
    {
      "id": "webgl",
      "score": 0, 
      "details": {
        "vendor": "Intel Inc.",
        "renderer": "Intel(R) HD Graphics 620",
        "unmaskedVendor": "Intel Inc.",
        "unmaskedRenderer": "Intel(R) HD Graphics 620"
      }
    }
  ],
  "score": 0,
  "bucket": "low"
}
```

This comprehensive telemetry provides deep visibility into browser environments for accurate bot/automation detection while maintaining performance and reliability.