import React from "react"
import { Button, Toggle, Form, Field, Input, Select, SelectOption } from "@fider/components"
import { actions, notify, Fider, Failure } from "@fider/services"
import { AdminBasePage } from "@fider/pages/Administration/components/AdminBasePage"

interface ScorecardSettingsState {
  isEnabled: boolean
  bandStrong: number
  bandGood: number
  bandRefine: number
  bandLow: number
  triggerStatusSlug: string
  error?: Failure
}

export default class ScorecardSettingsPage extends AdminBasePage<any, ScorecardSettingsState> {
  public id = "p-admin-scorecard-settings"
  public name = "scorecard-settings"
  public title = "Scorecard"
  public subtitle = "Enable the scorecard feature and set the band thresholds"

  constructor(props: any) {
    super(props)
    this.state = {
      isEnabled: Fider.session.tenant.isScorecardEnabled,
      bandStrong: Fider.session.tenant.scorecardBandStrong,
      bandGood: Fider.session.tenant.scorecardBandGood,
      bandRefine: Fider.session.tenant.scorecardBandRefine,
      bandLow: Fider.session.tenant.scorecardBandLow,
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

  private save = async () => {
    const result = await actions.updateScorecardSettings({
      isEnabled: this.state.isEnabled,
      bandStrong: this.state.bandStrong,
      bandGood: this.state.bandGood,
      bandRefine: this.state.bandRefine,
      bandLow: this.state.bandLow,
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
      <Form error={this.state.error}>
        <Field label="Enable scorecard">
          <Toggle disabled={!Fider.session.user.isAdministrator} active={this.state.isEnabled} onToggle={this.toggle} />
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

        <p className="text-muted mt-4">
          Band thresholds decide how a weighted score (0–100) maps to a label. Must be strictly descending: Strong &gt; Good &gt; Needs Refinement &gt; Low.
          Anything below the low threshold is labelled Not Recommended.
        </p>

        <Input
          field="bandStrong"
          label="Strong Candidate ≥"
          value={String(this.state.bandStrong)}
          disabled={!Fider.session.user.isAdministrator}
          onChange={this.setBand("bandStrong")}
        />
        <Input
          field="bandGood"
          label="Good Candidate ≥"
          value={String(this.state.bandGood)}
          disabled={!Fider.session.user.isAdministrator}
          onChange={this.setBand("bandGood")}
        />
        <Input
          field="bandRefine"
          label="Needs Refinement ≥"
          value={String(this.state.bandRefine)}
          disabled={!Fider.session.user.isAdministrator}
          onChange={this.setBand("bandRefine")}
        />
        <Input
          field="bandLow"
          label="Low Priority ≥"
          value={String(this.state.bandLow)}
          disabled={!Fider.session.user.isAdministrator}
          onChange={this.setBand("bandLow")}
        />

        <Button variant="primary" onClick={this.save} disabled={!Fider.session.user.isAdministrator}>
          Save
        </Button>
      </Form>
    )
  }
}
