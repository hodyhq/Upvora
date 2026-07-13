import "./InternalNotesPanel.scss"

import React, { useEffect, useRef, useState } from "react"
import { InternalNote } from "@fider/models"
import { actions } from "@fider/services"

interface InternalNotesPanelProps {
  postNumber: number
  initialNote?: InternalNote
}

// The shared team note for a post. Rendered on the post page AND the linked
// scorecard - both edit the same row server-side, so they can never drift.
// Collaborator-only: parents must gate rendering; the API enforces it too.
export const InternalNotesPanel = (props: InternalNotesPanelProps) => {
  const [note, setNote] = useState<InternalNote | undefined>(props.initialNote)
  const [content, setContent] = useState(props.initialNote?.content ?? "")
  const [state, setState] = useState<"idle" | "dirty" | "saving" | "saved">("idle")
  const timer = useRef<number>()
  const latest = useRef(content)

  useEffect(() => {
    if (props.initialNote === undefined) {
      actions.getInternalNote(props.postNumber).then((result) => {
        if (result.ok) {
          setNote(result.data)
          setContent(result.data.content)
          latest.current = result.data.content
        }
      })
    }
  }, [props.postNumber])

  const save = async (value: string) => {
    setState("saving")
    const result = await actions.setInternalNote(props.postNumber, value)
    if (result.ok) {
      setNote(result.data)
      // only report saved if nothing changed while the request was in flight
      setState(latest.current === value ? "saved" : "dirty")
    } else {
      setState("dirty")
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const value = e.target.value
    setContent(value)
    latest.current = value
    setState("dirty")
    window.clearTimeout(timer.current)
    // ponytail: debounce-autosave, last-write-wins; no conflict merge until
    // two people actually edit the same note at the same time
    timer.current = window.setTimeout(() => save(value), 1200)
  }

  const handleBlur = () => {
    if (state === "dirty") {
      window.clearTimeout(timer.current)
      save(latest.current)
    }
  }

  return (
    <div className="c-internal-notes">
      <div className="c-internal-notes__head">
        <span className="c-internal-notes__title">
          <svg viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
            <path
              fillRule="evenodd"
              d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z"
              clipRule="evenodd"
            />
          </svg>
          Internal notes
        </span>
        <span className="c-internal-notes__status">
          {state === "saving" && "Saving…"}
          {state === "saved" && "Saved"}
          {state === "dirty" && "Unsaved changes"}
          {state === "idle" && note?.updatedByName && `Last edited by ${note.updatedByName}`}
        </span>
      </div>
      <textarea
        className="c-internal-notes__textarea"
        value={content}
        onChange={handleChange}
        onBlur={handleBlur}
        placeholder="Notes only your team can see — context, blockers, decisions. Also shown on the linked scorecard."
        maxLength={10000}
      />
      <div className="c-internal-notes__hint">Only collaborators and administrators can see this. Synced with the scorecard.</div>
    </div>
  )
}
