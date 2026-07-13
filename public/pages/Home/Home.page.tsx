import "./Home.page.scss"
import NoDataIllustration from "@fider/assets/images/undraw-no-data.svg"
import IconArrowLeft from "@fider/assets/images/heroicons-arrowleft.svg"

import React, { useEffect, useState, useRef } from "react"
import { Post, Tag, PostStatus, Product } from "@fider/models"
import { Markdown, Hint, PoweredByFider, Icon, Header, Button, ShowTag } from "@fider/components"
import { PostsContainer } from "./components/PostsContainer"
import { useFider, usePostOverlay } from "@fider/hooks"
import { HStack } from "@fider/components/layout"
import { ShareFeedback } from "./components/ShareFeedback"
import { i18n } from "@lingui/core"
import { Trans } from "@lingui/react/macro"
import { isPostPending, setPostPending } from "./components/PostCache"
import { PostDetails } from "@fider/components/PostDetails"

export interface HomePageProps {
  posts: Post[]
  tags: Tag[]
  searchNoiseWords: string[]
  countPerStatus: { [key: string]: number }
  product?: Product
  countPerProduct?: { [id: string]: number }
}

export interface HomePageState {
  title: string
}

const Lonely = () => {
  const fider = useFider()

  return (
    <div className="text-center">
      <Hint permanentCloseKey="at-least-3-posts" condition={fider.session.isAuthenticated && fider.session.user.isAdministrator}>
        <p>
          <Trans id="home.lonely.suggestion">
            It&apos;s recommended that you create <strong>at least 3</strong> suggestions here before sharing this site. The initial content is important to
            start engaging your audience.
          </Trans>
        </p>
      </Hint>
      <Icon sprite={NoDataIllustration} height="120" className="mt-6 mb-2" />
      <p className="text-muted">
        <Trans id="home.lonely.text">No posts have been created yet.</Trans>
      </p>
    </div>
  )
}

