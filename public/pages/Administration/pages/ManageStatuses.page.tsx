import React from "react"
import { Button, ButtonClickEvent, Form, Input, Select, SelectOption, Toggle } from "@fider/components"
import { HStack, VStack } from "@fider/components/layout"
import { Status, StatusKind } from "@fider/models"
import { actions, Failure } from "@fider/services"
import { AdminBasePage } from "../components/AdminBasePage"

interface ManageStatusesPageProps {
  statuses: Status[]
}

interface ManageStatusesPageState {
  statuses: Status[]
  isAdding: boolean
  draftSlug: string
  draftLabel: string
  draftKind: StatusKind
  draftColor: string
  draftIcon: string
  draftShowOnHome: boolean
  draftFilterable: boolean
  draftSortOrder: number
  error?: Failure
  busy: boolean
}

const KIND_OPTIONS: SelectOption[] = [
  { value: "open", label: "Open — default initial state" },
  { value: "active", label: "Active — accepted, in progress" },
  { value: "closed-completed", label: "Closed (completed) — done, positive resolution" },
  { value: "closed-declined", label: "Closed (declined) — closed without action" },
  { value: "duplicate", label: "Duplicate — merged into another post" },
]

const COLOR_OPTIONS: SelectOption[] = [
  { value: "blue", label: "Blue" },
  { value: "green", label: "Green" },
  { value: "yellow", label: "Yellow" },
  { value: "red", label: "Red" },
  { value: "gray", label: "Gray" },
]

const slugify = (s: string): string =>
  s
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 50)

export default class ManageStatusesPage extends AdminBasePage<ManageStatusesPageProps, ManageStatusesPageState> {
  public id = "p-admin-statuses"
  public name = "statuses"
  public title = "Statuses"
  public subtitle = "Customize the list of post statuses your admins can apply"

  constructor(props: ManageStatusesPageProps) {
    super(props)
    this.state = {
      statuses: props.statuses ?? [],
      isAdding: false,
      draftSlug: "",
      draftLabel: "",
      draftKind: "open",
      draftColor: "blue",
      draftIcon: "lightbulb",
      draftShowOnHome: true,
      draftFilterable: true,
      draftSortOrder: 100,
      busy: false,
    }
  }

  private openAdd = () => {
    this.setState({
      isAdding: true,
      draftSlug: "",
      draftLabel: "",
      draftKind: "open",
      draftColor: "blue",
      draftIcon: "lightbulb",
      draftShowOnHome: true,
      draftFilterable: true,
      draftSortOrder: 100,
      error: undefined,
    })
  }

  private cancelAdd = () => {
    this.setState({ isAdding: false, error: undefined })
  }

  private updateLabel = (value: string) => {
    const auto = !this.state.draftSlug || this.state.draftSlug === slugify(this.state.draftLabel)
    this.setState({
      draftLabel: value,
      draftSlug: auto ? slugify(value) : this.state.draftSlug,
    })
  }

  private save = async (e: ButtonClickEvent) => {
    e.preventEnable()
    this.setState({ busy: true, error: undefined })
    const result = await actions.createStatus({
      slug: this.state.draftSlug,
      label: this.state.draftLabel,
      kind: this.state.draftKind,
      color: this.state.draftColor,
      icon: this.state.draftIcon,
      showOnHome: this.state.draftShowOnHome,
      filterable: this.state.draftFilterable,
      sortOrder: this.state.draftSortOrder,
    })
    if (result.ok) {
      this.setState({
        statuses: [...this.state.statuses, result.data].sort((a, b) => a.sortOrder - b.sortOrder),
        isAdding: false,
        busy: false,
      })
    } else {
      this.setState({ busy: false, error: result.error })
    }
  }

