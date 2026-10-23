import React, { useEffect, useRef, useState } from "react"
import { Trans } from "@lingui/react/macro"

interface PdfViewProps {
  url: string
}

// PdfView draws a PDF's pages onto canvases with pdf.js: no PDF scripting, no
// annotation or form layer (links are not clickable), and eval switched off.
// shortcut: renders every page up front; fine for 5 MB attachments, add page virtualisation if large PDFs become common.
const PdfView = (props: PdfViewProps) => {
  const host = useRef<HTMLDivElement>(null)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let cancelled = false
    // Each viewer has its own worker and loading task, both ended on close,
    // so one viewer closing never disturbs another.
    let task: { promise: Promise<any>; destroy: () => Promise<void> } | undefined
    let worker: Worker | undefined
    const load = async () => {
      try {
        const pdfjs: any = await import(/* webpackChunkName: "pdfjs" */ "pdfjs-dist")
        if (cancelled) return
        worker = new Worker(new URL("pdfjs-dist/build/pdf.worker.min.mjs", import.meta.url), { type: "module" })
        const loading = pdfjs.getDocument({ url: props.url, isEvalSupported: false, enableXfa: false, worker: new pdfjs.PDFWorker({ port: worker }) })
        task = loading
        const doc = await loading.promise
        const container = host.current
        if (cancelled) return
        if (!doc || !container) return
        const width = container.clientWidth || 800
        const ratio = window.devicePixelRatio || 1
        for (let n = 1; n <= doc.numPages && !cancelled; n++) {
          const page = await doc.getPage(n)
          const scale = width / page.getViewport({ scale: 1 }).width
          const viewport = page.getViewport({ scale: scale * ratio })
          const canvas = document.createElement("canvas")
          canvas.width = viewport.width
          canvas.height = viewport.height
          canvas.className = "c-idea-files__page"
          canvas.setAttribute("role", "img")
          canvas.setAttribute("aria-label", `Page ${n} of ${doc.numPages}`)
          container.appendChild(canvas)
          await page.render({ canvas, canvasContext: canvas.getContext("2d"), viewport }).promise
        }
      } catch {
        if (!cancelled) setFailed(true)
      }
    }
    load()
    return () => {
      cancelled = true
      const ending = task ? task.destroy().catch(() => undefined) : Promise.resolve()
      ending.then(() => worker?.terminate())
    }
  }, [props.url])

  if (failed) {
    return (
      <p className="text-muted">
        <Trans id="ideafiles.pdf.failed">This PDF could not be displayed. Open it in a new tab instead.</Trans>
      </p>
    )
  }
  return <div ref={host} className="c-idea-files__pdf" />
}

export default PdfView
