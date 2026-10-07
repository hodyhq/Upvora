import React, { useRef, useState } from "react"
import { Trans } from "@lingui/react/macro"
import { Button, TenantLogo } from "@fider/components"
import { actions, Fider } from "@fider/services"

import "./OAuthConsent.page.scss"

interface OAuthConsentPageProps {
  error?: string
  clientName?: string
  redirectHost?: string
  scope?: string
  clientId?: string
  redirectUri?: string
  codeChallenge?: string
  codeChallengeMethod?: string
  state?: string
  // Set when the connection goes through an MCP-only public address.
  connectingThrough?: string
}

// isSafeRedirect refuses schemes a browser would execute. The server already
// rejects them at client registration; this is defense in depth.
export const isSafeRedirect = (target: string): boolean => {
  try {
    const protocol = new URL(target).protocol.toLowerCase()
    return !["javascript:", "data:", "vbscript:", "file:", "blob:", "about:"].includes(protocol)
  } catch {
    return false
  }
}

const OAuthConsentPage = (props: OAuthConsentPageProps) => {
  const [busy, setBusy] = useState(false)
  const [failed, setFailed] = useState(false)
  // One decision at a time: a ref updates synchronously, so a double click or
  // Allow-then-Cancel cannot send a second decision while one is in flight.
  const inFlight = useRef(false)
  const siteName = Fider.session.tenant?.name
  const userName = Fider.session.isAuthenticated ? Fider.session.user.name : ""

  const decide = async (approve: boolean) => {
    if (inFlight.current) return
    inFlight.current = true
    setBusy(true)
    setFailed(false)
    const result = await actions.decideOAuthAuthorization({
      clientId: props.clientId || "",
      redirectUri: props.redirectUri || "",
      codeChallenge: props.codeChallenge || "",
      codeChallengeMethod: props.codeChallengeMethod || "",
      scope: props.scope || "",
      state: props.state || "",
      approve,
    })
    if (result.ok && result.data && isSafeRedirect(result.data.redirect)) {
      window.location.href = result.data.redirect
      return
    }
    inFlight.current = false
    setFailed(true)
    setBusy(false)
  }

  return (
    <div id="p-oauth-consent" className="page container">
      <div className="p-oauth-consent__card">
        <div className="p-oauth-consent__logo">
          <TenantLogo size={50} />
        </div>
        {props.error ? (
          <p className="p-oauth-consent__error" role="alert">
            {props.error}
          </p>
        ) : (
          <>
            <h1 className="p-oauth-consent__title">
              <Trans id="oauth.consent.title">
                Allow <strong>{props.clientName}</strong> to use {siteName} as you?
              </Trans>
            </h1>
            <p className="p-oauth-consent__text">
              {props.scope === "upvora:read" ? (
                <Trans id="oauth.consent.scope.read">It will be able to read everything you can see here, but not change anything.</Trans>
              ) : (
                <Trans id="oauth.consent.scope.full">
                  It will be able to do everything you can do here: share and edit ideas, comment, vote, and use any settings your role allows.
                </Trans>
              )}
            </p>
            {userName && (
              <p className="p-oauth-consent__text p-oauth-consent__signed-in">
                <Trans id="oauth.consent.signedInAs">
                  Signed in as <strong>{userName}</strong>. Not you? Close this page and sign out first.
                </Trans>
              </p>
            )}
            <p className="p-oauth-consent__text text-muted">
              <Trans id="oauth.consent.redirect">After you choose, you will return to {props.redirectHost}.</Trans>
            </p>
            {props.connectingThrough && (
              <p className="p-oauth-consent__text text-muted">
                <Trans id="oauth.consent.through">
                  You are connecting through {props.connectingThrough}, this site&apos;s public address for AI assistants.
                </Trans>
              </p>
            )}
            {failed && (
              <p className="p-oauth-consent__error" role="alert">
                <Trans id="oauth.consent.failed">Something went wrong. Please try again, or close this page and reconnect from your app.</Trans>
              </p>
            )}
            <div className="p-oauth-consent__actions">
              <Button variant="primary" disabled={busy} onClick={() => decide(true)}>
                <Trans id="oauth.consent.allow">Allow</Trans>
              </Button>
              <Button variant="secondary" disabled={busy} onClick={() => decide(false)}>
                <Trans id="oauth.consent.cancel">Cancel</Trans>
              </Button>
            </div>
          </>
        )}
      </div>
    </div>
  )
}

export default OAuthConsentPage
