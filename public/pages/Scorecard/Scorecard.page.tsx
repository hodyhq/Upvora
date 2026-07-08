import "./Scorecard.scss"

import React, { useMemo, useState } from "react"
import { Button, Header } from "@fider/components"
import { ScorecardFieldChoice } from "@fider/models"
import { actions, Fider, notify } from "@fider/services"
import { computeWeightedScore, bandForScore } from "./ScoreBand"

interface ScorecardRecord {
  id: number
  postId?: number
  title: string
  values: Record<string, unknown>
  createdAt: string
  updatedAt: string
  postNumber?: number
  postSlug?: string
  postVotes?: number
  submittedBy?: string
}

interface ScorecardPageProps {
  scorecards?: ScorecardRecord[]
}

type Bucket = "new" | "review" | "executive"
const TABS: { key: "all" | Bucket; label: string }[] = [
  { key: "all", label: "All" },
  { key: "new", label: "New" },
  { key: "review", label: "In review" },
  { key: "executive", label: "Executive" },
]

const parseChoices = (raw: unknown): ScorecardFieldChoice[] => {
  if (!raw || !Array.isArray(raw)) return []
  return raw
    .map((c: any) => (typeof c === "string" ? { value: c } : { value: String(c?.value ?? ""), color: c?.color, bucket: c?.bucket }))
    .filter((c) => c.value !== "")
}

const Scorecard: React.FC<ScorecardPageProps> = (props) => {
  const cards = props.scorecards ?? []
  const featureOff = !Fider.session.tenant.isScorecardEnabled
  const [tab, setTab] = useState<"all" | Bucket>("all")
  const [filter, setFilter] = useState("")

  const fields = Fider.session.tenant.scorecardFields ?? []
  const statusField = fields.find((f) => f.groupKey === "header" && f.type === "choice" && f.isActive)
  const bucketByStatus = useMemo(() => {
    const map = new Map<string, Bucket | undefined>()
    for (const c of parseChoices(statusField?.choices)) {
      map.set(c.value, c.bucket)
    }
    return map
  }, [statusField])

  const statusOf = (c: ScorecardRecord): string => String(c.values?.[statusField?.key ?? "status"] ?? "")
  const bucketOf = (c: ScorecardRecord): Bucket | undefined => bucketByStatus.get(statusOf(c))

  const counts: Record<string, number> = { all: cards.length, new: 0, review: 0, executive: 0 }
  for (const c of cards) {
    const b = bucketOf(c)
    if (b) counts[b]++
  }

  const visible = cards.filter((c) => {
    if (tab !== "all" && bucketOf(c) !== tab) return false
    if (filter !== "") {
      const q = filter.toLowerCase()
      const hay = `${c.title} ${statusOf(c)} ${c.submittedBy ?? ""}`.toLowerCase()
      if (!hay.includes(q)) return false
    }
    return true
  })

  const newScorecard = async () => {
    const r = await actions.createScorecard({})
    if (r.ok && r.data) {
      window.location.href = `/scorecard/${r.data.id}`
    } else {
      notify.error("Could not create scorecard.")
    }
  }

  return (
    <>
      <Header />
      <div id="p-scorecard" className="page container">
        {featureOff && (
          <div className="p-3 rounded bg-yellow-100 text-yellow-800 mt-4">
            Scorecard feature is <strong>disabled</strong> at the tenant level. An administrator can turn it on at{" "}
            <a href="/admin/scorecard-settings" className="text-link">
              Site Settings → Scorecard
            </a>
            .
          </div>
        )}

        <div className="c-scorecard__panel mt-4 mb-8">
          <div className="c-scorecard__toolbar">
            <div className="c-scorecard__tabs" role="tablist">
              {TABS.map((t) => (
                <button
                  key={t.key}
                  type="button"
                  role="tab"
                  aria-selected={tab === t.key}
                  className={`c-scorecard__tab ${tab === t.key ? "c-scorecard__tab--active" : ""}`}
                  onClick={() => setTab(t.key)}
                >
                  {t.label} <span className="c-scorecard__tab-count">{counts[t.key]}</span>
                </button>
              ))}
            </div>
            <div className="c-scorecard__toolbar-right">
              <input
                type="search"
                className="c-scorecard__search"
                placeholder="Filter cards…"
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                aria-label="Filter cards"
              />
              <Button variant="primary" onClick={newScorecard}>
                + New scorecard
              </Button>
            </div>
          </div>

          <div className="c-scorecard__table-wrap">
            <table className="c-scorecard__table">
              <thead>
                <tr>
                  <th>Idea</th>
                  <th>Status</th>
                  <th>Band</th>
                  <th className="is-right">Score</th>
                  <th>Submitted by</th>
                  <th className="is-right">Votes</th>
                </tr>
              </thead>
              <tbody>
                {visible.map((c) => {
                  const score = computeWeightedScore(c.values, fields)
                  const band = bandForScore(score)
                  const status = statusOf(c)
                  const scored = fields.some((f) => f.type === "score" && f.isActive && String(c.values?.[f.key] ?? "0") !== "0")
                  return (
                    <tr key={c.id} onClick={() => (window.location.href = `/scorecard/${c.id}`)}>
                      <td>
                        <div className="c-scorecard__row-title">{c.title || `Scorecard #${c.id}`}</div>
                        {c.postNumber != null && <div className="c-scorecard__row-sub">Idea #{c.postNumber}</div>}
                      </td>
                      <td>
                        {status ? (
                          <span className="c-scorecard__chip c-scorecard__chip--status c-scorecard__chip--plain">{status}</span>
                        ) : (
                          <span className="text-muted">—</span>
                        )}
                      </td>
                      <td>
                        {scored ? (
                          <span className={`c-scorecard__chip c-scorecard__chip--${band.key}`}>{band.label}</span>
                        ) : (
                          <span className="c-scorecard__chip c-scorecard__chip--neutral c-scorecard__chip--plain">Not scored</span>
                        )}
                      </td>
                      <td className="is-right">
                        {scored ? (
                          <>
                            <span className="c-scorecard__score-cell">
                              <span className="c-scorecard__score-cell-num">{score}</span>
                              <span className="c-scorecard__score-cell-of">/100</span>
                            </span>
                            <span className="c-scorecard__score-mini">
                              <i className={`is-${band.key}`} style={{ width: `${Math.min(100, score)}%` }} />
                            </span>
                          </>
                        ) : (
                          <span className="text-muted">—</span>
                        )}
                      </td>
                      <td>{c.submittedBy || "—"}</td>
                      <td className="is-right">{c.postNumber != null ? c.postVotes ?? 0 : "—"}</td>
                    </tr>
                  )
                })}
                {visible.length === 0 && (
                  <tr>
                    <td colSpan={6} className="c-scorecard__empty" style={{ cursor: "default" }}>
                      {cards.length === 0
                        ? "No scorecards yet. Cards get created automatically when a post's status changes to your configured trigger, or manually with the button above."
                        : "No cards match this view."}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </>
  )
}

export default Scorecard
