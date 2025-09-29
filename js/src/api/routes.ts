export type RouteConfig = { endpoint?: string };
export const pickEndpoint = (cfg: RouteConfig): string => cfg.endpoint || "/collect";