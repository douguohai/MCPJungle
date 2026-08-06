import { http } from "./client";
import type { DashboardDiagnosticsResponse } from "../types";

export const diagnosticsApi = {
  get: (signal?: AbortSignal) => http.get<DashboardDiagnosticsResponse>("/diagnostics", { signal }).then((r) => r.data),
};
