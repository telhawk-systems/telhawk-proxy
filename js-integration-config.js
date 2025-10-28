// proxy JS Integration Configuration
// This file configures the JS security pixel to work with the Go backend

// Configuration for the pixel
const PROXY_CONFIG = {
  endpoint: "http://localhost:19890/",
  version: 1
};

// Simple payload builder that matches Go Event structure
function buildProxyPayload(env, detectors, score) {
  const payload = {
    event_id: generateId(),
    ts: new Date().toISOString(),
    type: "pageview"
  };

  // URL information
  if (typeof location !== 'undefined') {
    payload.url = {
      referrer: document?.referrer || undefined,
      raw_query: location.search || undefined
    };

    payload.route = {
      domain: location.hostname || undefined,
      path: location.pathname || undefined,
      title: document?.title || undefined,
      protocol: location.protocol?.replace(':', '') || undefined
    };
  }

  // Device information
  payload.device = {};
  
  if (env && env.nav) {
    payload.device.ua = env.nav.ua;
    payload.device.language = env.nav.lang;
    payload.device.languages = env.nav.langs;
  }

  if (env && env.screen) {
    payload.device.viewport_w = env.screen.w;
    payload.device.viewport_h = env.screen.h;
    payload.device.device_pixel_ratio = env.screen.dpr;
  }

  // Session information
  if (env && env.session) {
    payload.session = {
      session_id: env.session.sid
    };
  }

  // Bot detection results
  if (score !== undefined || (detectors && detectors.length > 0)) {
    payload.server = {
      bot_score: score || 0,
      bot_reasons: (detectors || []).map(d => d.id).filter(Boolean)
    };
  }

  return payload;
}

// Generate a simple ID
function generateId() {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return 'evt_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
}

// Send data to Go backend
async function send(payload) {
  try {
    // Try fetch first
    const response = await fetch(PROXY_CONFIG.endpoint, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(payload),
      keepalive: true
    });
    
    if (response.ok) {
      console.log('✅ Proxy: Event sent successfully');
      return true;
    } else {
      console.warn('⚠️ TelHawk Proxy Server responded with', response.status);
      return false;
    }
  } catch (error) {
    console.warn('⚠️ TelHawk Proxy Fetch failed, trying fallback');
    
    // Fallback to image pixel
    try {
      const img = new Image(1, 1);
      const params = new URLSearchParams({
        e: payload.type || 'pageview',
        event_id: payload.event_id,
        ua: payload.device?.ua || '',
        url: location?.href || ''
      });
      img.src = `http://localhost:19890/px.gif?${params.toString()}`;
      console.log('✅ TelHawk Proxy Fallback pixel sent');
      return true;
    } catch (fallbackError) {
      console.error('❌ TelHawk Proxy All methods failed', fallbackError);
      return false;
    }
  }
}

// Basic environment collection (simplified version of what the TS pixel does)
function collectEnvironment() {
  const env = {
    nav: {},
    screen: {},
    session: {}
  };

  if (typeof navigator !== 'undefined') {
    env.nav.ua = navigator.userAgent || '';
    env.nav.lang = navigator.language || '';
    env.nav.langs = navigator.languages ? Array.from(navigator.languages).slice(0, 5) : [];
    env.nav.cookieEnabled = navigator.cookieEnabled;
  }

  if (typeof screen !== 'undefined') {
    env.screen.w = window.innerWidth || 0;
    env.screen.h = window.innerHeight || 0;
    env.screen.screenW = screen.width;
    env.screen.screenH = screen.height;
    env.screen.dpr = window.devicePixelRatio || 1;
    env.screen.colorDepth = screen.colorDepth;
  }

  // Simple session ID (in real implementation this would be more sophisticated)
  env.session.sid = sessionStorage.getItem('telhawk-proxy_sid') || (() => {
    const sid = generateId();
    try {
      sessionStorage.setItem('telhawk-proxy_sid', sid);
    } catch (e) {
      // Storage not available
    }
    return sid;
  })();

  return env;
}

// Initialize proxy pixel
function initproxy() {
  try {
    console.log('🚀 Initializing proxy pixel...');
    
    const env = collectEnvironment();
    const detectors = []; // Simplified - real version would run bot detection
    const score = 0; // Simplified scoring
    
    const payload = buildProxyPayload(env, detectors, score);
    
    // Send the payload
    send(payload);
    
  } catch (error) {
    console.error('❌ proxy initialization failed:', error);
  }
}

// Export for testing
if (typeof window !== 'undefined') {
  window.proxy = {
    init: initproxy,
    config: PROXY_CONFIG,
    buildPayload: buildProxyPayload,
    send: send
  };
}