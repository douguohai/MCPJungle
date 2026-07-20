import { http } from "./client";
import type { DashboardResourcesResponse } from "../types";

export const resourcesApi = {
  list: () => http.get<DashboardResourcesResponse>("/resources").then((r) => r.data),
};
