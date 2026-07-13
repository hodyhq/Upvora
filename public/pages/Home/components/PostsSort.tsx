import React from "react"
import { i18n } from "@lingui/core"

interface PostsSortProps {
  value: string
  onChange: (value: string) => void
}

export const PostsSort: React.FC<PostsSortProps> = ({ value = "trending", onChange }) => {
  const options = [
    { value: "trending", label: i18n._({ id: "home.postfilter.option.trending", message: "Trending" }) },
    { value: "most-wanted", label: i18n._({ id: "home.postfilter.option.mostwanted", message: "Most Wanted" }) },
    { value: "most-discussed", label: i18n._({ id: "home.postfilter.option.mostdiscussed", message: "Most Discussed" }) },
    { value: "recent", label: i18n._({ id: "home.postfilter.option.recent", message: "Recent" }) },
  ]

  return (
    <div className="c-posts-sort" role="group" aria-label={i18n._({ id: "home.postsort.label", message: "Sort by:" })}>
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          className={value === o.value ? "c-posts-sort__tab c-posts-sort__tab--active" : "c-posts-sort__tab"}
          aria-pressed={value === o.value}
          onClick={() => onChange(o.value)}
        >
          {o.label}
        </button>
      ))}
    </div>
  )
}
