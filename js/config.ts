export type PixelConfig = {
  endpoint: string;
  version?: string | number;
};

export const defaultConfig: PixelConfig = {
  endpoint: "/collect",
  version: 1
};