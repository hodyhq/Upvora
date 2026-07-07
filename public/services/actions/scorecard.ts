import { http, Result } from "@fider/services/http"
import { ScorecardField } from "@fider/models"

export interface ScorecardSettings {
  isEnabled: boolean
  bandStrong: number
  bandGood: number
  bandRefine: number
  bandLow: number
  triggerStatusSlug: string
}

export const updateScorecardSettings = async (settings: ScorecardSettings): Promise<Result> => {
  return await http.post("/_api/admin/scorecard-settings", settings)
}

export interface CreateScorecardFieldRequest {
  key: string
  label: string
  groupKey: string
  type: "text" | "note" | "date" | "number" | "url" | "choice" | "score"
  choices?: unknown
  weight?: number
  question?: string
  sortOrder: number
}

export const createScorecardField = async (req: CreateScorecardFieldRequest): Promise<Result<ScorecardField>> => {
  return await http.post<ScorecardField>("/_api/admin/scorecard-fields", req)
}

export interface UpdateScorecardFieldRequest {
  label: string
  choices?: unknown
  weight?: number
  question?: string
  sortOrder: number
  isActive: boolean
}

export const updateScorecardField = async (id: number, req: UpdateScorecardFieldRequest): Promise<Result> => {
  return await http.put(`/_api/admin/scorecard-fields/${id}`, req)
}

export const deleteScorecardField = async (id: number): Promise<Result> => {
  return await http.delete(`/_api/admin/scorecard-fields/${id}`)
}

export interface ScorecardRecord {
  id: number
  postId?: number
  title: string
  values: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export interface CreateScorecardRequest {
  postId?: number
  title?: string
}

export const createScorecard = async (req: CreateScorecardRequest): Promise<Result<ScorecardRecord>> => {
  return await http.post<ScorecardRecord>("/_api/scorecards", req)
}

export interface UpdateScorecardRequest {
  title: string
  values: Record<string, unknown>
}

export const updateScorecard = async (id: number, req: UpdateScorecardRequest): Promise<Result> => {
  return await http.put(`/_api/scorecards/${id}`, req)
}
