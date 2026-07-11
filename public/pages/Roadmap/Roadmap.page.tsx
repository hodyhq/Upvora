import "./Roadmap.page.scss"
import IconArrowLeft from "@fider/assets/images/heroicons-arrowleft.svg"
import IconCheckCircle from "@fider/assets/images/heroicons-check-circle.svg"
import IconTag from "@fider/assets/images/heroicons-tagsolid.svg"

import React, { useState, useCallback } from "react"
import { Post, Status, Tag } from "@fider/models"
import { Header, Button, Icon, Markdown, Moment } from "@fider/components"
import { VStack, HStack } from "@fider/components/layout"
import { useFider, usePostOverlay } from "@fider/hooks"
import { actions, notify, Fider } from "@fider/services"
import { PostDetails } from "@fider/components/PostDetails"
import { Trans } from "@lingui/react/macro"

interface RoadmapColumnData {
  status: Status
  posts: Post[]
}

interface RoadmapPageProps {
  columns?: RoadmapColumnData[]
  tags?: Tag[]
}

interface RoadmapColumnProps {
  status: string
  statusLabel: string
  statusColor: string
  kind?: string
  posts: Post[]
  tags: Tag[]
  currentLimit: number
  collapsed?: boolean
  canDrag?: boolean
  onToggleCollapse?: () => void
  onVote?: (post: Post) => void
  onDropPost?: (postNumber: number, targetStatus: string) => void
  onShowMore: () => void
  onPostClick?: (postNumber: number, slug: string) => void
}

// Must match the Limit sent by the server-side RoadmapPage handler. The "Show
// more" link uses posts.length >= currentLimit as its heuristic, so the SSR
// payload and the client's first render must agree on the initial cap.
const ROADMAP_DEFAULT_LIMIT = 10
const ROADMAP_LIMIT_STEP = 10

const RoadmapPost = (props: {
  post: Post
  tags: Tag[]
  status: string
  kind?: string
  draggable?: boolean
  onVote?: (post: Post) => void
  onPostClick?: (postNumber: number, slug: string) => void
}) => {
  const fider = useFider()
  const isModerationEnabled = fider.session.tenant.isModerationEnabled
  const isPending = isModerationEnabled && !props.post.isApproved
  // Anything in a closed-completed kind hides the upvote affordance.
  const isCompleted = props.kind === "closed-completed" || props.status === "completed"

  const handleClick = (e: React.MouseEvent<HTMLAnchorElement>) => {
    if (props.onPostClick) {
      e.preventDefault()
      props.onPostClick(props.post.number, props.post.slug)
    }
  }

  const handleVote = (e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (props.onVote) {
      props.onVote(props.post)
    }
  }

  const handleDragStart = (e: React.DragEvent<HTMLAnchorElement>) => {
    e.dataTransfer.setData("text/plain", String(props.post.number))
    e.dataTransfer.effectAllowed = "move"
  }

  return (
    <a
      href={`/posts/${props.post.number}/${props.post.slug}`}
      className="c-roadmap-post-link"
      onClick={handleClick}
      draggable={props.draggable}
      onDragStart={props.draggable ? handleDragStart : undefined}
    >
      <VStack className="c-roadmap-post w-full" spacing={2}>
        <HStack spacing={2} align="start" className="w-full">
          <h3 className="c-roadmap-post__title text-break">{props.post.title}</h3>
          {isPending && (
            <span className="text-xs bg-yellow-100 text-yellow-800 px-2 py-1 rounded flex-shrink-0">
              <Trans id="post.pending">pending</Trans>
            </span>
          )}
        </HStack>
        {props.post.description && <Markdown className="c-roadmap-post__desc" maxLength={110} text={props.post.description} style="plainText" />}
        {props.tags.length >= 1 && (
          <HStack spacing={1} className="flex-wrap">
            {props.tags.map((tag) => (
              <span key={tag.id} className="c-roadmap-post__tag">
                <Icon sprite={IconTag} className="c-roadmap-post__tag-icon" />
                {tag.name}
              </span>
            ))}
          </HStack>
        )}
        {isCompleted && props.post.response?.respondedAt ? (
          <HStack spacing={1} className="c-roadmap-post__completed flex-items-center">
            <Icon sprite={IconCheckCircle} className="h-4 w-4 text-green-600" />
            <Moment locale={fider.currentLocale} date={props.post.response.respondedAt} />
          </HStack>
        ) : (
          <button
            type="button"
            className={`c-roadmap-post__votebtn ${props.post.hasVoted ? "c-roadmap-post__votebtn--voted" : ""}`}
            onClick={handleVote}
            title="Vote"
          >
            ▲ <span className="text-semibold">{props.post.votesCount}</span>{" "}
            {props.post.votesCount === 1 ? <Trans id="label.vote">Vote</Trans> : <Trans id="label.votes">Votes</Trans>}
          </button>
        )}
      </VStack>
    </a>
  )
}

