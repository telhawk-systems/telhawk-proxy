import { rng } from "../utils/rand";

const KEY = "gt_sid";
export const getSessionId = (): string => {
  try {
    const s = localStorage.getItem(KEY);
    if (s) return s;
    const v = rng(16);
    localStorage.setItem(KEY, v);
    return v;
  } catch {
    return rng(16);
  }
};