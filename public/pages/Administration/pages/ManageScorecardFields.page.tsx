import "../../Scorecard/Scorecard.scss"

import React from "react"
import { Button, Field, Form, Input, Select, SelectOption, TextArea, Toggle } from "@fider/components"
import { HStack, VStack } from "@fider/components/layout"
import { ScorecardField, ScorecardFieldChoice } from "@fider/models"
import { actions, Failure, Fider } from "@fider/services"
import { AdminBasePage } from "../components/AdminBasePage"

interface ManageScorecardFieldsPageProps {
  fields: ScorecardField[]
  usage: { [key: string]: number }
}

interface ManageScorecardFieldsPageState {
  fields: ScorecardField[]
  isAdding: boolean
  editingId: number | null
  draftKey: string
  draftLabel: string
  draftGroup: string
  draftType: string
  draftChoicesCSV: string
  draftWeight: string
  draftQuestion: string
  draftSortOrder: number
  draftIsActive: boolean
  error?: Failure
  busy: boolean
}

const groupOptions: SelectOption[] = [
  { value: "header", label: "Header (top of the card, next to the title)" },
  { value: "intake", label: "Intake" },
  { value: "context", label: "Context" },
  { value: "workflow", label: "Workflow" },
  { value: "ownership", label: "Ownership" },
  { value: "classification", label: "Classification" },
  { value: "scoring", label: "Scoring (weighted 1-5)" },
  { value: "decision", label: "Decision" },
]

const typeOptions: SelectOption[] = [
  { value: "text", label: "Text — single line" },
  { value: "note", label: "Note — multi-line" },
  { value: "date", label: "Date" },
  { value: "number", label: "Number" },
  { value: "url", label: "URL" },
  { value: "choice", label: "Choice — dropdown with pre-set options" },
  { value: "score", label: "Score — 1-5 slider that feeds the weighted score" },
  { value: "user", label: "User — pick a collaborator/admin or type any name" },
]

// Choices come in over the wire as [{value, color?, bucket?}]. Admin edits them
// as a comma-separated string where each entry is "Value" or "Value:bucket"
// (bucket = new | review | executive, used by the scorecard dashboard tabs).
// Colors — and buckets not restated in the CSV — are preserved by merging with
// the previous choice list by value, so editing the list never wipes them.
const VALID_BUCKETS = ["new", "review", "executive"]

const choicesToCsv = (raw: unknown): string => {
  if (!raw || !Array.isArray(raw)) return ""
  return raw
    .map((c: any) => {
      if (typeof c === "string") return c
      if (!c?.value) return ""
      return c.bucket ? `${c.value}:${c.bucket}` : c.value
    })
    .filter(Boolean)
    .join(", ")
}

const csvToChoices = (s: string, previous?: ScorecardFieldChoice[]): ScorecardFieldChoice[] => {
  const prevByValue = new Map((previous ?? []).map((c) => [c.value, c]))
  return s
    .split(",")
    .map((v) => v.trim())
    .filter((v) => v.length > 0)
    .map((entry) => {
      const sep = entry.lastIndexOf(":")
      const maybeBucket =
        sep >= 0
          ? entry
              .slice(sep + 1)
              .trim()
              .toLowerCase()
          : ""
      const hasBucket = VALID_BUCKETS.includes(maybeBucket)
      const value = hasBucket ? entry.slice(0, sep).trim() : entry
      const prev = prevByValue.get(value)
      const choice: ScorecardFieldChoice = { value }
      if (prev?.color) choice.color = prev.color
      const bucket = hasBucket ? (maybeBucket as ScorecardFieldChoice["bucket"]) : prev?.bucket
      if (bucket) choice.bucket = bucket
      return choice
    })
}

const slugifyKey = (s: string): string =>
  s
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "")
    .slice(0, 60)

const groupLabel = (key: string): string => {
  const g = groupOptions.find((o) => o.value === key)
  return g ? String(g.label) : key === "scoring" ? "Scoring" : key
}

export default class ManageScorecardFieldsPage extends AdminBasePage<ManageScorecardFieldsPageProps, ManageScorecardFieldsPageState> {
  public id = "p-admin-scorecard-fields"
  public name = "scorecard-fields"
  public title = "Scorecard Fields"
  public subtitle = "Customize the fields shown on every scorecard. Eight scoring dimensions are locked and drive the weighted score."

