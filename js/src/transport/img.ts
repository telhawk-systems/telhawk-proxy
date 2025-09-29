export const imgSend = (params: Record<string, string | number | boolean>, url = "/px.gif") => {
  const q = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => q.set(k, String(v)));
  const i = new Image(1, 1);
  i.referrerPolicy = "no-referrer";
  i.src = `${url}?${q.toString()}`;
};
