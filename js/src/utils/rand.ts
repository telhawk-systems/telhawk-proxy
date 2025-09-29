export const rng = (len = 16): string => {
  const a = new Uint8Array(len);
  (globalThis.crypto || ({} as any)).getRandomValues?.(a);
  return Array.from(a, b => b.toString(16).padStart(2, "0")).join("");
};