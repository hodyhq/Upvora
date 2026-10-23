import "./IdeaFiles.scss"

import React, { lazy, Suspense, useEffect, useRef, useState } from "react"
import { Trans } from "@lingui/react/macro"
import { Button, Loader, Markdown, Modal } from "@fider/components"
import { ImageUpload } from "@fider/models"
import { Fider, fileToBase64 } from "@fider/services"

const PdfView = lazy(() => import(/* webpackChunkName: "pdfview" */ "./PdfView"))

export type FileKind = "pdf" | "md" | "txt"

const hardFileSizeLimit = 5 * 1024 * 1024
const maxAttachments = 3

// fileKind tells a file attachment's kind from its key or name, or undefined
// for anything that is not a file attachment (images, other types).
export const fileKind = (name: string): FileKind | undefined => {
  const lower = name.toLowerCase()
  if (lower.endsWith(".pdf")) return "pdf"
  if (lower.endsWith(".md") || lower.endsWith(".markdown")) return "md"
  if (lower.endsWith(".txt") || lower.endsWith(".text")) return "txt"
  return undefined
}

// isFileKey: stored file attachments live under files/, apart from images.
export const isFileKey = (bkey?: string): boolean => !!bkey && bkey.startsWith("files/")

export const fileURL = (bkey: string) => `${Fider.settings.baseURL}/static/files/${bkey}`

// fileName drops the random prefix of a stored key: files/<random>-<name>.
export const fileName = (bkey: string): string => {
  const base = bkey.substring(bkey.lastIndexOf("/") + 1)
  const dash = base.indexOf("-")
  return dash >= 0 ? base.substring(dash + 1) : base
}

const icon = (kind?: FileKind) => (kind === "pdf" ? "📄" : "📝")

interface FileViewerProps {
  bkey: string
  onClose: () => void
}

// FileViewer shows a file inside the portal: a PDF drawn by pdf.js, Markdown
// through the sanitized renderer, plain text as text. Nothing runs from it.
export const FileViewer = (props: FileViewerProps) => {
  const kind = fileKind(props.bkey)
  const url = fileURL(props.bkey)
  const [text, setText] = useState<string | undefined>(undefined)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    if (kind === "pdf") return
    let cancelled = false
    fetch(url, { credentials: "same-origin" })
      .then((res) => (res.ok ? res.text() : Promise.reject(res.status)))
      .then((body) => !cancelled && setText(body))
      .catch(() => !cancelled && setFailed(true))
    return () => {
      cancelled = true
    }
  }, [url, kind])

  return (
    <Modal.Window className="c-idea-files__viewer" isOpen={true} onClose={props.onClose} center={false} size="large">
      <Modal.Header>
        {icon(kind)} {fileName(props.bkey)}
      </Modal.Header>
      <Modal.Content>
        {kind === "pdf" ? (
          <Suspense fallback={<Loader />}>
            <PdfView url={url} />
          </Suspense>
        ) : failed ? (
          <p className="text-muted">
            <Trans id="ideafiles.load.failed">This file could not be loaded.</Trans>
          </p>
        ) : text === undefined ? (
          <Loader />
        ) : kind === "md" ? (
          <Markdown className="c-idea-files__markdown" text={text} style="full" />
        ) : (
          <pre className="c-idea-files__text">{text}</pre>
        )}
      </Modal.Content>
      <Modal.Footer>
        <a className="c-idea-files__open" href={url} target="_blank" rel="noopener noreferrer">
          <Trans id="ideafiles.open.newtab">Open in a new tab</Trans>
        </a>
        <Button variant="tertiary" onClick={props.onClose}>
          <Trans id="action.close">Close</Trans>
        </Button>
      </Modal.Footer>
    </Modal.Window>
  )
}

interface IdeaFilesListProps {
  bkeys?: string[]
}

