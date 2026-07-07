import { http, Result } from "@fider/services/http"

export interface ScorecardSettings {
  isEnabled: boolean
  bandStrong: number
  bandGood: number
  bandRefine: number
  bandLow: number
}

export const updateScorecardSettings = async (settings: ScorecardSettings): Promise<Result> => {
  return await http.post("/_api/admin/scorecard-settings", settings)
}