const HomePage = (props: HomePageProps) => {
  const fider = useFider()
  const postsContainerRef = useRef<PostsContainer>(null)
  const [isShareFeedbackOpen, setIsShareFeedbackOpen] = useState(isPostPending())

  const { selectedPostId, handlePostClick, handleCloseOverlay, setIsPostDirty } = usePostOverlay({
    basePath: "/",
    onPostClosed: (postNumber) => postsContainerRef.current?.updateSinglePost(postNumber),
  })

  useEffect(() => {
    // If we're showing the share feedback, make sure we clear the show pending flag (for draft posts)
    if (isShareFeedbackOpen) {
      if (isPostPending()) {
        setPostPending(false)
      }
    }
  })

  const defaultWelcomeMessage = i18n._({
    id: "home.form.defaultwelcomemessage",
    message: `We'd love to hear what you're thinking about.

What can we do better? This is the place for you to vote, discuss and share ideas.`,
  })

  const defaultInvitation = i18n._({ id: "home.form.defaultinvitation", message: "Enter your suggestion here..." })
  const defaultButtonLabel = i18n._({ id: "home.button.defaultlabel", message: "Share an Idea" })

  const isLonely = () => {
    const len = Object.keys(props.countPerStatus).length
    if (len === 0) {
      return true
    }

    if (len === 1 && PostStatus.Deleted.value in props.countPerStatus) {
      return true
    }

    return false
  }

  const handleNewPost = () => {
    setIsShareFeedbackOpen(true)
  }

  // Rail "By status" breakdown — tenant status catalogue when present,
  // built-in statuses otherwise; counts come from the server payload.
  const statusRows = (
    fider.session.tenant.statuses && fider.session.tenant.statuses.length > 0
      ? fider.session.tenant.statuses
          .filter((s) => s.isActive && s.showOnHome)
          .sort((a, b) => a.sortOrder - b.sortOrder)
          .map((s) => ({ slug: s.slug, label: s.label, color: s.color || "gray" }))
      : PostStatus.All.filter((p) => p.filterable).map((p) => ({ slug: p.value, label: p.title, color: "gray" }))
  ).map((s) => ({ ...s, count: props.countPerStatus[s.slug] || 0 }))
  const maxStatusCount = Math.max(1, ...statusRows.map((r) => r.count))

  const parseWelcomeHeader = (text: string): JSX.Element[] => {
    const parts: JSX.Element[] = []
    let currentIndex = 0
    const regex = /_([^_]+)_/g
    let match: RegExpExecArray | null

    while ((match = regex.exec(text)) !== null) {
      // Add text before the match
      if (match.index > currentIndex) {
        parts.push(<span key={currentIndex}>{text.slice(currentIndex, match.index)}</span>)
      }
      // Add the highlighted text
      parts.push(
        <span key={match.index} className="header-emphasis">
          {match[1]}
        </span>
      )
      currentIndex = regex.lastIndex
    }

    // Add remaining text
    if (currentIndex < text.length) {
      parts.push(<span key={currentIndex}>{text.slice(currentIndex)}</span>)
    }

    return parts
  }

  return (
    <>
      <ShareFeedback
        tags={props.tags}
        placeholder={fider.session.tenant.invitation || defaultInvitation}
        isOpen={isShareFeedbackOpen && !fider.isReadOnly}
        onClose={() => setIsShareFeedbackOpen(false)}
      />
      <div>
        <Header hasInert={isShareFeedbackOpen && !fider.isReadOnly} />
        <div
          id="p-home"
          className="page container"
          style={selectedPostId !== null ? { display: "none" } : undefined}
          {...(isShareFeedbackOpen && !fider.isReadOnly && { inert: "true" })}
        >
          <div className="p-home__head">
            <span className="p-home__eyebrow">{props.product ? props.product.name : <Trans id="home.head.eyebrow">Feedback</Trans>}</span>
            <h1 className="p-home__welcome-title mb-3">
              {props.product ? (
                <>What should we build next for {props.product.name}?</>
              ) : fider.session.tenant.welcomeHeader ? (
                parseWelcomeHeader(fider.session.tenant.welcomeHeader)
              ) : (
                <Trans id="home.head.title">What should we build next?</Trans>
              )}
            </h1>
            <Markdown className="p-home__welcome-body" text={fider.session.tenant.welcomeMessage || defaultWelcomeMessage} style="full" />
          </div>
          <div className="p-home__main">
            {isLonely() ? (
              <Lonely />
            ) : (
              <PostsContainer
                ref={postsContainerRef}
                posts={props.posts}
                tags={props.tags}
                countPerStatus={props.countPerStatus}
                onPostClick={handlePostClick}
                product={props.product}
              />
            )}
          </div>
          <aside className="p-home__rail">
            <div className="p-home__panel p-home__panel--cta">
              <h4 className="p-home__panel-title">{fider.session.tenant.railCtaHeading || "Have an idea?"}</h4>
              <p className="p-home__panel-text">{fider.session.tenant.railCtaText || "Post a suggestion and let the community vote it up."}</p>
              <button className="c-button c-button--primary p-home__cta-btn" onClick={handleNewPost}>
                {fider.session.tenant.railCtaButton || defaultButtonLabel}
              </button>
            </div>
            {statusRows.length > 0 && (
              <div className="p-home__panel">
                <h4 className="p-home__panel-title">By status</h4>
                <div className="p-home__statusbrk">
                  {statusRows.map((r) => (
                    <div key={r.slug} className={`p-home__statusrow p-home__statusrow--${r.color}`}>
                      <span className="p-home__statuschip">
                        <span className="p-home__statusdot" />
                        {r.label}
                      </span>
                      <span className="p-home__statusbar">
                        <i style={{ width: `${(r.count / maxStatusCount) * 100}%` }} />
                      </span>
                      <span className="p-home__statusnum">{r.count}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
            {!props.product && (fider.session.tenant.products?.length ?? 0) > 0 && props.countPerProduct && (
              <div className="p-home__panel">
                <h4 className="p-home__panel-title">By product</h4>
                <div className="p-home__statusbrk">
                  {(fider.session.tenant.products ?? []).map((pr) => {
                    const count = props.countPerProduct?.[String(pr.id)] ?? 0
                    const max = Math.max(1, ...Object.values(props.countPerProduct ?? {}))
                    return (
                      <a
                        key={pr.id}
                        href={`/p/${pr.slug}`}
                        className="p-home__statusrow"
                        style={{ "--st": pr.color || "var(--colors-primary-base)" } as React.CSSProperties}
                      >
                        <span className="p-home__statuschip">
                          <span className="p-home__statusdot" />
                          {pr.name}
                        </span>
                        <span className="p-home__statusbar">
                          <i style={{ width: `${(count / max) * 100}%` }} />
                        </span>
                        <span className="p-home__statusnum">{count}</span>
                      </a>
                    )
                  })}
                </div>
              </div>
            )}
            {props.tags.length > 0 && (
              <div className="p-home__panel">
                <h4 className="p-home__panel-title">Popular tags</h4>
                <div className="p-home__tags">
                  {props.tags.slice(0, 10).map((tag) => (
                    <ShowTag key={tag.id} tag={tag} />
                  ))}
                </div>
              </div>
            )}
            <PoweredByFider slot="home-rail" className="mt-3" />
          </aside>
        </div>
        {selectedPostId !== null && (
          <div className="page container">
            <Button onClick={handleCloseOverlay} variant="link">
              <HStack spacing={2}>
                <Icon sprite={IconArrowLeft} />
                <span className="text-body clickable text-blue-600 hover">
                  <Trans id="postdetails.backtoall">Back to all suggestions</Trans>
                </span>
              </HStack>
            </Button>
            <PostDetails postNumber={selectedPostId} onDataChanged={() => setIsPostDirty(true)} />
          </div>
        )}
      </div>
    </>
  )
}

export default HomePage
