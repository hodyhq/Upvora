import React, { useMemo, useState } from "react"
import { Button, Header, Input, Select, SelectOption, TextArea, Form } from "@fider/components"
import { HStack, VStack } from "@fider/components/layout"
import { ScorecardField, Post, User } from "@fider/models"
import { actions, notify, Fider } from "@fider/services"

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

const parseChoices = (raw: unknown): { value: string }[] => {
  if (!raw || !Array.isArray(raw)) return []
  return raw.map((c: any) => (typeof c === "string" ? { value: c } : { value: String(c?.value ?? "") })).filter((c) => c.value !== "")
}

const formatDate = (iso?: string): string => (iso ? new Date(iso).toLocaleString() : "—")

const bandForScore = (score: number): { label: string; color: string } => {
  const t = Fider.session.tenant
  if (score >= t.scorecardBandStrong) return { label: "Strong Candidate", color: "bg-green-100 text-green-800" }
  if (score >= t.scorecardBandGood) return { label: "Good Candidate", color: "bg-blue-100 text-blue-800" }
  if (score >= t.scorecardBandRefine) return { label: "Needs Refinement", color: "bg-yellow-100 text-yellow-800" }
  if (score >= t.scorecardBandLow) return { label: "Low Priority", color: "bg-orange-100 text-orange-800" }
  return { label: "Not Recommended", color: "bg-red-100 text-red-800" }
}

