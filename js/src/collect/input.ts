export type InputEntropy = { clicks: number; keys: number };
let clicks = 0, keys = 0;

if (typeof window !== "undefined") {
  window.addEventListener?.("click", () => { clicks++; }, { passive: true });
  window.addEventListener?.("keydown", () => { keys++; }, { passive: true });
}

export const readInputEntropy = (): InputEntropy => ({ clicks, keys });
