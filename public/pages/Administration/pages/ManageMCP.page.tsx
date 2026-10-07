import React from "react"
import { Button, Field, Form, Input, Select, Toggle } from "@fider/components"
import { actions, notify, Fider } from "@fider/services"
import { MCPClient, MCPMinRole, MCPSettings } from "@fider/services/actions/oauth"
import { AdminBasePage } from "../components/AdminBasePage"

interface ManageMCPPageProps {
  clients: MCPClient[]
  mcpUrl: string
}

interface ManageMCPPageState extends MCPSettings {
  clients: MCPClient[]
  newName: string
  newRedirect: string
}

const roleOptions = [
  { value: "administrator", label: "Administrators only" },
  { value: "collaborator", label: "Administrators and collaborators" },
  { value: "visitor", label: "Any signed-in member" },
]

export default class ManageMCPPage extends AdminBasePage<ManageMCPPageProps, ManageMCPPageState> {
  public id = "p-admin-mcp"
  public name = "mcp"
  public title = "MCP"
  public subtitle = "Let AI assistants (MCP clients) work in Upvora as the signed-in user"

  private confirmed: MCPSettings
  private saveSeq = 0
  private saving: Promise<void> = Promise.resolve()

  constructor(props: ManageMCPPageProps) {
    super(props)
    const tenant = Fider.session.tenant
    const settings: MCPSettings = {
      enabled: !!tenant.mcpEnabled,
      minRole: tenant.mcpMinRole || "administrator",
      dcrEnabled: tenant.mcpDcrEnabled ?? true,
    }
    this.state = { ...settings, clients: props.clients || [], newName: "", newRedirect: "" }
    this.confirmed = settings
  }

  // Saves run one at a time; only the latest may roll back, and only to the
  // last settings the server accepted (same pattern as the Privacy page).
  private updateSettings = (patch: Partial<MCPSettings>) => {
    const next: MCPSettings = {
      enabled: this.state.enabled,
      minRole: this.state.minRole,
      dcrEnabled: this.state.dcrEnabled,
      ...patch,
    }
    const seq = ++this.saveSeq
    this.setState(next as ManageMCPPageState)
    this.saving = this.saving.then(async () => {
      const response = await actions.updateMCPSettings(next).catch(() => ({ ok: false }))
      if (response.ok) {
        this.confirmed = next
        notify.success("Your MCP settings have been saved.")
      } else if (seq === this.saveSeq) {
        this.setState(this.confirmed as ManageMCPPageState)
      }
    })
  }

  private createClient = async (name: string, redirect: string) => {
    const result = await actions.createMCPClient(name.trim(), [redirect.trim()])
    if (result.ok && result.data) {
      const created = result.data
      this.setState((s) => ({ ...s, clients: [created, ...s.clients], newName: "", newRedirect: "" }))
    }
  }

  private deleteClient = async (id: number) => {
    const result = await actions.deleteMCPClient(id)
    if (result.ok) {
      this.setState((s) => ({ ...s, clients: s.clients.filter((c) => c.id !== id) }))
    }
  }

  public content() {
    return (
      <Form>
        <Field label="Enable MCP">
          <Toggle active={this.state.enabled} onToggle={(v) => this.updateSettings({ enabled: v })} />
          <p className="text-muted mt-1">
            When on, MCP clients such as Claude can connect at <code>{this.props.mcpUrl}</code>. Each person signs in with this site&apos;s normal sign-in and
            approves the connection; the client can then do only what that person can do.
          </p>
        </Field>
        <Select
          field="minRole"
          label="Who may connect"
          defaultValue={this.state.minRole}
          options={roleOptions}
          onChange={(o) => o && this.updateSettings({ minRole: o.value as MCPMinRole })}
        />
        <Field label="Clients may register themselves">
          <Toggle active={this.state.dcrEnabled} onToggle={(v) => this.updateSettings({ dcrEnabled: v })} />
          <p className="text-muted mt-1">
            Lets an MCP client register automatically. Registering grants nothing on its own: people still sign in and approve. Turn this off to allow only the
            clients listed below.
          </p>
        </Field>

        <h3 className="text-display mt-4">Registered clients</h3>
        {this.state.clients.length === 0 ? (
          <p className="text-muted">No clients yet.</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr>
                <th className="text-left">Name</th>
                <th className="text-left">Client ID</th>
                <th className="text-left">Returns to</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {this.state.clients.map((c) => (
                <tr key={c.id}>
                  <td>{c.name}</td>
                  <td>
                    <code>{c.clientId}</code>
                  </td>
                  <td>{c.redirectUris.join(", ")}</td>
                  <td>
                    <Button variant="danger" size="small" onClick={() => this.deleteClient(c.id)}>
                      Remove
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        <p className="text-muted mt-1">Removing a client disconnects everyone who approved it.</p>

        <h3 className="text-display mt-4">Register a client</h3>
        <Input field="newName" label="Name" value={this.state.newName} onChange={(v) => this.setState({ newName: v } as ManageMCPPageState)} />
        <Input
          field="newRedirect"
          label="Redirect URI"
          placeholder="https://claude.ai/api/mcp/auth_callback"
          value={this.state.newRedirect}
          onChange={(v) => this.setState({ newRedirect: v } as ManageMCPPageState)}
        />
        <Button variant="secondary" onClick={() => this.createClient(this.state.newName, this.state.newRedirect)}>
          Register client
        </Button>
      </Form>
    )
  }
}
