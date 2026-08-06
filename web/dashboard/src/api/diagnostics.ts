import { http } from "./client";
import type { DashboardDiagnosticsResponse } from "../types";

export const diagnosticsApi = {
  get: () => http.get<DashboardDiagnosticsResponse>("/diagnostics").then((r) => r.data),
};
