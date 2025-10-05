import { pluginsDetector } from '../../src/detect/plugins';

describe('Plugins Detector', () => {
  let originalNavigator: any;

  beforeEach(() => {
    originalNavigator = { ...navigator };
  });

  afterEach(() => {
    Object.defineProperty(global, 'navigator', {
      value: originalNavigator,
      writable: true,
      configurable: true
    });
  });

  test('has correct detector ID', () => {
    expect(pluginsDetector.id).toBe('plugins');
  });

  test('detects browser with plugins and gives low score', () => {
    const mockPlugins = [
      { name: 'Chrome PDF Plugin', length: 1 },
      { name: 'Chrome PDF Viewer', length: 1 },
      { name: 'Native Client', length: 1 },
      { name: 'Widevine Content Decryption Module', length: 1 }
    ];
    
    Object.defineProperty(navigator, 'plugins', {
      value: mockPlugins,
      writable: true,
      configurable: true
    });
    
    const result = pluginsDetector.run();
    
    expect(result.id).toBe('plugins');
    expect(result.score).toBe(0); // 4 plugins, should be 0
    expect(result.details.pluginsLen).toBe(4);
    expect(result.details.suspicious).toBe(false);
  });

  test('detects headless browser with no plugins', () => {
    Object.defineProperty(navigator, 'plugins', {
      value: [],
      writable: true,
      configurable: true
    });
    
    Object.defineProperty(navigator, 'mimeTypes', {
      value: [],
      writable: true,
      configurable: true
    });
    
    const result = pluginsDetector.run();
    
    expect(result.score).toBe(2); // Suspicious: no plugins and no mimeTypes
    expect(result.details.pluginsLen).toBe(0);
    expect(result.details.mimeLen).toBe(0);
    expect(result.details.suspicious).toBe(true);
  });

  test('detects browser with few plugins', () => {
    const mockPlugins = [
      { name: 'Chrome PDF Plugin', length: 1 },
      { name: 'Chrome PDF Viewer', length: 1 }
    ];
    
    Object.defineProperty(navigator, 'plugins', {
      value: mockPlugins,
      writable: true,
      configurable: true
    });
    
    const result = pluginsDetector.run();
    
    expect(result.score).toBe(1); // Less than 3 plugins
    expect(result.details.pluginsLen).toBe(2);
    expect(result.details.suspicious).toBe(false);
  });

  test('collects plugin names', () => {
    const mockPlugins = [
      { name: 'Chrome PDF Plugin', length: 1 },
      { name: 'Native Client', length: 1 },
      { name: 'Widevine CDM', length: 1 }
    ];
    
    Object.defineProperty(navigator, 'plugins', {
      value: mockPlugins,
      writable: true,
      configurable: true
    });
    
    const result = pluginsDetector.run();
    
    expect(result.details.plugins).toContain('Chrome PDF Plugin');
    expect(result.details.plugins).toContain('Native Client');
    expect(result.details.plugins).toContain('Widevine CDM');
  });

  test('collects mime types', () => {
    const mockMimeTypes = [
      { type: 'application/pdf' },
      { type: 'application/x-nacl' },
      { type: 'video/mp4' }
    ];
    
    Object.defineProperty(navigator, 'mimeTypes', {
      value: mockMimeTypes,
      writable: true,
      configurable: true
    });
    
    const result = pluginsDetector.run();
    
    expect(result.details.mimeTypes).toContain('application/pdf');
    expect(result.details.mimeTypes).toContain('application/x-nacl');
    expect(result.details.mimeTypes).toContain('video/mp4');
  });

  test('limits plugins array to 20 items', () => {
    const mockPlugins = Array.from({ length: 30 }, (_, i) => ({
      name: `Plugin ${i}`,
      length: 1
    }));
    
    Object.defineProperty(navigator, 'plugins', {
      value: mockPlugins,
      writable: true,
      configurable: true
    });
    
    const result = pluginsDetector.run();
    
    expect(result.details.plugins.length).toBe(20);
    expect(result.details.pluginsLen).toBe(30);
  });

  test('limits mimeTypes array to 20 items', () => {
    const mockMimeTypes = Array.from({ length: 25 }, (_, i) => ({
      type: `application/type-${i}`
    }));
    
    Object.defineProperty(navigator, 'mimeTypes', {
      value: mockMimeTypes,
      writable: true,
      configurable: true
    });
    
    const result = pluginsDetector.run();
    
    expect(result.details.mimeTypes.length).toBe(20);
    expect(result.details.mimeLen).toBe(25);
  });

  test('handles undefined navigator.plugins gracefully', () => {
    Object.defineProperty(navigator, 'plugins', {
      value: undefined,
      writable: true,
      configurable: true
    });
    
    const result = pluginsDetector.run();
    
    expect(result.details.pluginsLen).toBe(0);
    expect(result.details.plugins).toEqual([]);
  });

  test('handles undefined navigator.mimeTypes gracefully', () => {
    Object.defineProperty(navigator, 'mimeTypes', {
      value: undefined,
      writable: true,
      configurable: true
    });
    
    const result = pluginsDetector.run();
    
    expect(result.details.mimeLen).toBe(0);
    expect(result.details.mimeTypes).toEqual([]);
  });

  test('returns proper result structure', () => {
    const result = pluginsDetector.run();
    
    expect(result).toHaveProperty('id');
    expect(result).toHaveProperty('score');
    expect(result).toHaveProperty('details');
    
    expect(result.details).toHaveProperty('pluginsLen');
    expect(result.details).toHaveProperty('mimeLen');
    expect(result.details).toHaveProperty('plugins');
    expect(result.details).toHaveProperty('mimeTypes');
    expect(result.details).toHaveProperty('suspicious');
    
    expect(typeof result.details.pluginsLen).toBe('number');
    expect(typeof result.details.mimeLen).toBe('number');
    expect(Array.isArray(result.details.plugins)).toBe(true);
    expect(Array.isArray(result.details.mimeTypes)).toBe(true);
    expect(typeof result.details.suspicious).toBe('boolean');
  });
});
