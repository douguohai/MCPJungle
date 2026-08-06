import { http } from "./client";
import type { DashboardPromptsResponse } from "../types";

export const promptsApi = {
  list: () => http.get<DashboardPromptsResponse>("/prompts").then((r) => r.data),
  setEnabled: (name: string, enabled: boolean) =>
    http.patch(`/prompts/${encodeURIComponent(name)}/enabled`, { enabled }),
};
