import "./Scorecard.scss"

import IconTag from "@fider/assets/images/heroicons-tagsolid.svg"

import React, { useEffect, useMemo, useRef, useState } from "react"
import { Button, Header, Icon } from "@fider/components"
import { ScorecardField, ScorecardFieldChoice, Post, User } from "@fider/models"
import { actions, notify, Fider } from "@fider/services"
import { computeWeightedScore, bandForScore } from "./ScoreBand"

interface Scorecard {
  id: number
  postId?: number
  title: string
  values: Record<string, unknown>
  createdAt: string
  updatedAt: string
  post?: Post & { user?: User }
}

interface ScorecardCardPageProps {
  scorecard: Scorecard
  assigneeNames?: string[]
}

const GROUP_ORDER = ["intake", "context", "workflow", "ownership", "classification", "scoring", "decision"] as const
const GROUP_LABELS: Record<string, string> = {
  intake: "Intake",
  context: "Context",
  workflow: "Workflow",
  ownership: "Ownership",
  classification: "Classification",
  scoring: "Scoring",
  decision: "Decision",
}
const GROUP_HINTS: Record<string, string> = {
  intake: "Additional intake questions",
  context: "Filled in during review",
  workflow: "Current state and the proposed AI-assisted future state",
  ownership: "Who owns this if it moves forward",
  classification: "What kind of effort this is",
  scoring: "1–5 each · 0 = not scored · weights sum to 100",
  decision: "Committee recommendation and outcome",
}

const parseChoices = (raw: unknown): ScorecardFieldChoice[] => {
  if (!raw || !Array.isArray(raw)) return []
  return raw
    .map((c: any) => (typeof c === "string" ? { value: c } : { value: String(c?.value ?? ""), color: c?.color, bucket: c?.bucket }))
    .filter((c) => c.value !== "")
}

const formatDate = (iso?: string): string => (iso ? new Date(iso).toLocaleDateString() : "—")

// answered = has a value that isn't blank or score-zero; retired questions with
// answers on THIS card keep rendering so committee history stays visible.
const isAnswered = (v: unknown): boolean => v != null && String(v) !== "" && String(v) !== "0"

type SaveState = "idle" | "dirty" | "saving" | "saved" | "error"

const SAVE_DEBOUNCE_MS = 1000
const SAVE_RETRY_MS = 4000

