import "../../Scorecard/Scorecard.scss"

import React from "react"
import { Button, Toggle, Form, Field, Input, Select, SelectOption } from "@fider/components"
import { actions, notify, Fider, Failure } from "@fider/services"
import { AdminBasePage } from "@fider/pages/Administration/components/AdminBasePage"
import { ScorecardFieldsSection } from "@fider/pages/Administration/components/ScorecardFieldsSection"

interface ScorecardSettingsState {
  isEnabled: boolean
  bandStrong: number
  bandGood: number
  bandRefine: number
  bandLow: number
  bandStrongLabel: string
  bandGoodLabel: string
  bandRefineLabel: string
  bandLowLabel: string
  bandNoneLabel: string
  triggerStatusSlug: string
  error?: Failure
}

export default class ScorecardSettingsPage extends AdminBasePage<any, ScorecardSettingsState> {
  public bare = true
  public id = "p-admin-scorecard-settings"
  public name = "scorecard-settings"
  public title = "Scorecard"
  public subtitle = "Feature toggle, scoring stages, and the field catalogue"

  constructor(props: any) {
    super(props)
    this.state = {
      isEnabled: Fider.session.tenant.isScorecardEnabled,
      bandStrong: Fider.session.tenant.scorecardBandStrong,
      bandGood: Fider.session.tenant.scorecardBandGood,
      bandRefine: Fider.session.tenant.scorecardBandRefine,
      bandLow: Fider.session.tenant.scorecardBandLow,
      bandStrongLabel: Fider.session.tenant.scorecardBandStrongLabel || "Strong Candidate",
      bandGoodLabel: Fider.session.tenant.scorecardBandGoodLabel || "Good Candidate",
      bandRefineLabel: Fider.session.tenant.scorecardBandRefineLabel || "Needs Refinement",
      bandLowLabel: Fider.session.tenant.scorecardBandLowLabel || "Low Priority",
      bandNoneLabel: Fider.session.tenant.scorecardBandNoneLabel || "Not Recommended",
      triggerStatusSlug: Fider.session.tenant.scorecardTriggerStatusSlug ?? "",
    }
  }

  private toggle = (active: boolean) => this.setState({ isEnabled: active })
  private setBand = (key: "bandStrong" | "bandGood" | "bandRefine" | "bandLow") => (value: string) => {
    const n = parseInt(value, 10)
    // Empty input leaves NaN — coerce to 0 so the descending-order check surfaces
    // a clear error instead of a mysterious "must be a number" NaN failure.
    this.setState({ [key]: Number.isNaN(n) ? 0 : n } as any)
  }
  private setLabel = (key: "bandStrongLabel" | "bandGoodLabel" | "bandRefineLabel" | "bandLowLabel" | "bandNoneLabel") => (value: string) => {
    this.setState({ [key]: value } as any)
  }

  private save = async () => {
    const result = await actions.updateScorecardSettings({
      isEnabled: this.state.isEnabled,
      bandStrong: this.state.bandStrong,
      bandGood: this.state.bandGood,
      bandRefine: this.state.bandRefine,
      bandLow: this.state.bandLow,
      bandStrongLabel: this.state.bandStrongLabel,
      bandGoodLabel: this.state.bandGoodLabel,
      bandRefineLabel: this.state.bandRefineLabel,
      bandLowLabel: this.state.bandLowLabel,
      bandNoneLabel: this.state.bandNoneLabel,
      triggerStatusSlug: this.state.triggerStatusSlug,
    })
    if (result.ok) {
      notify.success("Scorecard settings saved.")
      this.setState({ error: undefined })
    } else {
      this.setState({ error: result.error })
    }
  }

