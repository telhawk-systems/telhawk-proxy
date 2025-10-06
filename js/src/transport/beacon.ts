import { fetchSend } from "./fetch";
import { imgSend } from "./img";
import { sign } from "./sign";

export const sendBeaconOrFetch = async (body: string, endpoint: string, secret?: string) => {
  // Use fetch with proper HMAC signing
  // The secret is passed through from the config
  try {
    await fetchSend(body, endpoint, secret);
    return;
  } catch {
    // Final fallback to img pixel (no HMAC support here)
    try {
      const data = JSON.parse(body);
      imgSend(data, endpoint);
    } catch {
      // If JSON parsing fails, send minimal data
      imgSend({ error: "fallback" }, endpoint);
    }
  }
};
