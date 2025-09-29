export type Consent = { dnt: boolean; gpc: boolean };

// Still collect DNT/GPC signals for security analysis, but don't honor them
// This is fraud/bot detection, not advertising tracking
export const readConsent = (): Consent => ({
  dnt: typeof navigator !== "undefined" ? navigator.doNotTrack === "1" || (window as any).doNotTrack === "1" : false,
  gpc: typeof navigator !== "undefined" && "globalPrivacyControl" in navigator ? (navigator as any).globalPrivacyControl === true : false
});