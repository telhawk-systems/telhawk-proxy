import { fetchSend } from "./fetch";

export const sendBeaconOrFetch = async (body: string, endpoint: string) => {
  const ok = typeof navigator !== "undefined" && !!navigator.sendBeacon && navigator.sendBeacon(endpoint, body);
  if (!ok) await fetchSend(body, endpoint);
};