const ScorecardCard: React.FC<ScorecardCardPageProps> = (props) => {
  const allFields = Fider.session.tenant.scorecardFields ?? []
  const [title, setTitle] = useState(props.scorecard.title)
  const [values, setValues] = useState<Record<string, unknown>>(props.scorecard.values ?? {})
  const [saveState, setSaveState] = useState<SaveState>("idle")

  // Autosave plumbing. State drives the render; refs hold the latest payload
  // and in-flight bookkeeping so the debounced save never captures stale data.
  const latest = useRef({ title: props.scorecard.title, values: props.scorecard.values ?? {} })
  const timer = useRef<number | undefined>(undefined)
  const inflight = useRef(false)
  const queued = useRef(false)

  const doSave = async () => {
    if (inflight.current) {
      queued.current = true
      return
    }
    inflight.current = true
    setSaveState("saving")
    const result = await actions.updateScorecard(props.scorecard.id, { title: latest.current.title, values: latest.current.values })
    inflight.current = false
    if (result.ok) {
      if (queued.current) {
        // More edits landed while this save was in flight — save again.
        queued.current = false
        void doSave()
      } else {
        setSaveState("saved")
      }
    } else {
      setSaveState("error")
      window.clearTimeout(timer.current)
      timer.current = window.setTimeout(() => void doSave(), SAVE_RETRY_MS)
    }
  }

  const scheduleSave = () => {
    setSaveState("dirty")
    window.clearTimeout(timer.current)
    timer.current = window.setTimeout(() => void doSave(), SAVE_DEBOUNCE_MS)
  }

  // Read save state through a ref so the beforeunload effect registers ONCE.
  // Depending on saveState here would re-run the effect on every state flip,
  // and its cleanup would clearTimeout the just-scheduled save — which made
  // single discrete edits (like the status dropdown) save only every other
  // time. The timer is cleared on real unmount only.
  const saveStateRef = useRef<SaveState>("idle")
  saveStateRef.current = saveState

  useEffect(() => {
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      const s = saveStateRef.current
      if (s === "dirty" || s === "saving" || s === "error") {
        e.preventDefault()
        e.returnValue = ""
      }
    }
    window.addEventListener("beforeunload", onBeforeUnload)
    return () => {
      window.removeEventListener("beforeunload", onBeforeUnload)
      window.clearTimeout(timer.current)
    }
  }, [])

  // Active fields always show. Inactive ("retired") fields show only where this
  // card already answered them — read-only history, hidden on new cards.
  const fields = allFields.filter((f) => f.isActive || isAnswered((props.scorecard.values ?? {})[f.key]))
  const scoringFields = allFields.filter((f) => f.isActive) // weighted score uses active only

  const setValue = (key: string, v: unknown) => {
    latest.current = { ...latest.current, values: { ...latest.current.values, [key]: v } }
    setValues((prev) => ({ ...prev, [key]: v }))
    scheduleSave()
  }

  const changeTitle = (t: string) => {
    latest.current = { ...latest.current, title: t }
    setTitle(t)
    scheduleSave()
  }

  const weightedScore = useMemo(() => computeWeightedScore(values, scoringFields), [values, scoringFields])
  const band = bandForScore(weightedScore)

  const grouped: Record<string, ScorecardField[]> = {}
  for (const f of fields) {
    ;(grouped[f.groupKey] ??= []).push(f)
  }
  const headerFields = (grouped["header"] ?? []).filter((f) => f.type === "choice")

  const post = props.scorecard.post

  const saveStateText: Record<SaveState, string> = {
    idle: "",
    dirty: "Unsaved changes…",
    saving: "Saving…",
    saved: "All changes saved",
    error: "Not saved — retrying…",
  }

  const renderStatusControl = (f: ScorecardField) => {
    const strVal = String(values[f.key] ?? "")
    const choices = parseChoices(f.choices)
    const hasLegacyValue = strVal !== "" && !choices.some((c) => c.value === strVal)
    return (
      <div key={f.key} className="c-scorecard__status">
        <span className="c-scorecard__status-label">{f.label}</span>
        <select className="c-scorecard__status-select" value={strVal} onChange={(e) => setValue(f.key, e.target.value)} aria-label={f.label}>
          <option value="">—</option>
          {choices.map((c) => (
            <option key={c.value} value={c.value}>
              {c.value}
            </option>
          ))}
          {hasLegacyValue && <option value={strVal}>{strVal} (retired)</option>}
        </select>
      </div>
    )
  }

  const renderField = (f: ScorecardField) => {
    const key = f.key
    const val = values[key]
    const strVal = val == null ? "" : String(val)
    const retired = !f.isActive

    const label = (
      <label htmlFor={`sc-${key}`}>
        {f.label}
        {retired && <span className="c-scorecard__field-retired">retired</span>}
      </label>
    )

    if (f.type === "score") {
      const n = typeof val === "number" ? val : parseInt(strVal, 10) || 0
      return (
        <div key={key} className="c-scorecard__score-row">
          <div>
            <div className="c-scorecard__score-head">
              <span className="c-scorecard__score-label">
                {f.label}
                {retired && <span className="c-scorecard__field-retired">retired</span>}
              </span>
              <span className="c-scorecard__score-weight">×{f.weight ?? 0}</span>
            </div>
            {f.question && <div className="c-scorecard__score-q">{f.question}</div>}
          </div>
          <div className="c-scorecard__score-slider">
            <input
              type="range"
              min={0}
              max={5}
              step={1}
              value={n}
              disabled={retired}
              onChange={(e) => setValue(key, parseInt(e.target.value, 10))}
              aria-label={`${f.label} score`}
            />
            <div className="c-scorecard__score-steps">
              <span>0</span>
              <span>1</span>
              <span>2</span>
              <span>3</span>
              <span>4</span>
              <span>5</span>
            </div>
          </div>
          <span className={`c-scorecard__score-val ${n === 0 ? "c-scorecard__score-val--unset" : ""}`}>{n === 0 ? "—" : n}</span>
        </div>
      )
    }

    if (f.type === "choice") {
      const choices = parseChoices(f.choices)
      const hasLegacyValue = strVal !== "" && !choices.some((c) => c.value === strVal)
      return (
        <div key={key} className="c-scorecard__field">
          {label}
          <select id={`sc-${key}`} value={strVal} disabled={retired} onChange={(e) => setValue(key, e.target.value)}>
            <option value="">—</option>
            {choices.map((c) => (
              <option key={c.value} value={c.value}>
                {c.value}
              </option>
            ))}
            {hasLegacyValue && <option value={strVal}>{strVal} (retired)</option>}
          </select>
        </div>
      )
    }

    if (f.type === "note") {
      return (
        <div key={key} className="c-scorecard__field c-scorecard__field--wide">
          {label}
          <textarea id={`sc-${key}`} rows={3} value={strVal} disabled={retired} onChange={(e) => setValue(key, e.target.value)} />
        </div>
      )
    }

    if (f.type === "multiline") {
      // Hybrid: text-field width (half the row, so two sit side by side) but a
      // vertically expandable textarea like note. Starts input-height.
      return (
        <div key={key} className="c-scorecard__field c-scorecard__field--multiline">
          {label}
          <textarea id={`sc-${key}`} rows={1} value={strVal} disabled={retired} onChange={(e) => setValue(key, e.target.value)} />
        </div>
      )
    }

    if (f.type === "date") {
      return (
        <div key={key} className="c-scorecard__field">
          {label}
          <input id={`sc-${key}`} type="date" value={strVal} disabled={retired} onChange={(e) => setValue(key, e.target.value)} />
        </div>
      )
    }

    if (f.type === "user") {
      const listId = `scorecard-users-${key}`
      const names = props.assigneeNames ?? []
      return (
        <div key={key} className="c-scorecard__field">
          {label}
          <input
            id={`sc-${key}`}
            type="text"
            list={listId}
            value={strVal}
            disabled={retired}
            onChange={(e) => setValue(key, e.target.value)}
            placeholder="Pick a collaborator/admin or type any name…"
          />
          <datalist id={listId}>
            {names.map((n) => (
              <option key={n} value={n} />
            ))}
          </datalist>
        </div>
      )
    }

    // text, number, url
    const inputType = f.type === "number" ? "number" : f.type === "url" ? "url" : "text"
    return (
      <div key={key} className="c-scorecard__field">
        {label}
        <input id={`sc-${key}`} type={inputType} value={strVal} disabled={retired} onChange={(e) => setValue(key, e.target.value)} />
      </div>
    )
  }

  return (
    <>
      <Header />
      <div id="p-scorecard-card" className="page container">
        <div className="c-scorecard__panel mt-4 mb-8">
          <div className="c-scorecard__head">
            <div className="c-scorecard__head-top">
              <a href="/scorecard" className="c-scorecard__crumb">
                ← All scorecards
              </a>
              <div className="c-scorecard__head-actions">
                <span className={`c-scorecard__save-state ${saveState === "error" ? "c-scorecard__save-state--error" : ""}`}>{saveStateText[saveState]}</span>
                {post && (
                  <a className="c-button c-button--default c-button--primary" href={`/posts/${post.number}/${post.slug}`} target="_blank" rel="noopener">
                    Open Idea ↗
                  </a>
                )}
              </div>
            </div>
            <div className="c-scorecard__title-row">
              <input className="c-scorecard__title-input" value={title} onChange={(e) => changeTitle(e.target.value)} aria-label="Scorecard title" />
              {headerFields.map(renderStatusControl)}
            </div>
            {post ? (
              // Everything in this block renders live from the linked post —
              // edits on the idea page show here on the next load, never a copy.
              <div className="c-scorecard__idea">
                <div className="c-scorecard__idea-title">
                  <span className="c-scorecard__idea-label">Idea</span>
                  <strong>{post.title}</strong>
                </div>
                {post.description && <div className="c-scorecard__idea-desc">{post.description}</div>}
                <div className="c-scorecard__meta">
                  <span>
                    Submitted by <strong>{post.user?.name ?? "—"}</strong> on {formatDate(post.createdAt)}
                  </span>
                  <span>To scorecard {formatDate(props.scorecard.createdAt)}</span>
                  <span>
                    <strong>{(post as any).votesCount ?? 0}</strong> votes · <strong>{(post as any).commentsCount ?? 0}</strong> comments
                  </span>
                  {Array.isArray(post.tags) && post.tags.length > 0 && (
                    <span className="c-scorecard__meta-tags">
                      {post.tags.map((t: any) => (
                        <span key={String(t)} className="c-scorecard__chip c-scorecard__chip--neutral c-scorecard__chip--plain">
                          <Icon sprite={IconTag} className="c-scorecard__tag-icon" />
                          {String(t)}
                        </span>
                      ))}
                    </span>
                  )}
                </div>
              </div>
            ) : (
              <div className="c-scorecard__meta">
                <span>Standalone scorecard (not linked to an idea) · created {formatDate(props.scorecard.createdAt)}</span>
              </div>
            )}
          </div>

          <div className="c-scorecard__gauge c-scorecard__gauge--ring">
            {(() => {
              const p = Math.min(100, Math.max(0, weightedScore))
              // Band segments, ascending: each colored slice ends where the
              // next band starts; the arc is clipped at the current score.
              const segments = [
                { from: 0, c: "#DC2626" },
                { from: Fider.session.tenant.scorecardBandLow, c: "#F97316" },
                { from: Fider.session.tenant.scorecardBandRefine, c: "#F59E0B" },
                { from: Fider.session.tenant.scorecardBandGood, c: "#2563EB" },
                { from: Fider.session.tenant.scorecardBandStrong, c: "#16A34A" },
              ]
              const stops: string[] = []
              for (let i = 0; i < segments.length; i++) {
                const start = segments[i].from
                const end = Math.min(i + 1 < segments.length ? segments[i + 1].from : 100, p)
                if (end <= start) break
                stops.push(`${segments[i].c} ${start}% ${end}%`)
              }
              stops.push(`color-mix(in srgb, var(--colors-gray-900) 9%, transparent) ${p}% 100%`)
              const ringGradient = `conic-gradient(from -90deg, ${stops.join(", ")})`
              // Dashes sit exactly where the arc changes color - band
              // thresholds that fall inside the filled part of the ring.
              const transitions = segments
                .slice(1)
                .map((seg) => seg.from)
                .filter((th) => th > 0 && th < p)
              return (
                <div
                  className="c-scorecard__ring"
                  style={
                    {
                      background: p === 0 ? undefined : ringGradient,
                      "--bc": weightedScore === 0 ? "var(--colors-gray-500)" : band.border,
                    } as React.CSSProperties
                  }
                >
                  {transitions.map((th) => (
                    <span key={th} className="c-scorecard__ring-tick" style={{ transform: `rotate(${th * 3.6}deg)` }} />
                  ))}
                  <div className="c-scorecard__ring-center">
                    <span className="c-scorecard__ring-num">{weightedScore}</span>
                    <span className="c-scorecard__ring-bandlabel" style={weightedScore === 0 ? undefined : { color: band.border }}>
                      {/* A card with nothing scored is "Not scored", not the bottom band — stage 5 starts at 1. */}
                      {weightedScore === 0 ? "Not scored" : band.label}
                    </span>
                  </div>
                </div>
              )
            })()}
            <div className="c-scorecard__ring-side">
              <span className="c-scorecard__gauge-label">Weighted score</span>
              <div className="c-scorecard__bands">
                {[
                  {
                    key: "strong",
                    label: Fider.session.tenant.scorecardBandStrongLabel || "Strong Candidate",
                    th: Fider.session.tenant.scorecardBandStrong,
                    c: "#16A34A",
                  },
                  {
                    key: "good",
                    label: Fider.session.tenant.scorecardBandGoodLabel || "Good Candidate",
                    th: Fider.session.tenant.scorecardBandGood,
                    c: "#2563EB",
                  },
                  {
                    key: "refine",
                    label: Fider.session.tenant.scorecardBandRefineLabel || "Needs Refinement",
                    th: Fider.session.tenant.scorecardBandRefine,
                    c: "#F59E0B",
                  },
                  { key: "low", label: Fider.session.tenant.scorecardBandLowLabel || "Low Priority", th: Fider.session.tenant.scorecardBandLow, c: "#F97316" },
                  { key: "reject", label: Fider.session.tenant.scorecardBandNoneLabel || "Not Recommended", th: 1, c: "#DC2626" },
                ].map((b) => (
                  <div key={b.key} className={`c-scorecard__bandrow ${weightedScore > 0 && band.key === b.key ? "c-scorecard__bandrow--cur" : ""}`}>
                    <span className="c-scorecard__banddot" style={{ background: b.c, boxShadow: `0 0 7px ${b.c}` }} />
                    <b>{b.label}</b>
                    <span className="c-scorecard__bandpts">{b.th}+ pts</span>
                  </div>
                ))}
              </div>
              <div className="c-scorecard__gauge-hint">
                {weightedScore === 0
                  ? "score any dimension to place this card"
                  : band.threshold > 0
                  ? `band threshold ≥ ${band.threshold}`
                  : `below ${Fider.session.tenant.scorecardBandLow}`}
              </div>
            </div>
          </div>

          <div className="c-scorecard__groups">
            {GROUP_ORDER.map((g) => {
              const rows = grouped[g]
              if (!rows || rows.length === 0) return null
              const sorted = [...rows].sort((a, b) => a.sortOrder - b.sortOrder)
              return (
                <div key={g} className="c-scorecard__group">
                  <div className="c-scorecard__group-head">
                    <span className="c-scorecard__group-label">{GROUP_LABELS[g]}</span>
                    <span className="c-scorecard__group-hint">{GROUP_HINTS[g]}</span>
                  </div>
                  {g === "scoring" ? <div>{sorted.map(renderField)}</div> : <div className="c-scorecard__grid">{sorted.map(renderField)}</div>}
                </div>
              )
            })}

            {Fider.session.user.isCollaborator && (
              <div className="c-scorecard__group">
                <Button
                  variant="danger"
                  onClick={async () => {
                    if (!window.confirm(`Delete scorecard "${title}"? This cannot be undone.`)) return
                    const r = await actions.deleteScorecard(props.scorecard.id)
                    if (r.ok) {
                      window.location.href = "/scorecard"
                    } else {
                      notify.error("Delete failed.")
                    }
                  }}
                >
                  Delete scorecard
                </Button>
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  )
}

export default ScorecardCard
