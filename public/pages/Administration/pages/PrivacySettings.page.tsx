import React from "react"
import { Toggle, Form, Field } from "@fider/components"
import { actions, notify, Fider } from "@fider/services"
import { AdminBasePage } from "@fider/pages/Administration/components/AdminBasePage"

export interface PrivacySettingsPageState {
  isPrivate: boolean
  isFeedEnabled: boolean
  isModerationEnabled: boolean
  membersPrivateIdeas: boolean
  membersCanPublishPrivate: boolean
}

export default class PrivacySettingsPage extends AdminBasePage<any, PrivacySettingsPageState> {
  public id = "p-admin-privacy"
  public name = "privacy"
  public title = "Privacy"
  public subtitle = "Manage your site's privacy"

  constructor(props: any) {
    super(props)

    this.state = {
      isPrivate: Fider.session.tenant.isPrivate,
      isFeedEnabled: Fider.session.tenant.isFeedEnabled,
      isModerationEnabled: Fider.session.tenant.isModerationEnabled,
      membersPrivateIdeas: Fider.session.tenant.membersPrivateIdeas,
      membersCanPublishPrivate: Fider.session.tenant.membersCanPublishPrivate,
    }
    this.confirmed = this.state
  }

  private confirmed: PrivacySettingsPageState // last settings the server accepted
  private saveSeq = 0
  private saving: Promise<void> = Promise.resolve()

  // Applies a partial change and saves the whole settings object. Saves run
  // one at a time, so the server applies them in click order; only the latest
  // save may roll back, and only to the last settings the server accepted.
  private updatePrivacy = (patch: Partial<PrivacySettingsPageState>) => {
    const next = { ...this.state, ...patch }
    if (next.isPrivate) next.isFeedEnabled = false // Disable feed if site is private
    const seq = ++this.saveSeq
    this.setState(next)
    this.saving = this.saving.then(async () => {
      const response = await actions.updateTenantPrivacy(next)
      if (response.ok) {
        this.confirmed = next
        notify.success("Your privacy settings have been saved.")
      } else if (seq === this.saveSeq) {
        this.setState(this.confirmed) // http already shows the error
      }
    })
  }

  public content() {
    return (
      <Form>
        <Field label="Private Board">
          <Toggle
            disabled={!Fider.session.user.isAdministrator}
            active={this.state.isPrivate}
            onToggle={(active) => this.updatePrivacy({ isPrivate: active })}
          />
          <p className="text-muted mt-1">
            A private board prevents unauthenticated users from viewing or interacting with its content. <br /> When enabled, only already registered users,
            invited users and users from trusted OAuth providers will have access to this board. Disables the feed feature.
          </p>
        </Field>
        <Field label="ATOM Feed">
          <Toggle
            disabled={!Fider.session.user.isAdministrator || this.state.isPrivate}
            active={this.state.isFeedEnabled}
            onToggle={(enabled) => this.updatePrivacy({ isFeedEnabled: enabled })}
          />
          <p className="text-muted mt-1">
            This feature lets users access this board via a feed reader. <br /> When enabled, the board makes its posts and comments available using the ATOM
            format. Links to feeds and autodiscovery metadata are shown on the board.
          </p>
        </Field>
        {Fider.session.tenant.isPro && (
          <Field label="Content Moderation">
            <Toggle
              disabled={!Fider.session.user.isAdministrator}
              active={this.state.isModerationEnabled}
              onToggle={(enabled) => this.updatePrivacy({ isModerationEnabled: enabled })}
            />
            <p className="text-muted mt-1">
              When enabled, new posts and comments will require approval from an administrator before being visible to other users. <br />
              Content creators can see their own unmoderated content, but it will be hidden from other users until approved.
            </p>
          </Field>
        )}
        <Field label="Member Private Ideas">
          <Toggle
            disabled={!Fider.session.user.isAdministrator}
            active={this.state.membersPrivateIdeas}
            onToggle={(enabled) => this.updatePrivacy({ membersPrivateIdeas: enabled })}
          />
          <p className="text-muted mt-1">
            Lets members mark an idea private when they submit it. <br /> A private idea is visible only to its author, collaborators and administrators.
          </p>
        </Field>
        <Field label="Members Can Publish Private Ideas">
          <Toggle
            disabled={!Fider.session.user.isAdministrator}
            active={this.state.membersCanPublishPrivate}
            onToggle={(enabled) => this.updatePrivacy({ membersCanPublishPrivate: enabled })}
          />
          <p className="text-muted mt-1">
            Lets a member make their own private idea public later. <br /> Members can never make an already-submitted idea private.
          </p>
        </Field>
      </Form>
    )
  }
}
