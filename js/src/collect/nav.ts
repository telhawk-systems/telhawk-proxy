export const readNav = () => {
  if (typeof navigator === "undefined") return {};
  return {
    ua: navigator.userAgent || "",
    lang: navigator.language || "",
    langs: (navigator.languages || []).slice(0, 5),
    plat: (navigator as any).platform || "",
    hc: (navigator as any).hardwareConcurrency ?? null,
    vendor: (navigator as any).vendor || ""
  };
};
