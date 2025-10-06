export type RouteConfig = { endpoint?: string };

// Get the tracking endpoint from window.GO_TRACK_URL or default to current page
// This allows posting to any URL to avoid ad-blocker detection
const getDefaultEndpoint = (): string => {
  // Check if window.GO_TRACK_URL is set
  if (typeof window !== 'undefined' && (window as any).GO_TRACK_URL) {
    return (window as any).GO_TRACK_URL;
  }
  // Default to current page location (harder to block)
  if (typeof window !== 'undefined' && window.location) {
    return window.location.pathname;
  }
  // Fallback to /collect
  return "/collect";
};

export const pickEndpoint = (cfg: RouteConfig): string => 
  cfg.endpoint || getDefaultEndpoint();