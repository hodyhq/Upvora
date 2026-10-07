import React from "react"
import { render } from "@testing-library/react"
import { SocialSignInButton } from "./SocialSignInButton"

describe("<SocialSignInButton />", () => {
  test("carries a multi-parameter return URL intact", () => {
    const target = "https://demo.test/oauth2/authorize?client_id=abc&redirect_uri=http%3A%2F%2F127.0.0.1%3A3000%2Fcb&state=x y"
    const { container } = render(<SocialSignInButton option={{ displayName: "Google", url: "https://demo.test/oauth/google" }} redirectTo={target} />)
    const href = container.querySelector("a")!.getAttribute("href")!
    expect(new URL(href).searchParams.get("redirect")).toBe(target)
  })
})
