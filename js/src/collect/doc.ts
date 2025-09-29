export const readDoc = () => {
  if (typeof document === "undefined") return {};
  return {
    ref: document.referrer || "",
    vis: (document as any).visibilityState || "visible",
    hasFocus: !!document.hasFocus?.()
  };
};
