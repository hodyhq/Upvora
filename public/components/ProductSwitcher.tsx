import "./ProductSwitcher.scss"

import React, { useEffect, useRef, useState } from "react"
import { useFider } from "@fider/hooks"

// Header product switcher. Renders only when the tenant has products.
// Navigation is plain links - every board is server-rendered, no SPA state.
export const ProductSwitcher = () => {
  const fider = useFider()
  const [isOpen, setIsOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  const products = fider.session.tenant.products ?? []

  const pathname = typeof window !== "undefined" && window.location ? window.location.pathname : "/"
  const search = typeof window !== "undefined" && window.location ? window.location.search : ""
  const pathMatch = /^\/p\/([^/]+)/.exec(pathname)
  const queryMatch = /[?&]product=([^&]+)/.exec(search)
  const currentSlug = pathMatch ? pathMatch[1] : queryMatch ? decodeURIComponent(queryMatch[1]) : undefined
  const current = currentSlug ? products.find((p) => p.slug === currentSlug) : undefined

  // Context-aware targets: switching products keeps you on the page you're on.
  const onRoadmap = pathname.startsWith("/roadmap")
  const onScorecard = pathname === "/scorecard" || pathname.startsWith("/scorecard?")
  const allHref = onRoadmap ? "/roadmap" : onScorecard ? "/scorecard" : "/"
  const productHref = (slug: string) => (onRoadmap ? `/roadmap?product=${slug}` : onScorecard ? `/scorecard?product=${slug}` : `/p/${slug}`)

  useEffect(() => {
    const close = () => setIsOpen(false)
    document.addEventListener("click", close)
    return () => document.removeEventListener("click", close)
  }, [])

  if (products.length === 0) {
    return null
  }

  return (
    <div className="c-prodsw" ref={ref}>
      <button
        type="button"
        className="c-prodsw__btn"
        aria-haspopup="true"
        aria-expanded={isOpen}
        onClick={(e) => {
          e.stopPropagation()
          setIsOpen(!isOpen)
        }}
      >
        {current ? (
          <span className="c-prodsw__dot" style={{ background: current.color || "var(--colors-primary-base)" }} />
        ) : (
          <span className="c-prodsw__all" />
        )}
        <span className="c-prodsw__name">{current ? current.name : "All products"}</span>
        <span className="c-prodsw__caret" />
      </button>
      {isOpen && (
        <div className="c-prodsw__menu" role="menu">
          <a href={allHref} aria-current={!current}>
            <span className="c-prodsw__all" />
            All products
          </a>
          {products.map((p) => (
            <a key={p.id} href={productHref(p.slug)} aria-current={current?.id === p.id}>
              <span className="c-prodsw__dot" style={{ background: p.color || "var(--colors-primary-base)" }} />
              {p.name}
            </a>
          ))}
          {fider.session.isAuthenticated && fider.session.user.isAdministrator && (
            <>
              <hr />
              <a href="/admin/products" className="c-prodsw__manage">
                Manage products →
              </a>
            </>
          )}
        </div>
      )}
    </div>
  )
}
