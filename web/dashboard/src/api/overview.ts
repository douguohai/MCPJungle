import { http } from "./client";
import type { DashboardOverviewResponse } from "../types";

export const overviewApi = {
  get: () => http.get<DashboardOverviewResponse>("/overview").then((r) => r.data),
};
