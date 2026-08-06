import { http } from "./client";
import type { DashboardPromptsResponse } from "../types";

export const promptsApi = {
  list: (signal?: AbortSignal) => http.get<DashboardPromptsResponse>("/prompts", { signal }).then((r) => r.data),
  setEnabled: (name: string, enabled: boolean) =>
    http.patch(`/prompts/${encodeURIComponent(name)}/enabled`, { enabled }),
};
