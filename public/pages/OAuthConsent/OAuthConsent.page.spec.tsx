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

  test("one decision only, even on a double click", async () => {
    let resolvePost: (v: any) => void = () => undefined
    const post = jest.fn(() => new Promise((r) => (resolvePost = r)))
    http.post = post as any

    render(<OAuthConsentPage {...props} />)
    const [allow, cancel] = screen.getAllByRole("button")
    await act(async () => {
      fireEvent.click(allow)
      fireEvent.click(allow)
      fireEvent.click(cancel)
    })
    expect(post).toHaveBeenCalledTimes(1)
    await act(async () => resolvePost({ ok: false }))
  })

  test("a failed decision shows an error and allows a retry", async () => {
    const post = jest.fn(() => Promise.resolve({ ok: false }))
    http.post = post as any

    render(<OAuthConsentPage {...props} />)
    await act(async () => {
      fireEvent.click(screen.getAllByRole("button")[0])
    })
    expect(screen.getByRole("alert")).toBeTruthy()
    await act(async () => {
      fireEvent.click(screen.getAllByRole("button")[0])
    })
    expect(post).toHaveBeenCalledTimes(2)
  })
})

describe("<OAuthConsentPage /> on an MCP-only address", () => {
  // Trans renders no text without an i18n provider, so count the notes.
  const notes = (container: HTMLElement) => container.querySelectorAll(".p-oauth-consent__text.text-muted").length

  test("adds a note naming the address the connection goes through", () => {
    const { container } = render(<OAuthConsentPage {...props} connectingThrough="mcp.example.com" />)
    expect(notes(container)).toBe(2)
  })
  test("adds nothing on the board itself", () => {
    const { container } = render(<OAuthConsentPage {...props} />)
    expect(notes(container)).toBe(1)
  })
})
