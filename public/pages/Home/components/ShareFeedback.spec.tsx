import React from "react"
import { render, screen } from "@testing-library/react"
import { FiderContext } from "@fider/services"
import { fiderMock, setupModalRoot } from "@fider/services/testing"
import { ShareFeedback } from "./ShareFeedback"

const open = (tenant: Record<string, unknown>, signedIn = true) => {
  const fider = signedIn ? fiderMock.authenticated() : fiderMock.notAuthenticated()
  Object.assign(fider.session.tenant, { aiEnabled: true, aiAgents: [{ productId: null }], products: [], allowedSchemes: "", ...tenant })
  return render(
    <FiderContext.Provider value={fider}>
      <ShareFeedback isOpen={true} placeholder="" onClose={() => undefined} tags={[]} />
    </FiderContext.Provider>
  )
}

const titleField = () => document.querySelector("#input-title")

describe("<ShareFeedback /> Vora-only", () => {
  beforeEach(() => {
    localStorage.clear()
    setupModalRoot()
  })

  test("off: the plain form is there next to Vora", () => {
    open({ voraRequired: false })
    expect(titleField()).not.toBeNull()
    expect(screen.getByText(/Talk it through with Vora/)).toBeTruthy()
  })

  test("on: only Vora until it has drafted the idea", () => {
    open({ voraRequired: true })
    expect(titleField()).toBeNull()
    expect(screen.getByText(/Talk it through with Vora/)).toBeTruthy()
  })

  test("on, but no agent serves the product: the plain form stays", () => {
    open({ voraRequired: true, aiAgents: [] })
    expect(titleField()).not.toBeNull()
  })

  test("file attachments switch adds the files control", () => {
    open({ voraRequired: false, fileAttachmentsEnabled: true })
    expect(screen.getByLabelText("Attach a file")).toBeTruthy()
  })
})
