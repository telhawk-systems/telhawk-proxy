export const hasWindow = typeof window !== "undefined";
export const hasDocument = typeof document !== "undefined";

export const createEl = (tag = "div"): HTMLElement | null =>
  hasDocument ? document.createElement(tag) : null;