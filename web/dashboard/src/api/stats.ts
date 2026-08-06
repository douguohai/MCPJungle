import { http } from "./client";
import type { CallStatsResponse } from "../types";

export const statsApi = {
  list: () => http.get<CallStatsResponse>("/stats").then((r) => r.data),
};
