import { http, notify } from "@fider/services"
import { fiderMock } from "@fider/services/testing"
import ManageMCPPage from "./ManageMCP.page"

type Pending = { resolve: (ok: boolean, data?: any) => void }

const queuePosts = () => {
  const pending: Pending[] = []
  const calls: any[] = []
  http.post = jest.fn(
    (_url: string, body: any) =>
      new Promise((resolve) => {
        calls.push(body)
        pending.push({ resolve: (ok, data) => resolve({ ok, data: data ?? null }) })
      })
  ) as any
  return { pending, calls }
}

const newPage = () => {
  const page = new ManageMCPPage({ clients: [], mcpUrl: "https://demo.test/mcp" })
  page.setState = ((s: any, cb?: () => void) => {
    page.state = { ...page.state, ...(typeof s === "function" ? s(page.state) : s) }
    if (cb) cb()
  }) as any
  return page
}

const flush = () => new Promise((r) => setTimeout(r, 0))

beforeEach(() => {
  fiderMock.authenticated()
  jest.spyOn(notify, "success").mockResolvedValue(undefined)
})

describe("ManageMCP settings", () => {
  test("saves the whole settings object and rolls back on failure", async () => {
    const { pending, calls } = queuePosts()
    const page = newPage()
    expect(page.state.enabled).toBeFalsy()

    ;(page as any).updateSettings({ enabled: true })
    await flush()
    expect(calls[0]).toEqual({ enabled: true, minRole: "administrator", dcrEnabled: true })
    pending[0].resolve(false)
    await flush()
    expect(page.state.enabled).toBeFalsy()
  })

  test("a registered client is added to the list", async () => {
    const { pending } = queuePosts()
    const page = newPage()

    ;(page as any).createClient("Team Claude", "https://claude.ai/api/mcp/auth_callback")
    await flush()
    pending[0].resolve(true, { id: 9, clientId: "cid", name: "Team Claude", redirectUris: [], createdByAdmin: true, createdAt: "" })
    await flush()
    expect(page.state.clients.map((c: any) => c.clientId)).toEqual(["cid"])
  })

  test("removing a client needs a confirmation", async () => {
    const del = jest.fn(() => Promise.resolve({ ok: true }))
    http.delete = del as any
    const page = new ManageMCPPage({ clients: [{ id: 3, clientId: "c", name: "n", redirectUris: [], createdByAdmin: false, createdAt: "" }], mcpUrl: "" })
    page.setState = ((s: any, cb?: () => void) => {
      page.state = { ...page.state, ...(typeof s === "function" ? s(page.state) : s) }
      if (cb) cb()
    }) as any

    ;(page as any).requestRemove(3)
    await flush()
    expect(del).not.toHaveBeenCalled()
    expect(page.state.confirmRemoveId).toBe(3)

    await (page as any).deleteClient(3)
    expect(del).toHaveBeenCalledTimes(1)
    expect(page.state.clients).toHaveLength(0)
  })
})
