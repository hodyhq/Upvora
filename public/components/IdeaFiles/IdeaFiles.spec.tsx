import React from "react"
import { render, screen, fireEvent, act } from "@testing-library/react"
import { fiderMock } from "@fider/services/testing"
import { fileKind, fileName, isFileKey, IdeaFilesList, IdeaFilesEditor } from "./IdeaFiles"

beforeEach(() => {
  fiderMock.authenticated()
})

describe("IdeaFiles helpers", () => {
  test("kinds, keys and names", () => {
    expect(fileKind("files/abc-spec.PDF")).toBe("pdf")
    expect(fileKind("plan.markdown")).toBe("md")
    expect(fileKind("notes.txt")).toBe("txt")
    expect(fileKind("logo.png")).toBeUndefined()
    expect(fileKind("page.html")).toBeUndefined()
    expect(isFileKey("files/abc-spec.pdf")).toBe(true)
    expect(isFileKey("attachments/abc-logo.png")).toBe(false)
    expect(fileName("files/0123abc-quarterly-plan.pdf")).toBe("quarterly-plan.pdf")
  })
})

describe("<IdeaFilesList />", () => {
  test("lists only file attachments, never images", () => {
    render(<IdeaFilesList bkeys={["files/a1-spec.pdf", "attachments/a2-logo.png", "files/a3-notes.md"]} />)
    expect(screen.getByText(/spec\.pdf/)).toBeTruthy()
    expect(screen.getByText(/notes\.md/)).toBeTruthy()
    expect(screen.queryByText(/logo\.png/)).toBeNull()
  })

  test("renders nothing without files", () => {
    const { container } = render(<IdeaFilesList bkeys={["attachments/a2-logo.png"]} />)
    expect(container.innerHTML).toBe("")
  })
})

describe("<IdeaFilesEditor />", () => {
  test("refuses other file types before upload", async () => {
    const onChange = jest.fn()
    const { container } = render(<IdeaFilesEditor attachments={[]} onChange={onChange} />)
    const input = container.querySelector("input[type=file]") as HTMLInputElement
    await act(async () => {
      fireEvent.change(input, { target: { files: [new File(["<svg/>"], "x.svg", { type: "image/svg+xml" })] } })
    })
    expect(onChange).not.toHaveBeenCalled()
    expect(screen.getByRole("alert")).toBeTruthy()
  })

  test("shows kept files, hides removed ones, and removes on click", () => {
    const onChange = jest.fn()
    render(
      <IdeaFilesEditor
        existing={["files/k1-keep.pdf", "files/k2-gone.md", "attachments/i-logo.png"]}
        attachments={[
          { bkey: "files/k2-gone.md", remove: true },
          { bkey: "pending-1", upload: { fileName: "new.txt", content: "aGk=" }, remove: false },
        ]}
        onChange={onChange}
      />
    )
    expect(screen.getByText(/keep\.pdf/)).toBeTruthy()
    expect(screen.queryByText(/gone\.md/)).toBeNull()
    expect(screen.queryByText(/logo\.png/)).toBeNull()
    expect(screen.getByText(/new\.txt/)).toBeTruthy()
    fireEvent.click(screen.getAllByLabelText("Remove")[0])
    expect(onChange).toHaveBeenCalledWith({ bkey: "files/k1-keep.pdf", remove: true })
  })
})

describe("<IdeaFilesEditor /> limits", () => {
  test("full at 3 attachments, images included: the picker is off", () => {
    const { container } = render(<IdeaFilesEditor existing={["attachments/i-1.png", "files/a-2.pdf", "files/b-3.md"]} attachments={[]} onChange={jest.fn()} />)
    const attach = container.querySelector("button.c-button") as HTMLButtonElement
    expect(attach).not.toBeNull()
    expect(attach.disabled || attach.classList.contains("c-button--disabled")).toBe(true)
  })

  test("switch off: existing files can be removed, none added", () => {
    const { container } = render(<IdeaFilesEditor existing={["files/a-2.pdf"]} attachments={[]} onChange={jest.fn()} allowAdd={false} />)
    expect(container.querySelector("button.c-button")).toBeNull()
    expect(screen.getByLabelText("Remove")).toBeTruthy()
  })
})
