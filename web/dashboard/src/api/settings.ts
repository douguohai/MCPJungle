import { http } from "./client";

export interface DashboardSettings {
  system_display_name: string;
}

export const defaultDashboardSettings: DashboardSettings = {
  system_display_name: "MCPJungle",
};

export const settingsApi = {
  get: () =>
    http
      .get<DashboardSettings>("/settings")
      .then((r) => r.data),
  update: (input: DashboardSettings) =>
    http
      .put<DashboardSettings>("/settings", input)
      .then((r) => r.data),
};
