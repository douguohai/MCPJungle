import { http } from "./client";
import type {
  DashboardRegisterServerInput,
  DashboardRegisterServerResponse,
  DashboardServersResponse,
} from "../types";

export const serversApi = {
  list: () => http.get<DashboardServersResponse>("/servers").then((r) => r.data),
  create: (body: DashboardRegisterServerInput) =>
    http.post<DashboardRegisterServerResponse>("/servers", body).then((r) => r.data),
  remove: (name: string) => http.delete(`/servers/${encodeURIComponent(name)}`),
  setEnabled: (name: string, enabled: boolean) =>
    http.patch(`/servers/${encodeURIComponent(name)}/enabled`, { enabled }),
};
