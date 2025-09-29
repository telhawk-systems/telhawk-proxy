export type Payload = {
  ver: number;
  ts: number;
  env?: Record<string, unknown>;
  detectors?: unknown[];
  score?: number;
  bucket?: "low" | "med" | "high";
};

export const toPayload = (p: Partial<Payload>): Payload => ({
  ver: 1,
  ts: Date.now(),
  ...p
});