  constructor(props: ManageScorecardFieldsPageProps) {
    super(props)
    this.state = {
      fields: props.fields ?? [],
      isAdding: false,
      editingId: null,
      draftKey: "",
      draftLabel: "",
      draftGroup: "context",
      draftType: "text",
      draftChoicesCSV: "",
      draftWeight: "",
      draftQuestion: "",
      draftSortOrder: 100,
      draftIsActive: true,
      busy: false,
    }
  }

  private openAdd = () => {
    this.setState({
      isAdding: true,
      editingId: null,
      draftKey: "",
      draftLabel: "",
      draftGroup: "context",
      draftType: "text",
      draftChoicesCSV: "",
      draftWeight: "",
      draftQuestion: "",
      draftSortOrder: 100,
      draftIsActive: true,
      error: undefined,
    })
  }

  private openEdit = (f: ScorecardField) => {
    this.setState({
      isAdding: false,
      editingId: f.id,
      draftKey: f.key,
      draftLabel: f.label,
      draftGroup: f.groupKey,
      draftType: f.type,
      draftChoicesCSV: choicesToCsv(f.choices),
      draftWeight: f.weight != null ? String(f.weight) : "",
      draftQuestion: f.question ?? "",
      draftSortOrder: f.sortOrder,
      draftIsActive: f.isActive,
      error: undefined,
    })
  }

  private cancel = () => this.setState({ isAdding: false, editingId: null, error: undefined })

  private setLabel = (label: string) => {
    // Auto-derive the key from the label when adding, until the user edits key
    // directly. On edit we never touch the key (immutable).
    this.setState((s) => {
      const key = s.isAdding && (s.draftKey === "" || s.draftKey === slugifyKey(s.draftLabel)) ? slugifyKey(label) : s.draftKey
      return { ...s, draftLabel: label, draftKey: key }
    })
  }

  private save = async () => {
    const s = this.state
    const editingField = s.editingId != null ? this.state.fields.find((f) => f.id === s.editingId) : undefined
    let choicesForReq: ScorecardFieldChoice[] | undefined = undefined
    if (s.draftType === "choice") {
      const parsed = csvToChoices(s.draftChoicesCSV, editingField?.choices)
      if (parsed.length === 0) {
        this.setState({ error: { errors: [{ field: "choices", message: "Enter at least one option (comma-separated)." }] } })
        return
      }
      choicesForReq = parsed
    }

    const weightForReq = s.draftType === "score" && s.draftWeight !== "" ? parseInt(s.draftWeight, 10) : undefined

    this.setState({ busy: true })
    if (s.isAdding) {
      const result = await actions.createScorecardField({
        key: s.draftKey,
        label: s.draftLabel,
        groupKey: s.draftGroup,
        type: s.draftType as any,
        choices: choicesForReq,
        weight: Number.isNaN(weightForReq as number) ? undefined : weightForReq,
        question: s.draftType === "score" ? s.draftQuestion : undefined,
        sortOrder: s.draftSortOrder,
      })
      this.setState({ busy: false })
      if (!result.ok) {
        this.setState({ error: result.error })
        return
      }
      this.setState({
        isAdding: false,
        fields: [...this.state.fields, result.data as ScorecardField].sort(this.byGroupThenOrder),
      })
    } else if (s.editingId != null) {
      const editing = this.state.fields.find((f) => f.id === s.editingId)
      if (!editing) {
        this.setState({ busy: false, editingId: null })
        return
      }
      // System rows keep their choices unchanged (locked type=score means no
      // choices anyway); admin can still edit weight/question/label/order.
      const weightForReq = s.draftType === "score" && s.draftWeight !== "" ? parseInt(s.draftWeight, 10) : undefined
      const result = await actions.updateScorecardField(s.editingId, {
        label: s.draftLabel,
        choices: editing.type === "choice" ? choicesForReq : undefined,
        weight: Number.isNaN(weightForReq as number) ? undefined : weightForReq,
        question: s.draftType === "score" ? s.draftQuestion : undefined,
        sortOrder: s.draftSortOrder,
        isActive: s.draftIsActive,
      })
      this.setState({ busy: false })
      if (!result.ok) {
        this.setState({ error: result.error })
        return
      }
      this.setState({
        editingId: null,
        fields: this.state.fields
          .map((f) =>
            f.id === s.editingId
              ? {
                  ...f,
                  label: s.draftLabel,
                  choices: f.type === "choice" ? choicesForReq : f.choices,
                  weight: s.draftType === "score" ? (Number.isNaN(weightForReq as number) ? f.weight : weightForReq) : f.weight,
                  question: s.draftType === "score" ? s.draftQuestion : f.question,
                  sortOrder: s.draftSortOrder,
                  isActive: s.draftIsActive,
                }
              : f
          )
          .sort(this.byGroupThenOrder),
      })
    }
  }

