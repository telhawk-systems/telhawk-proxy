export type Consent = { dnt: boolean; gpc: boolean };
export const readConsent = (): Consent => ({
  dnt: typeof navigator !== "undefined" ? navigator.doNotTrack === "1" || (window as any).doNotTrack === "1" : false,
  gpc: typeof navigator !== "undefined" && "globalPrivacyControl" in navigator ? (navigator as any).globalPrivacyControl === true : false
});