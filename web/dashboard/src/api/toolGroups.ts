import { http } from "./client";
import type { DashboardCreateToolGroupInput, DashboardToolGroupsResponse } from "../types";

export const toolGroupsApi = {
  list: () => http.get<DashboardToolGroupsResponse>("/tool-groups").then((r) => r.data),
  create: (body: DashboardCreateToolGroupInput) =>
    http.post("/tool-groups", body).then((r) => r.data),
  remove: (name: string) => http.delete(`/tool-groups/${encodeURIComponent(name)}`),
};
