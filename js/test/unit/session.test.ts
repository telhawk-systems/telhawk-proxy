import { getSessionId } from '../../src/ids/session';

// Mock localStorage
const createLocalStorageMock = () => {
  const store: Record<string, string> = {};
  
  return {
    getItem: jest.fn((key: string) => store[key] || null),
    setItem: jest.fn((key: string, value: string) => {
      store[key] = value;
    }),
    clear: jest.fn(() => {
      Object.keys(store).forEach(key => delete store[key]);
    }),
    removeItem: jest.fn((key: string) => {
      delete store[key];
    }),
    get length() {
      return Object.keys(store).length;
    },
    key: jest.fn((index: number) => Object.keys(store)[index] || null)
  };
};

let localStorageMock = createLocalStorageMock();

describe('Session ID management', () => {
  beforeEach(() => {
    // Create a fresh mock for each test
    localStorageMock = createLocalStorageMock();
    Object.defineProperty(window, 'localStorage', {
      value: localStorageMock,
      writable: true,
      configurable: true
    });
  });

  test('generates new session ID if none exists', () => {
    const sessionId = getSessionId();
    
    expect(sessionId).toBeDefined();
    expect(sessionId).toHaveLength(32); // 16 bytes = 32 hex chars
    expect(sessionId).toMatch(/^[0-9a-f]+$/);
    expect(localStorageMock.getItem).toHaveBeenCalledWith('gt_sid');
    expect(localStorageMock.setItem).toHaveBeenCalledWith('gt_sid', sessionId);
  });

  test('returns existing session ID if available', () => {
    const existingId = 'abc123def456';
    (localStorageMock.getItem as jest.Mock).mockReturnValue(existingId);
    
    const sessionId = getSessionId();
    
    expect(sessionId).toBe(existingId);
    expect(localStorageMock.getItem).toHaveBeenCalledWith('gt_sid');
    expect(localStorageMock.setItem).not.toHaveBeenCalled();
  });

  test('returns same session ID on multiple calls', () => {
    const sessionId1 = getSessionId();
    const sessionId2 = getSessionId();
    
    expect(sessionId1).toBe(sessionId2);
    expect(localStorageMock.setItem).toHaveBeenCalledTimes(1);
  });

  test('handles localStorage errors gracefully', () => {
    (localStorageMock.getItem as jest.Mock).mockImplementation(() => {
      throw new Error('localStorage disabled');
    });
    
    const sessionId = getSessionId();
    
    // Should still return a valid session ID
    expect(sessionId).toBeDefined();
    expect(sessionId).toHaveLength(32);
    expect(sessionId).toMatch(/^[0-9a-f]+$/);
    // But should not try to set it
    expect(localStorageMock.setItem).not.toHaveBeenCalled();
  });

  test('handles setItem errors gracefully', () => {
    (localStorageMock.setItem as jest.Mock).mockImplementation(() => {
      throw new Error('Storage quota exceeded');
    });
    
    const sessionId = getSessionId();
    
    // Should still return a valid session ID
    expect(sessionId).toBeDefined();
    expect(sessionId).toHaveLength(32);
    expect(sessionId).toMatch(/^[0-9a-f]+$/);
  });

  test('generates different session IDs for different calls when localStorage fails', () => {
    (localStorageMock.getItem as jest.Mock).mockImplementation(() => {
      throw new Error('localStorage disabled');
    });
    
    const sessionId1 = getSessionId();
    const sessionId2 = getSessionId();
    
    expect(sessionId1).not.toBe(sessionId2);
    expect(sessionId1).toHaveLength(32);
    expect(sessionId2).toHaveLength(32);
  });
});
