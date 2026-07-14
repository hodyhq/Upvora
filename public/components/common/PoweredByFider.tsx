import React from "react"
import { classSet, FiderContext } from "@fider/services"

import "./PoweredByFider.scss"

interface PoweredByFiderProps {
  slot: string
  className?: string
}

export const PoweredByFider = (props: PoweredByFiderProps) => {
  const fider = React.useContext(FiderContext)
  // Strip only the trailing commit-hash segment (8+ hex chars) so pre-release
  // markers like fork version suffixes stay visible in the footer.
  const version = fider.settings?.version?.replace(/-[0-9a-f]{7,}$/, "")
  const versionString = fider.isSingleHostMode() && version && version !== "dev" ? `${version}` : ""

  const className = classSet({
    "c-powered": true,
    [props.className || ""]: props.className,
  })

  return (
    <div className={className}>
      <a rel="noopener" className="c-powered__brand" href="https://github.com/hodyhq/Upvora" target="_blank">
        <svg className="c-powered__mark" viewBox="0 0 512 512" aria-hidden="true">
          <path
            d="M72 88 H168 V270 C168 346 202 382 256 382 C310 382 344 346 344 270 V88 H440 V276 C440 420 365 476 256 476 C147 476 72 420 72 276 Z"
            fill="currentColor"
          />
          <path d="M48 66 H144 L256 216 L370 74 C397 42 425 20 468 12 C433 51 407 91 384 126 L258 326 Z" fill="#F97316" />
        </svg>
        <span>
          Powered by <strong>Upvora</strong>
        </span>
      </a>
      {versionString && <span className="c-powered__version">{versionString}</span>}
    </div>
  )
}
