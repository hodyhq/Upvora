import "./IdeaBriefButton.scss"

import React, { useEffect, useState } from "react"
import { Markdown, Modal, Button } from "@fider/components"
import { actions, Fider } from "@fider/services"

interface IdeaBriefButtonProps {
  postNumber: number
}

// Shows "View Idea Brief" when the post has one. The served content never
// contains the submitter's email — admins fetch the raw file (with the email
// substituted server-side) through the download route only.
export const IdeaBriefButton = (props: IdeaBriefButtonProps) => {
  const [content, setContent] = useState<string | null>(null)
  const [isOpen, setIsOpen] = useState(false)

  useEffect(() => {
    if (!Fider.session.isAuthenticated) {
      return
    }
    actions.getIdeaBrief(props.postNumber).then((result) => {
      if (result.ok && result.data.content) {
        setContent(result.data.content)
      }
    })
  }, [props.postNumber])

  if (!content) {
    return null
  }

  const isAdmin = Fider.session.isAuthenticated && Fider.session.user.isAdministrator

  return (
    <>
      <button type="button" className="c-brief-btn" onClick={() => setIsOpen(true)}>
        📄 View Idea Brief
      </button>
      <Modal.Window isOpen={isOpen} onClose={() => setIsOpen(false)} center={false} size="large">
        <Modal.Content>
          <div className="c-brief-doc">
            <Markdown text={content} style="full" />
          </div>
        </Modal.Content>
        <Modal.Footer>
          {isAdmin && (
            <a className="c-button c-button--default c-button--secondary" href={`/_api/posts/${props.postNumber}/brief/download`} download>
              ⬇ Download .md
            </a>
          )}
          <Button variant="tertiary" onClick={() => setIsOpen(false)}>
            Close
          </Button>
        </Modal.Footer>
      </Modal.Window>
    </>
  )
}
