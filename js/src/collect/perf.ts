export const readPerf = () => {
  try {
    const t = performance.getEntriesByType?.("navigation")?.[0] as PerformanceNavigationTiming | undefined;
    if (!t) return {};
    return { ttfb: Math.max(0, t.responseStart - t.startTime), dom: Math.max(0, t.domContentLoadedEventEnd - t.startTime) };
  } catch { return {}; }
};
