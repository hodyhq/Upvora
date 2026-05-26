import React from "react"
import { i18n } from "@lingui/core"
import IconSparkles from "@fider/assets/images/heroicons-sparkles-outline.svg"
import IconThumbsUp from "@fider/assets/images/heroicons-thumbsup.svg"
import IconChat from "@fider/assets/images/heroicons-chat-alt-2.svg"
import IconClock from "@fider/assets/images/heroicons-clock.svg"
import { HStack } from "@fider/components/layout"
import { Icon } from "@fider/components"

interface PostsSortProps {
  value: string
  onChange: (value: string) => void
}

export const PostsSort: React.FC<PostsSortProps> = ({ value = "trending", onChange }) => {
  const options = [
    { value: "trending", label: i18n._({ id: "home.postfilter.option.trending", message: "Trending" }), icon: IconSparkles },
    { value: "most-wanted", label: i18n._({ id: "home.postfilter.option.mostwanted", message: "Most Wanted" }), icon: IconThumbsUp },
    { value: "most-discussed", label: i18n._({ id: "home.postfilter.option.mostdiscussed", message: "Most Discussed" }), icon: IconChat },
    { value: "recent", label: i18n._({ id: "home.postfilter.option.recent", message: "Recent" }), icon: IconClock },
  ]

  return (
    <HStack className="c-post-sort-pills" spacing={2}>
      {options.map((o) => {
        const isActive = o.value === value
        const className = `c-post-sort-pill${isActive ? " c-post-sort-pill--active" : ""}`
        return (
          <button key={o.value} type="button" className={className} onClick={() => onChange(o.value)}>
            <Icon sprite={o.icon} className="c-post-sort-pill__icon" />
            <span>{o.label}</span>
          </button>
        )
      })}
    </HStack>
  )
}