const RoadmapColumn = (props: RoadmapColumnProps) => {
  const [isOver, setIsOver] = useState(false)
  // If we received at least as many posts as we asked for there may be more on
  // the server — same heuristic the Home feed uses (PostsContainer.getShowMoreLink).
  const hasMore = props.posts.length >= props.currentLimit

  const handleDragOver = (e: React.DragEvent) => {
    if (!props.canDrag) return
    e.preventDefault()
    e.dataTransfer.dropEffect = "move"
    setIsOver(true)
  }

  const handleDrop = (e: React.DragEvent) => {
    if (!props.canDrag) return
    e.preventDefault()
    setIsOver(false)
    const number = parseInt(e.dataTransfer.getData("text/plain"), 10)
    if (number && props.onDropPost) {
      props.onDropPost(number, props.status)
    }
  }

  return (
    <div className={`c-roadmap-column c-roadmap-column--${props.statusColor || "blue"}`}>
      <div className="c-roadmap-column__header c-roadmap-column__header--clickable" onClick={props.onToggleCollapse} title="Collapse lane">
        <span className={`c-roadmap-column__chip c-roadmap-column__chip--${props.statusColor || "blue"}`}>{props.statusLabel}</span>
        <span className="c-roadmap-column__count">
          {props.posts.length}
          {props.collapsed ? " ▸" : ""}
        </span>
      </div>
      <div
        className={`c-roadmap-column__body ${isOver ? "c-roadmap-column__body--over" : ""}`}
        style={props.collapsed ? { display: "none" } : undefined}
        onDragOver={handleDragOver}
        onDragLeave={() => setIsOver(false)}
        onDrop={handleDrop}
      >
        {props.posts.map((post) => (
          <RoadmapPost
            key={post.id}
            post={post}
            tags={props.tags.filter((tag) => post.tags.indexOf(tag.slug) >= 0)}
            status={props.status}
            kind={props.kind}
            draggable={props.canDrag}
            onVote={props.onVote}
            onPostClick={props.onPostClick}
          />
        ))}
        {hasMore && (
          <div className="my-4 text-center">
            <a
              href="#"
              className="text-primary-base text-medium hover:underline"
              onClick={(e) => {
                e.preventDefault()
                props.onShowMore()
              }}
            >
              <Trans id="roadmap.column.showmore">Show more</Trans>
            </a>
          </div>
        )}
      </div>
    </div>
  )
}

interface ColumnState {
  status: Status
  posts: Post[]
  limit: number
}

