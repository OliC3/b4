import { apiDelete, apiPost, apiGet } from "./apiClient";
import { B4SetConfig } from "@b4.sets";
import { DiscoveryResponse, DiscoverySuite } from "@b4.discovery";

export const discoveryApi = {
  start: (check_url: string) =>
    apiPost<DiscoveryResponse>("/api/discovery/start", { check_url }),
  status: (id: string) => apiGet<DiscoverySuite>(`/api/discovery/status/${id}`),
  cancel: (id: string) => apiDelete(`/api/discovery/cancel/${id}`),
  addPresetAsSet: (preset: B4SetConfig) =>
    apiPost<B4SetConfig>("/api/discovery/add", preset),
};