  private remove = async (f: ScorecardField) => {
    const warning = f.isSystem
      ? `Delete the seeded field "${f.label}"? It has never been answered on any scorecard, so this is safe — but you'd have to rebuild it by hand if you change your mind.`
      : `Delete field "${f.label}"? It has never been answered on any scorecard, so nothing is lost.`
    if (!window.confirm(warning)) return
    const result = await actions.deleteScorecardField(f.id)
    if (result.ok) {
      this.setState({ fields: this.state.fields.filter((x) => x.id !== f.id) })
    } else {
      this.setState({ error: result.error })
    }
  }

  // One-click active toggle from the table row — the lifecycle action for
  // questions that already hold answers and therefore can't be deleted.
  private setActive = async (f: ScorecardField, isActive: boolean) => {
    const result = await actions.updateScorecardField(f.id, {
      label: f.label,
      choices: f.type === "choice" ? f.choices : undefined,
      weight: f.weight,
      question: f.question,
      sortOrder: f.sortOrder,
      isActive,
    })
    if (result.ok) {
      this.setState({ fields: this.state.fields.map((x) => (x.id === f.id ? { ...x, isActive } : x)) })
    } else {
      this.setState({ error: result.error })
    }
  }

  private move = async (f: ScorecardField, direction: -1 | 1) => {
    const same = this.state.fields.filter((x) => x.groupKey === f.groupKey).sort(this.byGroupThenOrder)
    const idx = same.findIndex((x) => x.id === f.id)
    const neighborIdx = idx + direction
    if (idx < 0 || neighborIdx < 0 || neighborIdx >= same.length) return
    const neighbor = same[neighborIdx]
    const newSortOrder = direction === -1 ? neighbor.sortOrder - 1 : neighbor.sortOrder + 1
    const result = await actions.updateScorecardField(f.id, {
      label: f.label,
      choices: f.type === "choice" ? f.choices : undefined,
      weight: f.weight,
      question: f.question,
      sortOrder: newSortOrder,
      isActive: f.isActive,
    })
    if (result.ok) {
      this.setState({
        fields: this.state.fields.map((x) => (x.id === f.id ? { ...x, sortOrder: newSortOrder } : x)).sort(this.byGroupThenOrder),
      })
    }
  }

  private byGroupThenOrder = (a: ScorecardField, b: ScorecardField): number => {
    const groupOrder = ["header", "intake", "context", "workflow", "ownership", "classification", "scoring", "decision"]
    const ga = groupOrder.indexOf(a.groupKey)
    const gb = groupOrder.indexOf(b.groupKey)
    if (ga !== gb) return ga - gb
    if (a.sortOrder !== b.sortOrder) return a.sortOrder - b.sortOrder
    return a.id - b.id
  }

  private weightSum(): number {
    return this.state.fields.filter((f) => f.type === "score" && f.isActive).reduce((sum, f) => sum + (f.weight ?? 0), 0)
  }

