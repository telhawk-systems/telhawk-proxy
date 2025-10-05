export const sign = async (body: string, secret?: string): Promise<string | null> => {
  if (!secret || typeof globalThis.crypto === "undefined" || !globalThis.crypto.subtle) {
    return null; // No signing if no secret or no crypto support
  }
  
  try {
    const encoder = new TextEncoder();
    const key = await globalThis.crypto.subtle.importKey(
      "raw",
      encoder.encode(secret),
      { name: "HMAC", hash: "SHA-256" },
      false,
      ["sign"]
    );
    
    const signature = await globalThis.crypto.subtle.sign("HMAC", key, encoder.encode(body));
    const hashArray = Array.from(new Uint8Array(signature));
    return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
  } catch {
    return null; // Return null on any error
  }
};
