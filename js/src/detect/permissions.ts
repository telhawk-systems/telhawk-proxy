import type { Detector } from "./types";
export const permissionsDetector: Detector = {
  id: "permissions",
  run: async () => {
    const q = (typeof navigator !== "undefined" && (navigator as any).permissions?.query) || null;
    if (!q) return { id: "permissions", score: 0, details: { available: false } };
    try {
      const r = await q.call((navigator as any).permissions, { name: "camera" as any });
      const suspicious = r?.state === "prompt" || r?.state === "unknown";
      return { id: "permissions", score: suspicious ? 1 : 0, details: { state: r?.state } };
    } catch { return { id: "permissions", score: 0 }; }
  }
};
