import { http } from "./client";
import type { DashboardCreateToolCollectionInput, DashboardToolCollectionsResponse } from "../types";

export const toolCollectionsApi = {
  list: () => http.get<DashboardToolCollectionsResponse>("/tool-collections").then((r) => r.data),
  create: (body: DashboardCreateToolCollectionInput) =>
    http.post("/tool-collections", body).then((r) => r.data),
  remove: (name: string) => http.delete(`/tool-collections/${encodeURIComponent(name)}`),
};
