import { http } from "./client";
import type { DashboardOverviewResponse } from "../types";

export const overviewApi = {
  get: (signal?: AbortSignal) => http.get<DashboardOverviewResponse>("/overview", { signal }).then((r) => r.data),
};
