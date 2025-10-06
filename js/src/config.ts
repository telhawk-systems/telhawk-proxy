export interface PixelConfig {
  endpoint?: string;
  batchSize?: number;
  timeout?: number;
  secret?: string; // For HMAC signing
}

// Note: endpoint will default to window.GO_TRACK_URL or current page path
// This allows tracking data to be posted to any URL for ad-blocker evasion
export const defaultConfig: PixelConfig = {
  endpoint: undefined, // Will be determined by pickEndpoint()
  batchSize: 10,
  timeout: 5000
};
