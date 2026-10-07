import { http, Result } from "@fider/services"

export interface OAuthDecisionRequest {
  clientId: string
  redirectUri: string
  codeChallenge: string
  codeChallengeMethod: string
  scope: string
  state: string
  approve: boolean
}

export const decideOAuthAuthorization = async (request: OAuthDecisionRequest): Promise<Result<{ redirect: string }>> => {
  return await http.post<{ redirect: string }>("/_api/oauth2/authorize", request)
}

export type MCPMinRole = "visitor" | "collaborator" | "administrator"

export interface MCPSettings {
  enabled: boolean
  minRole: MCPMinRole
  dcrEnabled: boolean
}

export interface MCPClient {
  id: number
  clientId: string
  name: string
  redirectUris: string[]
  createdByAdmin: boolean
  createdAt: string
}

export const updateMCPSettings = async (settings: MCPSettings): Promise<Result> => {
  return await http.post("/_api/admin/settings/mcp", settings)
}

export const createMCPClient = async (name: string, redirectUris: string[]): Promise<Result<MCPClient>> => {
  return await http.post<MCPClient>("/_api/admin/mcp/clients", { name, redirectUris })
}

export const deleteMCPClient = async (id: number): Promise<Result> => {
  return await http.delete(`/_api/admin/mcp/clients/${id}`)
}
