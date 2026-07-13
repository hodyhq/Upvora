import { http, Result } from "@fider/services/http"

export interface SystemStatus {
  currentVersion: string
  latest: {
    tag: string
    url: string
    checkedAt: string
  } | null
  state: "up-to-date" | "update-available" | "ahead" | "unknown"
  updaterConfigured: boolean
  updateStatus: "idle" | "requested" | "running" | "done" | "error"
  updateLog: string
}

export const getSystemStatus = async (force = false): Promise<Result<SystemStatus>> => {
  return http.get<SystemStatus>(`/_api/admin/system/status${force ? "?force=true" : ""}`)
}

export const triggerSystemUpdate = async (): Promise<Result> => {
  return http.post(`/_api/admin/system/update`)
}
