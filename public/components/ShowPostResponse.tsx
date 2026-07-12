import React from "react"
import { PostResponse, PostStatus, Status, resolveStatus } from "@fider/models"
import { Fider } from "@fider/services"
import { Markdown, UserName, Moment, Avatar } from "@fider/components"
import { HStack, VStack } from "./layout"
import { Trans } from "@lingui/react/macro"
import { useFider } from "@fider/hooks"

type Size = "micro" | "small" | "xsmall" | "normal"

interface PostResponseProps {
  status: string
  response: PostResponse | null
  size?: Size
}

export const ResponseDetails = (props: PostResponseProps): JSX.Element | null => {
  const fider = useFider()
  const status = PostStatus.Get(props.status)

  if (!props.response) {
    return null
  }

  return (
    <div className="c-response-details">
      <div className="c-response-details__card">
        <div className="c-response-details__inner">
          <VStack spacing={2}>
            <HStack spacing={2} align="center">
              <Avatar user={props.response.user} size="small" />
              <UserName user={props.response.user} />
              <span className="text-xs text-gray-600">•</span>
              <Moment className="text-xs text-gray-600" locale={fider.currentLocale} date={props.response.respondedAt} />
              <ResponseLozenge status={props.status} response={props.response} size="xsmall" />
            </HStack>

            {props.response?.text && status !== PostStatus.Duplicate && (
              <div className="c-response-details__content">
                <Markdown text={props.response.text} style="full" />
              </div>
            )}

            {status === PostStatus.Duplicate && props.response.original && (
              <div className="c-response-details__content">
                <a className="text-link" href={`/posts/${props.response.original.number}/${props.response.original.slug}`}>
                  {props.response.original.title}
                </a>
              </div>
            )}
          </VStack>
        </div>
      </div>
    </div>
  )
}

// Color-name fallbacks for the six legacy statuses when the tenant catalogue
// has not resolved (keep in sync with ListPosts builtInColors).
const builtinColorName: Record<string, string> = {
  open: "blue",
  planned: "blue",
  started: "yellow",
  completed: "green",
  declined: "red",
  duplicate: "gray",
}
const getStatusTranslation = (status: PostStatus, tenantStatus: Status | null): JSX.Element => {
  // Tenant catalogue label wins for custom slugs — i18n has no catalog entry
  // for admin-named statuses, so falling back to <Trans> would render the raw
  // message-id (e.g. "enum.poststatus.parked").
  if (tenantStatus && !["open", "planned", "started", "completed", "declined", "duplicate", "deleted"].includes(tenantStatus.slug)) {
    return <>{tenantStatus.label}</>
  }
  switch (status) {
    case PostStatus.Open:
      return <Trans id="enum.poststatus.open">Open</Trans>
    case PostStatus.Planned:
      return <Trans id="enum.poststatus.planned">Planned</Trans>
    case PostStatus.Started:
      return <Trans id="enum.poststatus.started">Started</Trans>
    case PostStatus.Completed:
      return <Trans id="enum.poststatus.completed">Completed</Trans>
    case PostStatus.Declined:
      return <Trans id="enum.poststatus.declined">Declined</Trans>
    case PostStatus.Duplicate:
      return <Trans id="enum.poststatus.duplicate">Duplicate</Trans>
    case PostStatus.Deleted:
      return <Trans id="enum.poststatus.deleted">Deleted</Trans>
    default:
      return <>{tenantStatus?.label || status.title}</>
  }
}

// Extracts the first non-empty line of the response markdown — that's where
// the Plane webhook writes the substage label (e.g. "In Beta Testing").
const extractSubstage = (text?: string): string | null => {
  if (!text) return null
  const firstLine = text
    .split("\n")
    .map((l) => l.trim())
    .find((l) => l.length > 0)
  if (!firstLine) return null
  // Strip simple markdown bold/italic markers so the bubble reads cleanly.
  const cleaned = firstLine.replace(/[*_`]/g, "").trim()
  return cleaned.length > 60 ? cleaned.slice(0, 57) + "..." : cleaned
}

export const ResponseLozenge = (props: PostResponseProps): JSX.Element | null => {
  const status = PostStatus.Get(props.status)
  const tenantStatus = resolveStatus(Fider.session.tenant, props.status)
  const colorName = tenantStatus?.color || builtinColorName[status.value] || "gray"
  const translatedStatus = getStatusTranslation(status, tenantStatus)
  const substage = props.size === "small" || props.size === "xsmall" || !props.size ? extractSubstage(props.response?.text || undefined) : null

  return (
    <span className={`c-lozenge c-lozenge--${props.size || "normal"}`} data-color={colorName}>
      <span className="c-lozenge__dot" />
      <span className="c-lozenge__label">
        {translatedStatus}
        {substage && <span className="c-lozenge__substage"> · {substage}</span>}
      </span>
    </span>
  )
}
