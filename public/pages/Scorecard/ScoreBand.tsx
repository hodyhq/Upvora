import React from "react"
import { Fider } from "@fider/services"
import type { ScorecardField } from "@fider/models"

// Compute the weighted score from a values-JSON blob + the tenant's scorecard
// field catalogue. Mirrors app/services/sqlstore/postgres/scorecard.go's Go
// side; both use round(sum(value * weight) / 5) across active scoring fields.
export const computeWeightedScore = (values: Record<string, unknown> | undefined, fields: ScorecardField[] | undefined): number => {
  if (!fields) return 0
  const v = values ?? {}
  let total = 0
  for (const f of fields) {
    if (f.type !== "score" || !f.isActive) continue
    const w = f.weight ?? 0
    const raw = v[f.key]
    const n = typeof raw === "number" ? raw : parseInt(String(raw ?? "0"), 10) || 0
    total += n * w
  }
  return Math.round(total / 5)
}

export interface BandInfo {
  label: string
  // key indexes the c-scorecard__chip--<key> / gauge segment classes in Scorecard.scss
  key: "strong" | "good" | "refine" | "low" | "reject"
  // threshold that put the score in this band (for the "band threshold ≥ N" hint)
  threshold: number
  bg: string
  fg: string
  border: string
}

// Palette works in light + dark mode because we use solid tinted backgrounds
// with a matching darker text — no reliance on Fider's Tailwind-lite utilities
// which don't ship orange.
export const bandForScore = (score: number): BandInfo => {
  const t = Fider.session.tenant
  if (score >= t.scorecardBandStrong)
    return { label: "Strong Candidate", key: "strong", threshold: t.scorecardBandStrong, bg: "#DCFCE7", fg: "#14532D", border: "#16A34A" }
  if (score >= t.scorecardBandGood)
    return { label: "Good Candidate", key: "good", threshold: t.scorecardBandGood, bg: "#DBEAFE", fg: "#1E3A8A", border: "#2563EB" }
  if (score >= t.scorecardBandRefine)
    return { label: "Needs Refinement", key: "refine", threshold: t.scorecardBandRefine, bg: "#FEF3C7", fg: "#78350F", border: "#F59E0B" }
  if (score >= t.scorecardBandLow) return { label: "Low Priority", key: "low", threshold: t.scorecardBandLow, bg: "#FFEDD5", fg: "#7C2D12", border: "#F97316" }
  return { label: "Not Recommended", key: "reject", threshold: 0, bg: "#FEE2E2", fg: "#7F1D1D", border: "#DC2626" }
}

// Small inline pill for use in tables and list rows.
export const ScoreBandPill = (props: { score: number }) => {
  const b = bandForScore(props.score)
  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 6,
        padding: "3px 10px",
        borderRadius: 999,
        background: b.bg,
        color: b.fg,
        border: `1px solid ${b.border}`,
        fontSize: 12,
        fontWeight: 600,
        whiteSpace: "nowrap",
      }}
    >
      <strong>{props.score}</strong>
      <span style={{ fontWeight: 500 }}>{b.label}</span>
    </span>
  )
}

// Large hero-sized band panel for the card view page.
export const ScoreBandHero = (props: { score: number }) => {
  const b = bandForScore(props.score)
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        gap: 16,
        padding: "16px 20px",
        borderRadius: 12,
        background: b.bg,
        color: b.fg,
        border: `1px solid ${b.border}`,
      }}
    >
      <div>
        <div style={{ fontSize: 12, textTransform: "uppercase", letterSpacing: "0.05em", opacity: 0.75 }}>Weighted score</div>
        <div style={{ fontSize: 20, fontWeight: 700, marginTop: 2 }}>{b.label}</div>
      </div>
      <div style={{ fontSize: 40, fontWeight: 800, lineHeight: 1 }}>{props.score}</div>
    </div>
  )
}
