export const nowMs = (): number => (typeof performance !== "undefined" && performance.now) ? performance.now() : Date.now();
export const ts = (): number => Date.now();