  public content() {
    const statusOptions: SelectOption[] = [
      { value: "", label: "— No auto-trigger — cards are created manually" },
      ...(Fider.session.tenant.statuses ?? []).filter((s) => s.isActive).map((s) => ({ value: s.slug, label: s.label })),
    ]
    return (
      <>
        <div className="c-scorecard__panel">
          <div className="c-scorecard__form">
            <div className="c-scorecard__form-head">
              <span className="c-scorecard__group-label">Scorecard</span>
              <span className="c-scorecard__group-hint">Feature toggle, auto-create trigger, and the five scoring stages</span>
            </div>
            <Form error={this.state.error}>
              <Field label="Enable scorecard">
                <Toggle disabled={!Fider.session.user.isCollaborator} active={this.state.isEnabled} onToggle={this.toggle} />
                <p className="text-muted mt-1">
                  When enabled, collaborators and administrators see a Scorecard page for reviewing and scoring ideas across the weighted dimensions you define.
                  Visitors never see it.
                </p>
              </Field>

              <Select
                field="triggerStatusSlug"
                label="Auto-create trigger status"
                defaultValue={this.state.triggerStatusSlug}
                options={statusOptions}
                onChange={(o) => this.setState({ triggerStatusSlug: o?.value ?? "" })}
              />
              <p className="text-muted -mt-2">
                When a post&apos;s status changes to this one, a scorecard is automatically created and linked to it. Change the list of statuses at{" "}
                <a href="/admin/statuses" className="text-link">
                  Site Settings → Statuses
                </a>
                . Select &quot;No auto-trigger&quot; if you&apos;d rather create every scorecard by hand.
              </p>

              <div className="c-scorecard__form-head mt-4">
                <span className="c-scorecard__group-label">Stages</span>
                <span className="c-scorecard__group-hint">
                  Rename any stage; thresholds must be strictly descending. Stage 5 catches every scored card below stage 4 — cards with no scores show as
                  &quot;Not scored&quot;.
                </span>
              </div>

              <div className="c-scorecard__form-grid" style={{ gridTemplateColumns: "1fr 110px", alignItems: "end" }}>
                <Input
                  field="bandStrongLabel"
                  label="Stage 1 name"
                  value={this.state.bandStrongLabel}
                  disabled={!Fider.session.user.isCollaborator}
                  onChange={this.setLabel("bandStrongLabel")}
                />
                <Input
                  field="bandStrong"
                  label="Score ≥"
                  value={String(this.state.bandStrong)}
                  disabled={!Fider.session.user.isCollaborator}
                  onChange={this.setBand("bandStrong")}
                />
                <Input
                  field="bandGoodLabel"
                  label="Stage 2 name"
                  value={this.state.bandGoodLabel}
                  disabled={!Fider.session.user.isCollaborator}
                  onChange={this.setLabel("bandGoodLabel")}
                />
                <Input
                  field="bandGood"
                  label="Score ≥"
                  value={String(this.state.bandGood)}
                  disabled={!Fider.session.user.isCollaborator}
                  onChange={this.setBand("bandGood")}
                />
                <Input
                  field="bandRefineLabel"
                  label="Stage 3 name"
                  value={this.state.bandRefineLabel}
                  disabled={!Fider.session.user.isCollaborator}
                  onChange={this.setLabel("bandRefineLabel")}
                />
                <Input
                  field="bandRefine"
                  label="Score ≥"
                  value={String(this.state.bandRefine)}
                  disabled={!Fider.session.user.isCollaborator}
                  onChange={this.setBand("bandRefine")}
                />
                <Input
                  field="bandLowLabel"
                  label="Stage 4 name"
                  value={this.state.bandLowLabel}
                  disabled={!Fider.session.user.isCollaborator}
                  onChange={this.setLabel("bandLowLabel")}
                />
                <Input
                  field="bandLow"
                  label="Score ≥"
                  value={String(this.state.bandLow)}
                  disabled={!Fider.session.user.isCollaborator}
                  onChange={this.setBand("bandLow")}
                />
                <Input
                  field="bandNoneLabel"
                  label="Stage 5 name (scored, below stage 4)"
                  value={this.state.bandNoneLabel}
                  disabled={!Fider.session.user.isCollaborator}
                  onChange={this.setLabel("bandNoneLabel")}
                />
                <p className="text-muted" style={{ margin: "0 0 14px" }}>
                  &lt; {this.state.bandLow}
                </p>
              </div>

              <div className="c-scorecard__form-actions">
                <Button variant="primary" onClick={this.save} disabled={!Fider.session.user.isCollaborator}>
                  Save
                </Button>
              </div>
            </Form>
          </div>
        </div>
        <div className="mt-4">
          <ScorecardFieldsSection fields={this.props.fields ?? []} usage={this.props.usage ?? {}} />
        </div>
      </>
    )
  }
}