  private toggleActive = async (status: Status) => {
    const result = await actions.updateStatus(status.id, {
      label: status.label,
      color: status.color,
      icon: status.icon,
      showOnHome: status.showOnHome,
      filterable: status.filterable,
      sortOrder: status.sortOrder,
      isActive: !status.isActive,
    })
    if (result.ok) {
      this.setState({
        statuses: this.state.statuses.map((s) => (s.id === status.id ? { ...s, isActive: !s.isActive } : s)),
      })
    }
  }

  private remove = async (status: Status) => {
    if (status.isSystem) return
    if (!window.confirm(`Delete the "${status.label}" status? Posts already using it must be reassigned first.`)) return
    const result = await actions.deleteStatus(status.id)
    if (result.ok) {
      this.setState({ statuses: this.state.statuses.filter((s) => s.id !== status.id) })
    } else {
      window.alert((result.error?.errors?.[0]?.message ?? "Could not delete this status."))
    }
  }

  public content() {
    return (
      <VStack spacing={4}>
        <p className="text-sm text-muted">
          The 6 built-in statuses are seeded for every site and can be renamed, recolored, or deactivated but not deleted.
          Add custom statuses for your own workflow — e.g. <em>Under Review</em> between Open and Planned.
        </p>

        <table className="w-full text-sm">
          <thead>
            <tr className="text-left text-muted">
              <th className="py-2">Label</th>
              <th>Slug</th>
              <th>Kind</th>
              <th>Color</th>
              <th>Home</th>
              <th>Filter</th>
              <th>Active</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {this.state.statuses.map((s) => (
              <tr key={s.id} className="border-t">
                <td className="py-2">
                  {s.label} {s.isSystem && <span className="text-xs text-muted ml-1">(system)</span>}
                </td>
                <td><code>{s.slug}</code></td>
                <td>{s.kind}</td>
                <td>{s.color}</td>
                <td>{s.showOnHome ? "yes" : "—"}</td>
                <td>{s.filterable ? "yes" : "—"}</td>
                <td>
                  <Toggle active={s.isActive} onToggle={() => this.toggleActive(s)} />
                </td>
                <td>
                  {!s.isSystem && (
                    <Button variant="danger" size="small" onClick={() => this.remove(s)}>
                      Delete
                    </Button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {this.state.isAdding ? (
          <Form error={this.state.error}>
            <VStack spacing={4}>
              <Input field="label" label="Label" value={this.state.draftLabel} onChange={this.updateLabel} />
              <Input field="slug" label="Slug (URL-safe)" value={this.state.draftSlug} onChange={(v) => this.setState({ draftSlug: slugify(v) })} />
              <Select
                field="kind"
                label="Semantic kind"
                defaultValue={this.state.draftKind}
                options={KIND_OPTIONS}
                onChange={(opt) => this.setState({ draftKind: (opt?.value ?? "open") as StatusKind })}
              />
              <Select
                field="color"
                label="Color"
                defaultValue={this.state.draftColor}
                options={COLOR_OPTIONS}
                onChange={(opt) => this.setState({ draftColor: opt?.value ?? "blue" })}
              />
              <HStack spacing={4}>
                <Toggle active={this.state.draftShowOnHome} onToggle={(v) => this.setState({ draftShowOnHome: v })} />
                <span className="text-sm">Show posts in this status on the home page</span>
              </HStack>
              <HStack spacing={4}>
                <Toggle active={this.state.draftFilterable} onToggle={(v) => this.setState({ draftFilterable: v })} />
                <span className="text-sm">Include this status in the home-page filter</span>
              </HStack>
              <HStack spacing={2}>
                <Button variant="primary" onClick={this.save} disabled={this.state.busy}>
                  Save status
                </Button>
                <Button variant="tertiary" onClick={this.cancelAdd}>
                  Cancel
                </Button>
              </HStack>
            </VStack>
          </Form>
        ) : (
          <Button variant="primary" onClick={this.openAdd}>
            Add a custom status
          </Button>
        )}
      </VStack>
    )
  }
}
