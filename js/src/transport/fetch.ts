import { sign } from "./sign";

export const fetchSend = async (body: string, endpoint: string, secret?: string) => {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    // Always add marker header to identify this as a GoTrack request
    // If HMAC is enabled, the /hmac.js script will replace this with real signature
    "X-GoTrack-HMAC": "tracking"
  };
  
  // If secret is provided directly, generate HMAC here (legacy support)
  if (secret) {
    const signature = await sign(body, secret);
    if (signature) {
      headers["X-GoTrack-HMAC"] = signature;
    }
  }
  
  await fetch(endpoint, { 
    method: "POST", 
    keepalive: true, 
    headers, 
    body 
  });
};
