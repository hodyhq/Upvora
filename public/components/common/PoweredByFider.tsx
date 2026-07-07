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
      <a rel="noopener" className="text-2xs" href="https://github.com/hodyhq/fiderH" target="_blank">
        Powered by FiderH ⚡
      </a>
      {versionString && <span className="text-2xs block">{versionString}</span>}
    </div>
  )
}
