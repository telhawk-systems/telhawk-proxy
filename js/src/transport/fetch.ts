export const fetchSend = async (body: string, endpoint: string) => {
  await fetch(endpoint, { method: "POST", keepalive: true, headers: { "Content-Type": "application/json" }, body });
};
