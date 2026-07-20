import { http } from "./client";
import type { DashboardToolsResponse } from "../types";

export const toolsApi = {
  list: () => http.get<DashboardToolsResponse>("/tools").then((r) => r.data),
  setEnabled: (name: string, enabled: boolean) =>
    http.patch(`/tools/${encodeURIComponent(name)}/enabled`, { enabled }),
};
