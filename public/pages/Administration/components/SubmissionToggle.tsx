import React, { useState } from "react"
import { Toggle } from "@fider/components"
import { actions, Fider, notify } from "@fider/services"

interface SubmissionToggleProps {
  field: "voraRequired" | "fileAttachmentsEnabled"
  label: string
  children?: React.ReactNode
}

// SubmissionToggle saves one idea-submission switch as soon as it changes;
// the server leaves the other one as it is.
export const SubmissionToggle = (props: SubmissionToggleProps) => {
  const [active, setActive] = useState(!!Fider.session.tenant[props.field])
  const [saving, setSaving] = useState(false)

  const change = async (value: boolean) => {
    setSaving(true)
    setActive(value)
    const result = await actions.updateSubmissionSettings({ [props.field]: value })
    setSaving(false)
    if (result.ok) {
      Fider.session.tenant[props.field] = value
      notify.success("Saved.")
    } else {
      setActive(!value) // http already shows the error
    }
  }

  return (
    <div className="c-form-field">
      <label>{props.label}</label>
      <Toggle
        active={active}
        disabled={saving || !Fider.session.user.isAdministrator}
        onToggle={change}
        label={active ? "On" : "Off"}
        ariaLabel={props.label}
      />
      {props.children && <div className="text-muted mt-1">{props.children}</div>}
    </div>
  )
}