const RoadmapBoard = (props: RoadmapPageProps) => {
  const [columns, setColumns] = useState<ColumnState[]>((props.columns || []).map((c) => ({ status: c.status, posts: c.posts, limit: ROADMAP_DEFAULT_LIMIT })))
  const tags = props.tags || []
  const [query, setQuery] = useState("")
  const [tagFilter, setTagFilter] = useState<string | null>(null)
  const [collapsed, setCollapsed] = useState<{ [slug: string]: boolean }>({})
  const canDrag = Fider.session.isAuthenticated && Fider.session.user.isCollaborator

  const matchesFilters = (p: Post): boolean => {
    if (tagFilter && p.tags.indexOf(tagFilter) < 0) return false
    if (query) {
      const q = query.toLowerCase()
      if (p.title.toLowerCase().indexOf(q) < 0 && (p.description || "").toLowerCase().indexOf(q) < 0) return false
    }
    return true
  }

  const handleVote = async (post: Post) => {
    if (!Fider.session.isAuthenticated) {
      return
    }
    // optimistic flip; revert on failure
    const apply = (delta: number, voted: boolean) =>
      setColumns((cols) =>
        cols.map((c) => ({ ...c, posts: c.posts.map((p) => (p.number === post.number ? { ...p, votesCount: p.votesCount + delta, hasVoted: voted } : p)) }))
      )
    const wasVoted = post.hasVoted
    apply(wasVoted ? -1 : 1, !wasVoted)
    const result = await actions.toggleVote(post.number)
    if (!result.ok) {
      apply(wasVoted ? 1 : -1, wasVoted)
      notify.error("Could not register your vote. Please try again.")
    }
  }

  const handleDropPost = async (postNumber: number, targetStatus: string) => {
    const source = columns.find((c) => c.posts.some((p) => p.number === postNumber))
    if (!source || source.status.slug === targetStatus) return
    const post = source.posts.find((p) => p.number === postNumber) as Post
    // optimistic move; revert on failure
    const move = (from: string, to: string) =>
      setColumns((cols) =>
        cols.map((c) => {
          if (c.status.slug === from) return { ...c, posts: c.posts.filter((p) => p.number !== postNumber) }
          if (c.status.slug === to) return { ...c, posts: [post, ...c.posts] }
          return c
        })
      )
    move(source.status.slug, targetStatus)
    const result = await actions.respond(postNumber, { status: targetStatus, text: post.response?.text || "" })
    if (!result.ok) {
      move(targetStatus, source.status.slug)
      notify.error("Could not change the status. Please try again.")
    } else {
      reloadPosts()
    }
  }

  const reloadPosts = useCallback(async () => {
    const next = await Promise.all(
      columns.map(async (col) => {
        const result = await actions.searchPosts({ statuses: [col.status.slug], limit: col.limit })
        return result.ok ? { ...col, posts: result.data as Post[] } : col
      })
    )
    setColumns(next)
  }, [columns])

  const showMore = async (slug: string) => {
    const target = columns.find((c) => c.status.slug === slug)
    if (!target) return
    const nextLimit = target.limit + ROADMAP_LIMIT_STEP
    const result = await actions.searchPosts({ statuses: [slug], limit: nextLimit })
    if (!result.ok) return
    setColumns(columns.map((c) => (c.status.slug === slug ? { ...c, posts: result.data as Post[], limit: nextLimit } : c)))
  }

  const { selectedPostId, handlePostClick, handleCloseOverlay, setIsPostDirty } = usePostOverlay({
    basePath: "/roadmap",
    onPostClosed: () => reloadPosts(),
  })

  // Any column the admin opted into the roadmap (via show_on_roadmap) counts —
  // if Completed is enabled and has posts, the board is not "waiting for its
  // first update."
  const hasAnyOpenWork = columns.some((c) => c.posts.length > 0)

  if (!hasAnyOpenWork && selectedPostId === null) {
    return <RoadmapBlankState />
  }

  return (
    <div id="p-roadmap" className="page container">
      <div style={selectedPostId !== null ? { display: "none" } : undefined}>
        <div className="p-roadmap__head">
          <span className="p-roadmap__eyebrow">
            <Trans id="roadmap.head.eyebrow">Roadmap</Trans>
          </span>
          <h1 className="p-roadmap__title">
            <Trans id="roadmap.head.title">Where things stand</Trans>
          </h1>
          <p className="p-roadmap__subtitle">
            <Trans id="roadmap.head.subtitle">From triage to shipped. Every card started as a vote on the board.</Trans>
          </p>
        </div>
        <VStack spacing={4}>
          <div className="c-roadmap-toolbar">
            <input className="c-roadmap-toolbar__search" placeholder="Search the roadmap" value={query} onChange={(e) => setQuery(e.target.value)} />
            <div className="c-roadmap-toolbar__tags">
              {tags.map((t) => (
                <button
                  key={t.id}
                  type="button"
                  className={`c-tag c-roadmap-toolbar__tag ${tagFilter === t.slug ? "c-roadmap-toolbar__tag--on" : ""}`}
                  onClick={() => setTagFilter((f) => (f === t.slug ? null : t.slug))}
                >
                  {t.name}
                </button>
              ))}
            </div>
          </div>
          <div className="c-roadmap-board">
            {columns.map((col) => (
              <RoadmapColumn
                key={col.status.slug}
                status={col.status.slug}
                statusLabel={col.status.label}
                statusColor={col.status.color}
                kind={col.status.kind}
                posts={col.posts.filter(matchesFilters)}
                tags={tags}
                currentLimit={col.limit}
                collapsed={collapsed[col.status.slug]}
                canDrag={canDrag}
                onToggleCollapse={() => setCollapsed((c) => ({ ...c, [col.status.slug]: !c[col.status.slug] }))}
                onVote={handleVote}
                onDropPost={handleDropPost}
                onShowMore={() => showMore(col.status.slug)}
                onPostClick={handlePostClick}
              />
            ))}
          </div>
        </VStack>
      </div>
      {selectedPostId !== null && (
        <div>
          <Button onClick={handleCloseOverlay} variant="link">
            <HStack spacing={2}>
              <Icon sprite={IconArrowLeft} />
              <span className="text-body clickable text-blue-600 hover">
                <Trans id="postdetails.backtoroadmap">Back to roadmap</Trans>
              </span>
            </HStack>
          </Button>
          <PostDetails postNumber={selectedPostId} onDataChanged={() => setIsPostDirty(true)} />
        </div>
      )}
    </div>
  )
}

