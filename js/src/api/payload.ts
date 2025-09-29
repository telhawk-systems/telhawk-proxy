// Event structure matching the Go backend
export type Payload = {
  event_id?: string;
  ts?: string; // ISO8601
  type?: string;
  url?: {
    referrer?: string;
    referrer_hostname?: string;
    raw_query?: string;
  };
  route?: {
    domain?: string;
    path?: string;
    title?: string;
    protocol?: string;
  };
  device?: {
    ua?: string;
    language?: string;
    languages?: string[];
    tz?: string;
    tz_offset_minutes?: number;
    viewport_w?: number;
    viewport_h?: number;
    device_pixel_ratio?: number;
    hardware_concurrency?: number;
    prefers_color_scheme?: string;
    cookie_enabled?: boolean;
    storage_available?: boolean;
    screens?: Array<{
      width?: number;
      height?: number;
      availWidth?: number;
      availHeight?: number;
      colorDepth?: number;
      pixelDepth?: number;
    }>;
  };
  session?: {
    visitor_id?: string;
    session_id?: string;
  };
  server?: {
    bot_score?: number;
    bot_reasons?: string[];
  };
};

// Generate a simple UUID-like string for browsers without crypto.randomUUID
const generateId = (): string => {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  // Simple fallback
  return 'evt_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
};

export const toPayload = (data: {
  env?: any;
  detectors?: any[];
  score?: number;
  bucket?: "low" | "med" | "high";
}): Payload => {
  const payload: Payload = {
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
      bot_reasons: (data.detectors || []).map((d: any) => d.id).filter(Boolean),
    };
  }

  return payload;
};