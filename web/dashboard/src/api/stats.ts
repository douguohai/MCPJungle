import { http } from "./client";
import type { CallStatsResponse } from "../types";

export const statsApi = {
  list: (signal?: AbortSignal) => http.get<CallStatsResponse>("/stats", { signal }).then((r) => r.data),
};
