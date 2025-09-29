export type PixelConfig = {
  endpoint: string;
  version?: string | number;
  secret?: string;
};

export const defaultConfig: PixelConfig = {
  endpoint: "/collect",
  version: 1
};