// IdeaFilesList lists an idea's file attachments; each opens in FileViewer.
export const IdeaFilesList = (props: IdeaFilesListProps) => {
  const [open, setOpen] = useState<string | undefined>(undefined)
  const files = (props.bkeys || []).filter(isFileKey)
  if (files.length === 0) return null
  return (
    <div className="c-idea-files">
      <ul className="c-idea-files__list">
        {files.map((bkey) => (
          <li key={bkey}>
            <button type="button" className="c-idea-files__chip" onClick={() => setOpen(bkey)}>
              <span aria-hidden="true">{icon(fileKind(bkey))}</span> {fileName(bkey)}
            </button>
          </li>
        ))}
      </ul>
      {open && <FileViewer bkey={open} onClose={() => setOpen(undefined)} />}
    </div>
  )
}

interface IdeaFilesEditorProps {
  existing?: string[]
  attachments: ImageUpload[]
  onChange: (upload: ImageUpload) => void
  disabled?: boolean
  // allowAdd false keeps removal of existing files (the site switch is off).
  allowAdd?: boolean
}

// IdeaFilesEditor attaches PDF, text and Markdown files to an idea (images
// go into the description editor). New files and removals travel in the same
// attachments list as images; the server validates and scans every file.
export const IdeaFilesEditor = (props: IdeaFilesEditorProps) => {
  const input = useRef<HTMLInputElement>(null)
  const [error, setError] = useState<string | undefined>(undefined)
  const removed = new Set(props.attachments.filter((a) => a.remove && a.bkey).map((a) => a.bkey))
  const kept = (props.existing || []).filter((bkey) => isFileKey(bkey) && !removed.has(bkey))
  const pending = props.attachments.filter((a) => !a.remove && a.upload?.fileName && fileKind(a.upload.fileName))

  const keptAttachments = (props.existing || []).filter((bkey) => !removed.has(bkey)).length
  const newUploads = props.attachments.filter((a) => !a.remove && a.upload).length
  const full = keptAttachments + newUploads >= maxAttachments

  const picked = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    e.target.value = ""
    if (!file) return
    if (full) {
      setError("An idea can have up to 3 attachments, images included.")
      return
    }
    if (!fileKind(file.name)) {
      setError("Only PDF, .txt and .md files can be attached.")
      return
    }
    if (file.size > hardFileSizeLimit) {
      setError("The file size must be smaller than 5MB.")
      return
    }
    setError(undefined)
    let content: string
    try {
      content = await fileToBase64(file)
    } catch {
      setError("This file could not be read.")
      return
    }
    props.onChange({ bkey: `pending-${Math.random().toString(36).slice(2)}`, upload: { fileName: file.name, content, contentType: file.type }, remove: false })
  }

  return (
    <div className="c-form-field c-idea-files">
      <label>
        <Trans id="ideafiles.label">Files</Trans>
      </label>
      {(kept.length > 0 || pending.length > 0) && (
        <ul className="c-idea-files__list">
          {kept.map((bkey) => (
            <li key={bkey} className="c-idea-files__chip">
              <span aria-hidden="true">{icon(fileKind(bkey))}</span> {fileName(bkey)}
              {!props.disabled && (
                <button type="button" className="c-idea-files__remove" aria-label="Remove" onClick={() => props.onChange({ bkey, remove: true })}>
                  ×
                </button>
              )}
            </li>
          ))}
          {pending.map((a) => (
            <li key={a.bkey} className="c-idea-files__chip">
              <span aria-hidden="true">{icon(fileKind(a.upload?.fileName || ""))}</span> {a.upload?.fileName}
              {!props.disabled && (
                <button type="button" className="c-idea-files__remove" aria-label="Remove" onClick={() => props.onChange({ bkey: a.bkey, remove: true })}>
                  ×
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
      <input
        ref={input}
        type="file"
        hidden
        accept=".pdf,.txt,.md,.markdown,application/pdf,text/plain,text/markdown"
        onChange={picked}
        aria-label="Attach a file"
      />
      {!props.disabled && props.allowAdd !== false && (
        <Button
          variant="secondary"
          size="small"
          disabled={full}
          onClick={() => {
            if (!full) input.current?.click()
          }}
        >
          <Trans id="ideafiles.attach">Attach a PDF, text or Markdown file</Trans>
        </Button>
      )}
      <p className="text-muted text-xs mt-1">
        <Trans id="ideafiles.hint">Up to 3 attachments in total, 5 MB each.</Trans>
      </p>
      {error && (
        <p className="c-idea-files__error" role="alert">
          {error}
        </p>
      )}
    </div>
  )
}
