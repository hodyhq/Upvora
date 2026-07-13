import React, { useState } from "react"
import { Post, PostStatus, postStatusValue } from "@fider/models"
import { actions } from "@fider/services"
import { SignInModal } from "@fider/components"
import { useFider } from "@fider/hooks"
import { Trans } from "@lingui/react/macro"

interface VoteSectionProps {
  post: Post
  votes: number
  onDataChanged?: () => void
}

export const VoteSection = (props: VoteSectionProps) => {
  const fider = useFider()
  const [votes, setVotes] = useState(props.votes)
  const [hasVoted, setHasVoted] = useState(props.post.hasVoted)
  const [isSignInModalOpen, setIsSignInModalOpen] = useState(false)

  const voteOrUndo = async () => {
    if (!fider.session.isAuthenticated) {
      setIsSignInModalOpen(true)
      return
    }

    const action = hasVoted ? actions.removeVote : actions.addVote
    const response = await action(props.post.number)
    if (response.ok) {
      setVotes(hasVoted ? votes - 1 : votes + 1)
      setHasVoted(!hasVoted)
      props.onDataChanged?.() // Notify parent that data changed
    }
  }

  const hideModal = () => setIsSignInModalOpen(false)

  const status = PostStatus.Get(postStatusValue(props.post))
  const isDisabled = status.closed || fider.isReadOnly

  return (
    <>
      <SignInModal isOpen={isSignInModalOpen} onClose={hideModal} />
      <button
        className="c-post__vote p-show-post__voteblock"
        data-voted={hasVoted ? "true" : "false"}
        onClick={voteOrUndo}
        disabled={isDisabled}
        aria-pressed={hasVoted}
      >
        <svg
          className="c-post__chev"
          viewBox="0 0 20 20"
          fill="none"
          stroke="currentColor"
          strokeWidth="2.3"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <path d="M5 12l5-5 5 5" />
        </svg>
        <span className="c-post__votes">{votes}</span>
        <span className="c-post__voteslabel">{votes === 1 ? <Trans id="label.vote">Vote</Trans> : <Trans id="label.votes">Votes</Trans>}</span>
      </button>
    </>
  )
}