  public content() {
    const canEdit = Fider.session.user.isCollaborator
    const rows = [...this.state.fields].sort(this.byGroupThenOrder)
    const weightSum = this.weightSum()
    const weightSumOK = weightSum === 100

    return (
      <VStack spacing={4}>
        <div className={`c-scorecard__callout ${weightSumOK ? "" : "c-scorecard__callout--warn"}`}>
          <span className="c-scorecard__callout-num">{weightSum}</span>
          <div>
            <strong>Scoring weights</strong>
            {weightSumOK
              ? " — active weights sum to 100, the weighted score maps cleanly to 0–100."
              : " — active weights must sum to 100 for the weighted score to map to 0–100. Edit the weights of the scoring rows to rebalance."}
          </div>
        </div>

        {/* The form renders ABOVE the table so "+ Add a field" (in the toolbar
            right below) opens it in place — no scrolling past the catalogue. */}
        {(this.state.isAdding || this.state.editingId != null) && this.renderForm()}

        <div className="c-scorecard__panel">
          <div className="c-scorecard__toolbar">
            <div className="c-scorecard__group-head" style={{ marginBottom: 0, paddingBottom: 9 }}>
              <span className="c-scorecard__group-label">Field catalogue</span>
              <span className="c-scorecard__group-hint">Answered questions can be deactivated, never deleted — history stays intact</span>
            </div>
            <div className="c-scorecard__toolbar-right">
              {!this.state.isAdding && this.state.editingId == null && (
                <Button variant="primary" onClick={this.openAdd} disabled={!canEdit}>
                  + Add a field
                </Button>
              )}
            </div>
          </div>
          <div className="c-scorecard__table-wrap">
            <table className="c-scorecard__table c-scorecard__table--static">
              <thead>
                <tr>
                  <th>Order</th>
                  <th>Question</th>
                  <th>Type</th>
                  <th>Group</th>
                  <th className="is-right">Weight</th>
                  <th>Usage</th>
                  <th className="is-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((f) => {
                  const answers = this.props.usage?.[f.key] ?? 0
                  return (
                    <tr key={f.id} style={f.isActive ? undefined : { opacity: 0.55 }}>
                      <td>
                        <HStack spacing={1}>
                          <Button variant="tertiary" size="small" onClick={() => this.move(f, -1)} disabled={!canEdit}>
                            ↑
                          </Button>
                          <Button variant="tertiary" size="small" onClick={() => this.move(f, 1)} disabled={!canEdit}>
                            ↓
                          </Button>
                          <span className="c-scorecard__order-num">{f.sortOrder}</span>
                        </HStack>
                      </td>
                      <td>
                        <div className="c-scorecard__row-title">
                          {f.label}
                          {!f.isActive && <span className="c-scorecard__field-retired">inactive</span>}
                        </div>
                        <div className="c-scorecard__row-sub">
                          {f.key}
                          {f.isSystem ? " · system" : ""}
                        </div>
                      </td>
                      <td>
                        <span className="c-scorecard__chip c-scorecard__chip--neutral c-scorecard__chip--plain">{f.type}</span>
                      </td>
                      <td>{groupLabel(f.groupKey)}</td>
                      <td className="is-right">{f.type === "score" ? `${f.weight ?? 0}%` : "—"}</td>
                      <td>
                        {answers > 0
                          ? f.isActive
                            ? `${answers} card${answers === 1 ? "" : "s"}`
                            : `on ${answers} old card${answers === 1 ? "" : "s"} · hidden on new`
                          : "never answered"}
                      </td>
                      <td className="is-right">
                        <HStack spacing={2}>
                          <Button variant="tertiary" size="small" onClick={() => this.openEdit(f)} disabled={!canEdit}>
                            Edit
                          </Button>
                          {answers > 0 ? (
                            f.isActive ? (
                              <Button variant="secondary" size="small" onClick={() => this.setActive(f, false)} disabled={!canEdit}>
                                Deactivate
                              </Button>
                            ) : (
                              <Button variant="secondary" size="small" onClick={() => this.setActive(f, true)} disabled={!canEdit}>
                                Reactivate
                              </Button>
                            )
                          ) : (
                            <Button variant="danger" size="small" onClick={() => this.remove(f)} disabled={!canEdit}>
                              Delete
                            </Button>
                          )}
                        </HStack>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </div>
      </VStack>
    )
  }

  private renderForm() {
    const canEdit = Fider.session.user.isCollaborator
    return (
      <div className="c-scorecard__panel">
        <div className="c-scorecard__form">
          <div className="c-scorecard__form-head">
            <span className="c-scorecard__group-label">{this.state.isAdding ? "Add a field" : "Edit field"}</span>
            <span className="c-scorecard__group-hint">
              {this.state.isAdding ? "New questions appear on new scorecards immediately" : `key: ${this.state.draftKey}`}
            </span>
          </div>
          <Form error={this.state.error}>
            <div className="c-scorecard__form-grid">
              {/* Label is always editable */}
              <Input field="label" label="Label" value={this.state.draftLabel} onChange={this.setLabel} />

              {/* Key is only editable when adding. On edit, show read-only. */}
              {this.state.isAdding ? (
                <Input
                  field="key"
                  label="Key (machine name — lowercase letters, digits, underscores)"
                  value={this.state.draftKey}
                  onChange={(v) => this.setState({ draftKey: slugifyKey(v) })}
                />
              ) : (
                <Input field="key" label="Key" value={this.state.draftKey} disabled />
              )}

              {/* Group + type are only editable when adding. */}
              {this.state.isAdding ? (
                <>
                  <Select
                    field="groupKey"
                    label="Group"
                    defaultValue={this.state.draftGroup}
                    options={groupOptions}
                    onChange={(o) => this.setState({ draftGroup: o?.value ?? "context" })}
                  />
                  <Select
                    field="type"
                    label="Type"
                    defaultValue={this.state.draftType}
                    options={typeOptions}
                    onChange={(o) => this.setState({ draftType: o?.value ?? "text" })}
                  />
                </>
              ) : (
                <>
                  <Input field="groupKey" label="Group" value={groupLabel(this.state.draftGroup)} disabled />
                  <Input field="type" label="Type" value={this.state.draftType} disabled />
                </>
              )}

              {/* Weight + question are only meaningful for score-type rows. */}
              {this.state.draftType === "score" && (
                <>
                  <Input
                    field="weight"
                    label="Weight (0-100; active weights across all scoring rows must sum to 100)"
                    value={this.state.draftWeight}
                    onChange={(v) => this.setState({ draftWeight: v.replace(/[^0-9]/g, "") })}
                  />
                  <Input
                    field="sortOrder"
                    label="Sort order (lower shows first within its group)"
                    value={String(this.state.draftSortOrder)}
                    onChange={(v) => this.setState({ draftSortOrder: parseInt(v || "0", 10) || 0 })}
                  />
                  <div className="c-scorecard__form-wide">
                    <TextArea
                      field="question"
                      label="Question (shown next to the 1-5 slider on the card page)"
                      value={this.state.draftQuestion}
                      onChange={(v) => this.setState({ draftQuestion: v })}
                    />
                  </div>
                </>
              )}

              {/* Choices are only meaningful for choice-type rows. Comma-separated,
                      in the order you want them to appear in the dropdown. */}
              {this.state.draftType === "choice" && (
                <div className="c-scorecard__form-wide">
                  <Input
                    field="choices"
                    label="Choices (comma-separated, in order — add :new / :review / :executive after a value to map it to a dashboard tab)"
                    value={this.state.draftChoicesCSV}
                    onChange={(v) => this.setState({ draftChoicesCSV: v })}
                  />
                </div>
              )}

              {this.state.draftType !== "score" && (
                <Input
                  field="sortOrder"
                  label="Sort order (lower shows first within its group)"
                  value={String(this.state.draftSortOrder)}
                  onChange={(v) => this.setState({ draftSortOrder: parseInt(v || "0", 10) || 0 })}
                />
              )}

              {!this.state.isAdding && (
                <Field label="Active">
                  <Toggle active={this.state.draftIsActive} onToggle={(v) => this.setState({ draftIsActive: v })} />
                  <p className="text-muted mt-1">Deactivating hides the field from new cards without deleting it — cards that answered it keep showing it.</p>
                </Field>
              )}
            </div>

            <div className="c-scorecard__form-actions">
              <Button variant="primary" onClick={this.save} disabled={this.state.busy || !canEdit}>
                {this.state.isAdding ? "Add field" : "Save changes"}
              </Button>
              <Button variant="tertiary" onClick={this.cancel}>
                Cancel
              </Button>
            </div>
          </Form>
        </div>
      </div>
    )
  }
}
