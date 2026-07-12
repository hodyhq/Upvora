import React, { useState } from "react"
import { Post, Tag, CurrentUser, postStatusValue } from "@fider/models"
import { ShowTag, Markdown, Icon, ResponseLozenge, SignInModal } from "@fider/components"
import IconChatAlt2 from "@fider/assets/images/heroicons-chat-alt-2.svg"
import { HStack, VStack } from "@fider/components/layout"
import { useFider } from "@fider/hooks"
import { Trans } from "@lingui/react/macro"

interface ListPostsProps {
  posts?: Post[]
  tags: Tag[]
  emptyText: string
  minimalView?: boolean
  showStatus?: boolean
  maxVisible?: number
  onPostClick?: (postNumber: number, slug: string) => void
  onVote?: (post: Post) => void
}

const ListPostItem = (props: {
  post: Post
  user?: CurrentUser
  tags: Tag[]
  rank?: number
  showStatus?: boolean
  onPostClick?: (postNumber: number, slug: string) => void
  onVote?: (post: Post) => void
}) => {
  const fider = useFider()
  const [isSignInModalOpen, setIsSignInModalOpen] = useState(false)
  const isModerationEnabled = fider.session.tenant.isModerationEnabled
  const isPending = isModerationEnabled && !props.post.isApproved

  const handleClick = (e: React.MouseEvent<HTMLAnchorElement>) => {
    e.stopPropagation()
    if (props.onPostClick) {
      e.preventDefault()
      props.onPostClick(props.post.number, props.post.slug)
    }
  }

  const navigateToPost = () => {
    if (props.onPostClick) {
      props.onPostClick(props.post.number, props.post.slug)
    } else {
      window.location.href = `/posts/${props.post.number}/${props.post.slug}`
    }
  }

  const handleVoteClick = (e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (!fider.session.isAuthenticated) {
      setIsSignInModalOpen(true)
      return
    }
    if (props.onVote) {
      props.onVote(props.post)
    }
  }

  const status = postStatusValue(props.post)
  const builtInColors: { [key: string]: string } = { open: "blue", planned: "blue", started: "yellow", completed: "green", declined: "red", duplicate: "gray" }
  const statusColor = fider.session.tenant.statuses?.find((s) => s.slug === status)?.color || builtInColors[status] || "gray"

  return (
    <div className="c-post" data-status={status} data-color={statusColor} onClick={navigateToPost}>
      <SignInModal isOpen={isSignInModalOpen} onClose={() => setIsSignInModalOpen(false)} />
      <button className="c-post__vote" data-voted={props.post.hasVoted ? "true" : "false"} onClick={handleVoteClick} aria-pressed={props.post.hasVoted}>
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
        <span className="c-post__votes">{props.post.votesCount}</span>
        <span className="c-post__voteslabel">{props.post.votesCount === 1 ? <Trans id="label.vote">Vote</Trans> : <Trans id="label.votes">Votes</Trans>}</span>
      </button>
      <div className="c-post__body">
        <div className="c-post__titlerow">
          <h3 className="c-post__title text-break">
            <a href={`/posts/${props.post.number}/${props.post.slug}`} onClick={handleClick}>
              {props.post.title}
            </a>
          </h3>
          {isPending && (
            <span className="c-post__pending">
              <Trans id="post.pending">pending</Trans>
            </span>
          )}
        </div>
        <Markdown className="c-post__desc" maxLength={140} text={props.post.description} style="plainText" />
        <div className="c-post__meta">
          {props.showStatus !== false && status !== "open" && <ResponseLozenge status={status} response={props.post.response} size={"small"} />}
          {props.tags.map((tag) => (
            <ShowTag key={tag.id} tag={tag} />
          ))}
        </div>
      </div>
      <div className="c-post__side">
        {props.rank !== undefined && <span className="c-post__rank">#{props.rank}</span>}
        <span className="c-post__cmts">
          <Icon sprite={IconChatAlt2} className="h-4 w-4" />
          {props.post.commentsCount}
        </span>
      </div>
    </div>
  )
}

const MinimalListPostItem = (props: { post: Post; tags: Tag[]; onPostClick?: (postNumber: number, slug: string) => void }) => {
  const fider = useFider()
  const isModerationEnabled = fider.session.tenant.isModerationEnabled
  const isPending = isModerationEnabled && !props.post.isApproved

  const handleClick = (e: React.MouseEvent<HTMLAnchorElement>) => {
    if (props.onPostClick) {
      e.preventDefault()
      props.onPostClick(props.post.number, props.post.slug)
    }
  }

  return (
    <HStack spacing={4} align="start" className="c-posts-container__post-minimal">
      <HStack className="w-full" justify="between" align="start">
        <HStack spacing={2} align="start" justify="between" className="w-full">
          <a className="text-link" href={`/posts/${props.post.number}/${props.post.slug}`} onClick={handleClick}>
            {props.post.title}
          </a>
          {isPending && <span className="text-xs bg-yellow-100 text-yellow-800 px-2 py-1 rounded">pending</span>}
        </HStack>
        {postStatusValue(props.post) !== "open" ? (
          <div>
            <ResponseLozenge status={postStatusValue(props.post)} response={props.post.response} size={"micro"} />
          </div>
        ) : (
          <span className="text-gray-700 text-sm">+{props.post.votesCount}</span>
        )}
      </HStack>
    </HStack>
  )
}

export const ListPosts = (props: ListPostsProps) => {
  const { minimalView = false } = props
  const [expanded, setExpanded] = useState(false)

  if (!props.posts) {
    return null
  }

  if (props.posts.length === 0) {
    return <p className="text-center">{props.emptyText}</p>
  }

  const allPosts = props.posts
  const hasMore = props.maxVisible !== undefined && !expanded && allPosts.length > props.maxVisible
  const visiblePosts = hasMore ? allPosts.slice(0, props.maxVisible) : allPosts
  const remainingCount = allPosts.length - visiblePosts.length

  return (
    <>
      {minimalView ? (
        <VStack spacing={2}>
          {visiblePosts.map((post) => (
            <MinimalListPostItem
              key={post.id}
              post={post}
              tags={props.tags.filter((tag) => post.tags.indexOf(tag.slug) >= 0)}
              onPostClick={props.onPostClick}
            />
          ))}
        </VStack>
      ) : (
        <>
          {visiblePosts.map((post, idx) => (
            <ListPostItem
              key={post.id}
              post={post}
              rank={idx + 1}
              tags={props.tags.filter((tag) => post.tags.indexOf(tag.slug) >= 0)}
              showStatus={props.showStatus}
              onPostClick={props.onPostClick}
              onVote={props.onVote}
            />
          ))}
        </>
      )}
      {hasMore && (
        <div className="my-4 text-center">
          <a
            href="#"
            className="text-primary-base text-medium hover:underline"
            onClick={(e) => {
              e.preventDefault()
              setExpanded(true)
            }}
          >
            <Trans id="listposts.label.showmore">Show {remainingCount} more</Trans>
          </a>
        </div>
      )}
    </>
  )
}
