import { http, Result } from "@fider/services/http"
import { AIMessage } from "@fider/models"

export interface AIFinalizeResponse {
  title: string
  description: string
  brief: string
  tags: string[]
}

export const aiIdeate = async (productId: number, messages: AIMessage[]): Promise<Result<{ reply: string; ready: boolean }>> => {
  return http.post<{ reply: string; ready: boolean }>(`/api/v1/ai/ideate`, { productId, messages })
}

export const aiFinalize = async (productId: number, messages: AIMessage[]): Promise<Result<AIFinalizeResponse>> => {
  return http.post<AIFinalizeResponse>(`/api/v1/ai/finalize`, { productId, messages })
}

export const getIdeaBrief = async (postNumber: number): Promise<Result<{ content: string; createdAt: string }>> => {
  return http.get<{ content: string; createdAt: string }>(`/api/v1/posts/${postNumber}/brief`)
}

export const updateAISettings = async (settings: {
  enabled: boolean
  provider: string
  apiKey: string
  model: string
  customBaseUrl: string
  customModel: string
}): Promise<Result> => {
  return http.post(`/_api/admin/settings/ai`, settings)
}

export const upsertAIAgent = async (agent: { productId: number | null; description: string; instructions: string; enabled: boolean }): Promise<Result> => {
  return http.post(`/_api/admin/ai/agents`, agent)
}