const ScorecardCard: React.FC<ScorecardCardPageProps> = (props) => {
  const fields = (Fider.session.tenant.scorecardFields ?? []).filter((f) => f.isActive)
  const [title, setTitle] = useState(props.scorecard.title)
  const [values, setValues] = useState<Record<string, unknown>>(props.scorecard.values ?? {})
  const [saving, setSaving] = useState(false)

  const setValue = (key: string, v: unknown) => setValues((prev) => ({ ...prev, [key]: v }))

  // Live weighted score: same math as the Go compute. Score fields carry
  // a 1-5 int in values; missing/0 contributes 0.
  const { weightedScore, band } = useMemo(() => {
    let total = 0
    for (const f of fields) {
      if (f.type !== "score" || !f.weight) continue
      const raw = values[f.key]
      const n = typeof raw === "number" ? raw : parseInt(String(raw ?? "0"), 10) || 0
      total += n * (f.weight ?? 0)
    }
    const score = Math.round(total / 5)
    return { weightedScore: score, band: bandForScore(score) }
  }, [values, fields])

  const grouped: Record<string, ScorecardField[]> = {}
  for (const f of fields) {
    ;(grouped[f.groupKey] ??= []).push(f)
  }

  const save = async () => {
    setSaving(true)
    const result = await actions.updateScorecard(props.scorecard.id, { title, values })
    setSaving(false)
    if (result.ok) {
      notify.success("Scorecard saved.")
    } else {
      notify.error("Save failed. Try again.")
    }
  }

  const post = props.scorecard.post

  const renderField = (f: ScorecardField) => {
    const key = f.key
    const val = values[key]
    const strVal = val == null ? "" : String(val)

    if (f.type === "score") {
      const n = typeof val === "number" ? val : parseInt(strVal, 10) || 0
      return (
        <div key={key} className="mb-4">
          <div className="flex justify-between items-baseline mb-1">
            <label className="font-semibold">
              {f.label} <span className="text-xs text-muted">weight {f.weight ?? 0}</span>
            </label>
            <span className="text-sm text-muted">{n === 0 ? "not scored" : `${n} / 5`}</span>
          </div>
          {f.question && <p className="text-sm text-muted mb-1">{f.question}</p>}
          <input
            type="range"
            min={0}
            max={5}
            step={1}
            value={n}
            onChange={(e) => setValue(key, parseInt(e.target.value, 10))}
            className="w-full"
          />
        </div>
      )
    }

    if (f.type === "choice") {
      const choices = parseChoices(f.choices)
      const opts: SelectOption[] = [{ value: "", label: "— none —" }, ...choices.map((c) => ({ value: c.value, label: c.value }))]
      return (
        <Select
          key={key}
          field={key}
          label={f.label}
          defaultValue={strVal}
          options={opts}
          onChange={(o) => setValue(key, o?.value ?? "")}
        />
      )
    }

    if (f.type === "note") {
      return <TextArea key={key} field={key} label={f.label} value={strVal} onChange={(v) => setValue(key, v)} />
    }

    if (f.type === "date") {
      return (
        <div key={key} className="mb-4">
          <label className="font-semibold block mb-1">{f.label}</label>
          <input type="date" value={strVal} onChange={(e) => setValue(key, e.target.value)} className="c-input" />
        </div>
      )
    }

    if (f.type === "user") {
      const listId = `scorecard-users-${key}`
      const names = props.assigneeNames ?? []
      return (
        <div key={key} className="mb-4">
          <label className="font-semibold block mb-1">{f.label}</label>
          <input
            type="text"
            list={listId}
            value={strVal}
            onChange={(e) => setValue(key, e.target.value)}
            placeholder="Pick a collaborator/admin or type any name…"
            className="c-input w-full"
          />
          <datalist id={listId}>
            {names.map((n) => <option key={n} value={n} />)}
          </datalist>
        </div>
      )
    }

    // text, number, url — one-line input; type attr handled by native browser
    return <Input key={key} field={key} label={f.label} value={strVal} onChange={(v) => setValue(key, v)} />
  }

  const currentStatusLabel =
    post
      ? (Fider.session.tenant.statuses ?? []).find((s) => s.slug === (post as any).status)?.label ?? (post as any).status
      : ""

  return (
    <>
      <Header />
      <div id="p-scorecard-card" className="page container">
        <VStack spacing={4}>
          <a href="/scorecard" className="text-link text-sm">← All scorecards</a>

          {post && (
            <div className="p-4 bg-white rounded shadow">
              <VStack spacing={2}>
                <HStack className="justify-between items-start">
                  <div>
                    <h2 className="text-header">{post.title}</h2>
                    <p className="text-sm text-muted">
                      Submitted by <strong>{post.user?.name ?? "—"}</strong> on {formatDate(post.createdAt)}
                    </p>
                    <p className="text-sm text-muted">
                      Submitted to <strong>{currentStatusLabel || "—"}</strong> on {formatDate(props.scorecard.createdAt)}
                    </p>
                  </div>
                  <a className="c-button c-button--tertiary" href={`/posts/${post.number}/${post.slug}`} target="_blank" rel="noopener">
                    View idea →
                  </a>
                </HStack>
                {post.description && <p className="whitespace-pre-wrap">{post.description}</p>}
                <HStack spacing={4} className="text-sm text-muted">
                  <span>👍 {(post as any).votesCount ?? 0} votes</span>
                  <span>💬 {(post as any).commentsCount ?? 0} comments</span>
                  {Array.isArray(post.tags) && post.tags.length > 0 && (
                    <span>🏷 {post.tags.join(", ")}</span>
                  )}
                </HStack>
              </VStack>
            </div>
          )}

          <Form>
            <Input field="title" label="Scorecard title" value={title} onChange={setTitle} />

            {/* Weighted score gauge — recomputes on every value change. */}
            <div className={`p-3 rounded ${band.color}`}>
              <div className="flex justify-between items-baseline">
                <strong>Weighted score</strong>
                <div className="flex items-baseline gap-3">
                  <span className="text-2xl font-bold">{weightedScore}</span>
                  <span className="font-semibold">{band.label}</span>
                </div>
              </div>
              <p className="text-xs mt-1">Formula: round(sum(value × weight) / 5) across active scoring dimensions. Bands set in Admin → Scorecard settings.</p>
            </div>

            {GROUP_ORDER.map((g) => {
              const rows = grouped[g]
              if (!rows || rows.length === 0) return null
              return (
                <div key={g} className="mt-4">
                  <h3 className="text-md font-semibold border-b pb-1 mb-3">{GROUP_LABELS[g]}</h3>
                  {rows.sort((a, b) => a.sortOrder - b.sortOrder).map(renderField)}
                </div>
              )
            })}

            <HStack spacing={2}>
              <Button variant="primary" onClick={save} disabled={saving}>
                {saving ? "Saving…" : "Save"}
              </Button>
              {Fider.session.user.isAdministrator && (
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
              )}
            </HStack>
          </Form>
        </VStack>
      </div>
    </>
  )
}

export default ScorecardCard
