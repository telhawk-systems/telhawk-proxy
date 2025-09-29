import { fetchSend } from "./fetch";
import { imgSend } from "./img";
import { sign } from "./sign";

export const sendBeaconOrFetch = async (body: string, endpoint: string, secret?: string) => {
  // Add signature if secret is provided
  let finalBody = body;
  if (secret) {
    const signature = await sign(body, secret);
    if (signature) {
      const payload = JSON.parse(body);
      payload._sig = signature;
      finalBody = JSON.stringify(payload);
    }
  }

  // Try sendBeacon first (preferred)
  if (typeof navigator !== "undefined" && navigator.sendBeacon) {
    const ok = navigator.sendBeacon(endpoint, finalBody);
    if (ok) return;
  }
  
  // Fallback to fetch
  try {
    await fetchSend(finalBody, endpoint);
    return;
  } catch {
    // Final fallback to img pixel
    try {
      const data = JSON.parse(finalBody);
      imgSend(data, endpoint);
    } catch {
      // If JSON parsing fails, send minimal data
      imgSend({ error: "fallback" }, endpoint);
    }
  }
};
