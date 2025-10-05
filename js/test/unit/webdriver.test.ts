import { webdriverDetector } from '../../src/detect/webdriver';
import type { DetectorResult } from '../../src/detect/types';

describe('Webdriver Detector', () => {
  let originalNavigator: any;
  let originalWindow: any;

  beforeEach(() => {
    // Save originals
    originalNavigator = { ...navigator };
    originalWindow = { ...window };
  });

  afterEach(() => {
    // Restore originals
    Object.defineProperty(global, 'navigator', {
      value: originalNavigator,
      writable: true,
      configurable: true
    });
  });

  test('has correct detector ID', () => {
    expect(webdriverDetector.id).toBe('webdriver');
  });

  test('detects clean browser with score 0', () => {
    const result = webdriverDetector.run() as DetectorResult;
    
    expect(result.id).toBe('webdriver');
    expect(result.score).toBe(0);
    expect(result.details!.suspicious).toBe(false);
    expect(result.details!.webdriver).toBe(false);
    expect(result.details!.automation).toBe(false);
  });

  test('detects navigator.webdriver flag', () => {
    Object.defineProperty(navigator, 'webdriver', {
      value: true,
      writable: true,
      configurable: true
    });
    
    const result = webdriverDetector.run() as DetectorResult;
    
    expect(result.score).toBe(3);
    expect(result.details!.webdriver).toBe(true);
    expect(result.details!.suspicious).toBe(true);
    expect(result.reliable).toBe(true);
  });

  test('detects navigator.automation flag', () => {
    Object.defineProperty(navigator, 'automation', {
      value: true,
      writable: true,
      configurable: true
    });
    
    const result = webdriverDetector.run() as DetectorResult;
    
    expect(result.score).toBe(3);
    expect(result.details!.automation).toBe(true);
    expect(result.details!.suspicious).toBe(true);
    expect(result.reliable).toBe(true);
  });

  test('detects Selenium global variables', () => {
    (window as any).__selenium_unwrapped = {};
    (window as any).__webdriver_evaluate = {};
    
    const result = webdriverDetector.run() as DetectorResult;
    
    expect(result.score).toBe(3);
    expect(result.details!.seleniumGlobals).toBe(true);
    expect(result.details!.globals).toContain('__selenium_unwrapped');
    expect(result.details!.globals).toContain('__webdriver_evaluate');
    expect(result.details!.suspicious).toBe(true);
    expect(result.reliable).toBe(true);
    
    // Cleanup
    delete (window as any).__selenium_unwrapped;
    delete (window as any).__webdriver_evaluate;
  });

  test('detects PhantomJS global variables', () => {
    (window as any).__phantomas = {};
    (window as any)._phantom = {};
    
    const result = webdriverDetector.run() as DetectorResult;
    
    expect(result.score).toBe(3);
    expect(result.details!.phantomGlobals).toBe(true);
    expect(result.details!.globals).toContain('__phantomas');
    expect(result.details!.globals).toContain('_phantom');
    expect(result.details!.suspicious).toBe(true);
    
    // Cleanup
    delete (window as any).__phantomas;
    delete (window as any)._phantom;
  });

  test('detects Chrome CDP global variables', () => {
    (window as any).cdc_adoQpoasnfa76pfcZLmcfl_Array = {};
    (window as any).$chrome_asyncScriptInfo = {};
    
    const result = webdriverDetector.run() as DetectorResult;
    
    expect(result.score).toBe(3);
    expect((result.details!.globals as unknown[]).length).toBeGreaterThan(0);
    expect(result.details!.suspicious).toBe(true);
    
    // Cleanup
    delete (window as any).cdc_adoQpoasnfa76pfcZLmcfl_Array;
    delete (window as any).$chrome_asyncScriptInfo;
  });

  test('limits globals array to 10 items', () => {
    // Add more than 10 globals
    for (let i = 0; i < 15; i++) {
      (window as any)[`__selenium_test_${i}`] = {};
    }
    
    // Manually set some known selenium vars
    (window as any).__selenium_unwrapped = {};
    (window as any).__webdriver_evaluate = {};
    
    const result = webdriverDetector.run() as DetectorResult;
    
    expect((result.details!.globals as unknown[]).length).toBeLessThanOrEqual(10);
    
    // Cleanup
    for (let i = 0; i < 15; i++) {
      delete (window as any)[`__selenium_test_${i}`];
    }
    delete (window as any).__selenium_unwrapped;
    delete (window as any).__webdriver_evaluate;
  });

  test('detects missing Chrome runtime in Chrome browser', () => {
    Object.defineProperty(navigator, 'userAgent', {
      value: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36',
      writable: true,
      configurable: true
    });
    
    // Ensure chrome.runtime is NOT present
    delete (window as any).chrome;
    
    const result = webdriverDetector.run() as DetectorResult;
    
    expect(result.score).toBe(3);
    expect(result.details!.suspicious).toBe(true);
    expect(result.details!.chromeRuntime).toBe(false);
  });

  test('does not flag Chrome with chrome.runtime present', () => {
    Object.defineProperty(navigator, 'userAgent', {
      value: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36',
      writable: true,
      configurable: true
    });
    
    (window as any).chrome = {
      runtime: {}
    };
    
    const result = webdriverDetector.run() as DetectorResult;
    
    expect(result.score).toBe(0);
    expect(result.details!.suspicious).toBe(false);
    expect(result.details!.chromeRuntime).toBe(true);
    
    // Cleanup
    delete (window as any).chrome;
  });

  test('returns proper result structure', () => {
    const result = webdriverDetector.run() as DetectorResult;
    
    expect(result).toHaveProperty('id');
    expect(result).toHaveProperty('score');
    expect(result).toHaveProperty('details');
    expect(result).toHaveProperty('reliable');
    
    expect(result.details).toHaveProperty('webdriver');
    expect(result.details).toHaveProperty('automation');
    expect(result.details).toHaveProperty('chromeRuntime');
    expect(result.details).toHaveProperty('seleniumGlobals');
    expect(result.details).toHaveProperty('phantomGlobals');
    expect(result.details).toHaveProperty('globals');
    expect(result.details).toHaveProperty('suspicious');
    
    expect(Array.isArray(result.details!.globals)).toBe(true);
  });
});
