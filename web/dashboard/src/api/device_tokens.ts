import { http } from "./client";

export interface DeviceToken {
  id: number;
  name: string;
  status: string;
  scope_mode: string;
  token_prefix: string;
  expires_at: string;
  created_at: string;
}

export interface CreateDeviceTokenResponse {
  raw_token: string;
  token: DeviceToken;
}

export const deviceTokensApi = {
  list: (signal?: AbortSignal) => http.get<DeviceToken[]>("/device-tokens", { signal }).then((r) => r.data),
  create: (body: {
    name: string;
    scope_mode: string;
    restricted_server_names?: string[];
  }) =>
    http
      .post<CreateDeviceTokenResponse>("/device-tokens", body)
      .then((r) => r.data),
  revoke: (id: number) =>
    http.post(`/device-tokens/${id}/revoke`).then((r) => r.data),
  remove: (id: number) => http.delete(`/device-tokens/${id}`),
};
