export const readScreen = () => {
  if (typeof window === "undefined") return {};
  const s = window.screen || ({} as any);
  const dpr = (window as any).devicePixelRatio || 1;
  return { w: s.width, h: s.height, aw: s.availWidth, ah: s.availHeight, dpr };
};