const SkeletonCard = () => (
  <div className="c-roadmap-upsell__card">
    <div className="c-roadmap-upsell__bar c-roadmap-upsell__bar--lg" />
    <div className="c-roadmap-upsell__bar c-roadmap-upsell__bar--sm" />
    <div className="c-roadmap-upsell__bar c-roadmap-upsell__bar--md" />
    <div className="c-roadmap-upsell__bar c-roadmap-upsell__bar--md" />
  </div>
)

const SkeletonColumn = ({ status }: { status: string }) => (
  <div className="c-roadmap-column">
    <div className="c-roadmap-column__header">
      <span className="c-roadmap-column__chip c-roadmap-column__chip--gray">{status}</span>
    </div>
    <div className="c-roadmap-column__body">
      <SkeletonCard />
      <SkeletonCard />
    </div>
  </div>
)

const RoadmapSkeletonBackdrop = () => (
  <div className="c-roadmap-upsell__skeleton" aria-hidden="true">
    <div className="c-roadmap-board">
      <SkeletonColumn status="planned" />
      <SkeletonColumn status="started" />
      <SkeletonColumn status="completed" />
    </div>
  </div>
)

const RoadmapUpsell = () => {
  const fider = useFider()
  const isAdmin = fider.session.isAuthenticated && fider.session.user.isAdministrator
  const showBillingCta = isAdmin && fider.settings.isBillingEnabled

  return (
    <div id="p-roadmap-upsell" className="page container">
      <RoadmapSkeletonBackdrop />
      <VStack spacing={4} className="c-roadmap-upsell flex-items-center text-center">
        <h1 className="c-roadmap-upsell__title text-display">
          <Trans id="roadmap.upsell.title">See what&apos;s happening in the Roadmap view</Trans>
        </h1>
        <p className="c-roadmap-upsell__subtitle text-muted">
          <Trans id="roadmap.upsell.description">Upgrade to Pro to unlock your Roadmap</Trans>
        </p>
        {showBillingCta && (
          <a href="/admin/billing">
            <Button variant="primary" size="large">
              <Trans id="roadmap.upsell.billing">Upgrade to PRO</Trans>
            </Button>
          </a>
        )}
      </VStack>
    </div>
  )
}

const RoadmapBlankState = () => (
  <div id="p-roadmap-blank" className="page container">
    <RoadmapSkeletonBackdrop />
    <VStack spacing={4} className="c-roadmap-upsell flex-items-center text-center">
      <h1 className="c-roadmap-upsell__title text-display">
        <Trans id="roadmap.blank.title">Your roadmap is waiting for its first update</Trans>
      </h1>
      <p className="c-roadmap-upsell__subtitle text-muted">
        <Trans id="roadmap.blank.description">Mark posts as planned or in progress and they&apos;ll show up here on the roadmap.</Trans>
      </p>
    </VStack>
  </div>
)

const RoadmapPage = (props: RoadmapPageProps) => {
  const fider = useFider()
  const hasRoadmap = fider.isSingleHostMode() || fider.session.tenant.isPro

  return (
    <>
      <Header />
      {hasRoadmap ? <RoadmapBoard {...props} /> : <RoadmapUpsell />}
    </>
  )
}

export default RoadmapPage
