import { accessHttp } from "./client";
import type {
  CreateDeviceTokenResponse,
  DeviceToken,
  DeviceTokenScope,
} from "../types";

export interface CreateDeviceTokenInput {
  name: string;
  scope: DeviceTokenScope;
  expires_at?: string;
  service_ids?: number[];
}

export const deviceTokensApi = {
  list: () =>
    accessHttp
      .get<{ device_tokens: DeviceToken[] }>("/device-tokens")
      .then((response) => response.data.device_tokens ?? []),
  create: (body: CreateDeviceTokenInput) =>
    accessHttp
      .post<CreateDeviceTokenResponse>("/device-tokens", body)
      .then((response) => response.data),
  revoke: (id: number) => accessHttp.delete(`/device-tokens/${id}`),
};
