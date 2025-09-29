export const imgSend = (params: Record<string, string | number | boolean>, endpoint = "/px.gif") => {
  const q = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => q.set(k, String(v)));
  const i = new Image(1, 1);
  i.referrerPolicy = "no-referrer";
  // Ensure endpoint supports query params
  const separator = endpoint.includes('?') ? '&' : '?';
  i.src = `${endpoint}${separator}${q.toString()}`;
};
