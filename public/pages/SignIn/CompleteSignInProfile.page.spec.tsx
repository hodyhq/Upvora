import { sameOriginPath } from "./CompleteSignInProfile.page"

describe("sameOriginPath", () => {
  const origin = "https://demo.test"
  test("keeps same-site paths", () => {
    expect(sameOriginPath("/oauth2/authorize?client_id=a", origin)).toBe("/oauth2/authorize?client_id=a")
  })
  test("refuses anything that leaves the site", () => {
    for (const bad of ["//evil.example/x", "/\\evil.example/x", "https://evil.example/", "javascript:alert(1)", "", undefined]) {
      expect(sameOriginPath(bad as any, origin)).toBeUndefined()
    }
  })
})
