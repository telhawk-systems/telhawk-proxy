import { init } from '../../src/index';
import { pluginsDetector } from '../../src/detect/plugins';
import { webdriverDetector } from '../../src/detect/webdriver';
import { functionToStringDetector } from '../../src/detect/functionToString';

describe('Pixel Library', () => {
  test('init function exists and can be called', () => {
    expect(typeof init).toBe('function');
    
    // Should not throw
    expect(() => {
      init();
    }).not.toThrow();
  });

  test('all detectors export correctly', () => {
    expect(pluginsDetector.id).toBe('plugins');
    expect(webdriverDetector.id).toBe('webdriver');
    expect(functionToStringDetector.id).toBe('fn_to_string');
    
    expect(typeof pluginsDetector.run).toBe('function');
    expect(typeof webdriverDetector.run).toBe('function');
    expect(typeof functionToStringDetector.run).toBe('function');
  });

  test('detectors return proper result structure', () => {
    const pluginResult = pluginsDetector.run();
    expect(pluginResult).toHaveProperty('id');
    expect(pluginResult).toHaveProperty('score');
    expect(pluginResult).toHaveProperty('details');
    
    const webdriverResult = webdriverDetector.run();
    expect(webdriverResult).toHaveProperty('id');
    expect(webdriverResult).toHaveProperty('score');
    expect(webdriverResult).toHaveProperty('details');
  });

  test('init with custom config works', () => {
    expect(() => {
      init({
        endpoint: 'https://custom.example.com/collect',
        batchSize: 5
      });
    }).not.toThrow();
  });
});