import React from "react"
import { render, screen, fireEvent } from "@testing-library/react"
import { act } from "react"
import { http } from "@fider/services"
import { fiderMock } from "@fider/services/testing"
import OAuthConsentPage, { isSafeRedirect } from "./OAuthConsent.page"

const props = {
  clientName: "Claude",
  redirectHost: "claude.ai",
  scope: "upvora",
  clientId: "cid",
  redirectUri: "https://claude.ai/cb",
  codeChallenge: "ch",
  codeChallengeMethod: "S256",
  state: "st",
}

beforeEach(() => {
  fiderMock.authenticated()
})

describe("isSafeRedirect", () => {
  test("allows https, loopback http and native app schemes", () => {
    expect(isSafeRedirect("https://claude.ai/cb?code=x")).toBe(true)
    expect(isSafeRedirect("http://127.0.0.1:3000/cb")).toBe(true)
    expect(isSafeRedirect("cursor://anysphere/cb")).toBe(true)
  })
  test("blocks executable or opaque schemes", () => {
    expect(isSafeRedirect("javascript:alert(1)")).toBe(false)
    expect(isSafeRedirect("JaVaScRiPt:alert(1)")).toBe(false)
    expect(isSafeRedirect("data:text/html,x")).toBe(false)
    expect(isSafeRedirect("vbscript:x")).toBe(false)
    expect(isSafeRedirect("not a url")).toBe(false)
  })
})

describe("<OAuthConsentPage />", () => {
  test("shows the application and posts the decision", async () => {
    const post = jest.fn(() => Promise.resolve({ ok: true, data: { redirect: "javascript:alert(1)" } }))
    http.post = post as any

    // Trans renders empty under the jest lingui mock, so buttons are found by order: Allow, Cancel.
    render(<OAuthConsentPage {...props} />)
    const [allow, cancel] = screen.getAllByRole("button")

    await act(async () => {
      fireEvent.click(allow)
    })
    expect(post).toHaveBeenCalledWith("/_api/oauth2/authorize", expect.objectContaining({ approve: true, clientId: "cid", state: "st" }))

    await act(async () => {
      fireEvent.click(cancel)
    })
    expect(post).toHaveBeenLastCalledWith("/_api/oauth2/authorize", expect.objectContaining({ approve: false }))
  })

  test("shows an error without decision buttons", () => {
    render(<OAuthConsentPage error="This application is not registered on this site." />)
    expect(screen.getByText("This application is not registered on this site.")).toBeTruthy()
    expect(screen.queryAllByRole("button")).toHaveLength(0)
  })
})
