import type { Detector, DetectorResult } from "./types";

export const runRegistry = async (detectors: Detector[]) => {
  const results: DetectorResult[] = [];

  for (const d of detectors) {
    try {
      const r = await Promise.resolve(d.run());
      results.push({ ...r, score: (r.score ?? 0) * (d.weight ?? 1) });
    } catch {
      results.push({ id: d.id, score: 0, details: { error: true } });
    }
  }

  const score = results.reduce((s, r) => s + (r.score || 0), 0);
  const bucket: "low" | "med" | "high" = score >= 3 ? "high" : score >= 1 ? "med" : "low";

  return { results, score, bucket };
};
