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
