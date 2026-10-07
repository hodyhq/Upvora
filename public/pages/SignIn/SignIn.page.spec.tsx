import { TenantStatus } from "@fider/models"
import { signInIntro } from "./SignIn.page"

describe("signInIntro", () => {
  test("only a locked site says it is locked", () => {
    expect(signInIntro({ isPrivate: true, status: TenantStatus.Active })).toBe("private")
    expect(signInIntro({ isPrivate: false, status: TenantStatus.Locked })).toBe("locked")
    expect(signInIntro({ isPrivate: true, status: TenantStatus.Locked })).toBe("locked")
    // a public, active site reached with ?redirect= (e.g. connecting an MCP client)
    expect(signInIntro({ isPrivate: false, status: TenantStatus.Active })).toBe("default")
  })
})
