import { http, notify } from "@fider/services"
import { fiderMock } from "@fider/services/testing"
import PrivacySettingsPage from "./PrivacySettings.page"

type Deferred = { resolve: (ok: boolean) => void }

// Each http.post call waits until the test resolves it, so saves can finish out of order.
const queuePosts = () => {
  const pending: Deferred[] = []
  http.post = jest.fn(
    () =>
      new Promise((resolve) => {
        pending.push({ resolve: (ok) => resolve({ ok, data: null as any }) })
      })
  ) as any
  return pending
}

// Instance with setState applied synchronously (no DOM needed for the save logic).
const newPage = () => {
  const page = new PrivacySettingsPage({})
  page.setState = ((s: any, cb?: () => void) => {
    page.state = { ...page.state, ...s }
    if (cb) cb()
  }) as any
  return page
}

const update = (page: PrivacySettingsPage, patch: object) => (page as any).updatePrivacy(patch)
const flush = () => new Promise((r) => setTimeout(r, 0))

beforeEach(() => {
  fiderMock.authenticated()
  jest.spyOn(notify, "success").mockResolvedValue(undefined) // toastify pulls in CSS jest cannot parse
})

describe("PrivacySettings save", () => {
  test("a failed save rolls back to the last confirmed settings", async () => {
    const pending = queuePosts()
    const page = newPage()

    update(page, { membersPrivateIdeas: true })
    pending[0].resolve(false)
    await flush()

    expect(page.state.membersPrivateIdeas).toBeFalsy()
  })

  test("a slow failed save does not undo a newer successful save", async () => {
    const pending = queuePosts()
    const page = newPage()

    update(page, { membersPrivateIdeas: true }) // save A
    update(page, { isModerationEnabled: true }) // save B, includes A's change
    pending[1].resolve(true)
    await flush()
    pending[0].resolve(false)
    await flush()

    expect(page.state.membersPrivateIdeas).toBe(true)
    expect(page.state.isModerationEnabled).toBe(true)
  })
})
