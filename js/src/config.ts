export interface PixelConfig {
  endpoint?: string;
  batchSize?: number;
  timeout?: number;
  secret?: string; // For HMAC signing
}

export const defaultConfig: PixelConfig = {
  endpoint: "/collect",
  batchSize: 10,
  timeout: 5000
};
