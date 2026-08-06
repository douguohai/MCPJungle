import { http } from "./client";
import type { DashboardResourcesResponse } from "../types";

export const resourcesApi = {
  list: (signal?: AbortSignal) => http.get<DashboardResourcesResponse>("/resources", { signal }).then((r) => r.data),
};
