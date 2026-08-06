import { http } from "./client";
import type { DashboardCreateToolCollectionInput, DashboardToolCollectionsResponse } from "../types";

export const toolCollectionsApi = {
  list: (signal?: AbortSignal) => http.get<DashboardToolCollectionsResponse>("/tool-collections", { signal }).then((r) => r.data),
  create: (body: DashboardCreateToolCollectionInput) =>
    http.post("/tool-collections", body).then((r) => r.data),
  remove: (name: string) => http.delete(`/tool-collections/${encodeURIComponent(name)}`),
};
