import React, { useCallback, useState, useEffect } from "react"

import { Post } from "@fider/models"
import { Button, Form } from "@fider/components"
import { SignInModal } from "@fider/components"

import { cache, actions, Failure, Fider } from "@fider/services"
import { i18n } from "@lingui/core"
import { Trans } from "@lingui/react/macro"

import { useFider } from "@fider/hooks"
import { useAttachments } from "@fider/hooks/useAttachments"
import CommentEditor from "@fider/components/common/form/CommentEditor"

interface CommentInputProps {
  post: Post
}

const CACHE_TITLE_KEY = "CommentInput-Comment-Title-"
const CACHE_ATTACHMENTS_KEY = "CommentInput-Comment-Attachments-"

export const CommentInput = (props: CommentInputProps) => {
  const getCacheKey = (cachePrefix: string) => `${cachePrefix}${props.post.id}`

  const getContentFromCache = () => {
    return cache.session.get(getCacheKey(CACHE_TITLE_KEY))
  }

  const COMMENT_MAX_LENGTH = 4000

  const fider = useFider()
  const [isSignInModalOpen, setIsSignInModalOpen] = useState(false)
  const [isInternal, setIsInternal] = useState(false)
  const [error, setError] = useState<Failure | undefined>(undefined)
  const [isClient, setIsClient] = useState(false)
  const [contentLength, setContentLength] = useState((cache.session.get(`${CACHE_TITLE_KEY}${props.post.id}`) ?? "").length)

  // Use the attachments hook
  const { attachments, handleImageUploaded, getImageSrc, clearAttachments } = useAttachments({
    cacheKey: getCacheKey(CACHE_ATTACHMENTS_KEY),
    maxAttachments: 2,
  })

  // Check if we're running on the client after component mounts
  useEffect(() => {
    setIsClient(true)
  }, [])

  const hideModal = () => setIsSignInModalOpen(false)
  const clearError = () => setError(undefined)

  const submit = async () => {
    clearError()

    const content = getContentFromCache()

    const result = await actions.createComment(props.post.number, content || "", attachments, isInternal)
    if (result.ok) {
      clearAttachments()
      cache.session.remove(getCacheKey(CACHE_TITLE_KEY))
      if (fider.session.isModerationRequiredForNewPost) {
        cache.session.set("COMMENT_CREATED_MODERATION", "true")
      }
      location.reload()
    } else {
      setError(result.error)
    }
  }

  const editorFocused = () => {
    if (!fider.session.isAuthenticated) {
      setIsSignInModalOpen(true)
    }
  }

  const hasContent = true

  const commentChanged = useCallback((value: string): void => {
    cache.session.set(getCacheKey(CACHE_TITLE_KEY), value)
    setContentLength(value.length)
  }, [])

  const isOverLimit = contentLength > COMMENT_MAX_LENGTH
  const canPostInternal = Fider.session.isAuthenticated && Fider.session.user.isCollaborator

  return (
    <>
      <SignInModal isOpen={isSignInModalOpen} onClose={hideModal} />
      <div className="c-comment-input">
        <div className={isInternal ? "c-comment-input-card c-comment-input-card--internal" : "c-comment-input-card"}>
          <Form error={error}>
            {isClient ? (
              <>
                <CommentEditor
                  field="content"
                  disabled={!Fider.session.isAuthenticated}
                  onChange={commentChanged}
                  onFocus={editorFocused}
                  initialValue={getContentFromCache()}
                  placeholder={i18n._({ id: "showpost.commentinput.placeholder", message: "Leave a comment" })}
                  maxAttachments={2}
                  maxImageSizeKB={5 * 1024}
                  maxLength={COMMENT_MAX_LENGTH}
                  onGetImageSrc={getImageSrc}
                  onImageUploaded={handleImageUploaded}
                />

                {(hasContent || isInternal) && (
                  <div className="c-comment-input__footer">
                    <Button disabled={!fider.session.isAuthenticated || isOverLimit || !hasContent} variant="primary" onClick={submit}>
                      <Trans id="action.postcomment">Post</Trans>
                    </Button>
                    {canPostInternal && (
                      <button
                        type="button"
                        className="c-internal-toggle"
                        data-on={isInternal}
                        onClick={() => setIsInternal(!isInternal)}
                        aria-pressed={isInternal}
                      >
                        <span className="c-internal-toggle__dot" />
                        <Trans id="showpost.commentinput.internal">Internal — only your team sees this</Trans>
                      </button>
                    )}
                  </div>
                )}
              </>
            ) : (
              <div className="comment-input-placeholder p-2">{i18n._({ id: "showpost.commentinput.placeholder", message: "Leave a comment" })}</div>
            )}
          </Form>
        </div>
      </div>
    </>
  )
}
