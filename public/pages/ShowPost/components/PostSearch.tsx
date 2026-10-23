import "./PostSearch.scss"

import React, { useEffect, useState } from "react"
import IconSearch from "@fider/assets/images/heroicons-search.svg"
import { Input, ShowPostStatus } from "@fider/components"
import { actions } from "@fider/services"
import { Post, PostStatus, postStatusValue } from "@fider/models"
import { HStack } from "@fider/components/layout"
import { i18n } from "@lingui/core"
import { Trans } from "@lingui/react/macro"

interface PostSearchProps {
  exclude?: number[]
  onChanged(postNumber: number): void
}

export const PostSearch = (props: PostSearchProps) => {
  const [query, setQuery] = useState("")
  const [posts, setPosts] = useState<Post[]>([])
  const [selectedPost, setSelectedPost] = useState<Post>()

  useEffect(() => {
    if (!query) {
      return
    }

    const timer = setTimeout(() => {
      actions.searchPosts({ query, limit: 6 }).then((res) => {
        if (res.ok) {
          const filteredPosts =
            props.exclude && props.exclude.length > 0 ? res.data.filter((i) => props.exclude && props.exclude.indexOf(i.number) === -1) : res.data
          setPosts(filteredPosts)
        }
      })
    }, 500)

    return () => {
      clearTimeout(timer)
    }
  }, [query])

  const selectPost = (post: Post) => () => {
    props.onChanged(post.number)
    setSelectedPost(post)
  }

  return (
    <>
      <Input
        field="query"
        icon={IconSearch}
        placeholder={i18n._({ id: "showpost.postsearch.query.placeholder", message: "Search original post..." })}
        value={query}
        onChange={setQuery}
      />
      {posts.length > 0 && (
        <ul className="c-post-search__results">
          {posts.map((p) => (
            <li key={p.id}>
              <button
                type="button"
                aria-pressed={selectedPost === p}
                onClick={selectPost(p)}
                className={`c-post-search__item ${selectedPost === p ? "c-post-search__item--selected" : ""}`}
              >
                <span className="c-post-search__title">{p.title}</span>
                <HStack className="c-post-search__meta">
                  <span>#{p.number}</span> <span>&middot;</span> <ShowPostStatus status={PostStatus.Get(postStatusValue(p))} /> <span>&middot;</span>{" "}
                  <span>
                    <Trans id="showpost.postsearch.numofvotes">{p.votesCount} votes</Trans>
                  </span>
                </HStack>
              </button>
            </li>
          ))}
        </ul>
      )}
    </>
  )
}
