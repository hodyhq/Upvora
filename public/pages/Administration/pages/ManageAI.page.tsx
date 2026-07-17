import React from "react"
import { Button, Form, Input, TextArea, Toggle } from "@fider/components"
import { actions, notify, Fider } from "@fider/services"
import { AdminBasePage } from "../components/AdminBasePage"
import { HStack, VStack } from "@fider/components/layout"

interface AgentConfig {
  id: number
  productId: number | null
  description: string
  instructions: string
  enabled: boolean
}

interface ManageAIPageProps {
  agents: AgentConfig[]
  enabled: boolean
  provider: string
  model: string
  customBaseUrl: string
  customModel: string
  hasKey: boolean
}

interface ManageAIPageState {
  enabled: boolean
  provider: string
  apiKey: string
  model: string
  customBaseUrl: string
  customModel: string
  agents: AgentConfig[]
  busy: boolean
  hasKey: boolean
}

const claudeModels = [
  { value: "haiku", label: "Haiku" },
  { value: "sonnet-5", label: "Sonnet 5" },
]
const openaiModels = [
  { value: "luna", label: "GPT-5.6 Luna" },
  { value: "terra", label: "GPT-5.6 Terra" },
]

// Admin · AI — the Vora control room. The agent is always named Vora; per
// product you tune description, instructions and the switch.
export default class ManageAIPage extends AdminBasePage<ManageAIPageProps, ManageAIPageState> {
  public id = "p-admin-ai"
  public name = "ai"
  public title = "AI"
  public subtitle = "Vora, the ideation agent — one switch, one provider, per-product instructions"

  constructor(props: ManageAIPageProps) {
    super(props)
    const agents = [...props.agents]
    if (!agents.some((a) => a.productId === null)) {
      agents.unshift({ id: 0, productId: null, description: "", instructions: "", enabled: true })
    }
    for (const p of Fider.session.tenant.products ?? []) {
      if (!agents.some((a) => a.productId === p.id)) {
        agents.push({ id: 0, productId: p.id, description: "", instructions: "", enabled: false })
      }
    }
    this.state = {
      enabled: props.enabled,
      provider: props.provider || "claude",
      apiKey: "",
      model: props.model || (props.provider === "openai" ? "luna" : "sonnet-5"),
      customBaseUrl: props.customBaseUrl,
      customModel: props.customModel,
      agents,
      busy: false,
      hasKey: props.hasKey,
    }
  }

  private saveSettings = async () => {
    this.setState({ busy: true })
    const result = await actions.updateAISettings({
      enabled: this.state.enabled,
      provider: this.state.provider,
      apiKey: this.state.apiKey,
      model: this.state.model,
      customBaseUrl: this.state.customBaseUrl,
      customModel: this.state.customModel,
    })
    this.setState({ busy: false })
    if (result.ok) {
      notify.success("AI settings saved.")
      this.setState((prev) => ({ apiKey: "", hasKey: prev.hasKey || prev.apiKey.trim() !== "" }))
    } else {
      notify.error(result.error?.errors?.[0]?.message || "Failed to save AI settings.")
    }
  }

  private saveAgent = async (agent: AgentConfig) => {
    const result = await actions.upsertAIAgent({
      productId: agent.productId,
      description: agent.description,
      instructions: agent.instructions,
      enabled: agent.enabled,
    })
    if (result.ok) {
      notify.success("Vora updated.")
    } else {
      notify.error(result.error?.errors?.[0]?.message || "Failed to save the agent.")
    }
  }

  private setAgent(idx: number, patch: Partial<AgentConfig>) {
    const agents = [...this.state.agents]
    agents[idx] = { ...agents[idx], ...patch }
    this.setState({ agents })
  }

  private productName(productId: number | null): string {
    if (productId === null) {
      return (Fider.session.tenant.products?.length ?? 0) > 0 ? "Default (General)" : "Default"
    }
    return Fider.session.tenant.products?.find((p) => p.id === productId)?.name ?? `Product #${productId}`
  }

  private setModelsForProvider(provider: string) {
    const model = provider === "openai" ? "luna" : provider === "claude" ? "sonnet-5" : ""
    this.setState({ provider, model })
  }

  public content() {
    const s = this.state
    const models = s.provider === "claude" ? claudeModels : s.provider === "openai" ? openaiModels : []
    return (
      <VStack spacing={8}>
        <div>
          <h2>Ideation agent</h2>
          <HStack spacing={2} align="center" className="mt-2">
            <Toggle active={s.enabled} onToggle={(enabled) => this.setState({ enabled })} label={s.enabled ? "On" : "Off"} />
            <span className="text-muted" style={{ fontSize: 12 }}>
              master switch — off hides Vora everywhere
            </span>
          </HStack>
        </div>

        <div>
          <h2>Provider</h2>
          <HStack spacing={2} className="mt-2">
            {["claude", "openai", "custom"].map((p) => (
              <Button key={p} size="small" variant={s.provider === p ? "primary" : "tertiary"} onClick={() => this.setModelsForProvider(p)}>
                {p === "claude" ? "Claude" : p === "openai" ? "OpenAI" : "Custom"}
              </Button>
            ))}
          </HStack>
          {models.length > 0 && (
            <HStack spacing={2} className="mt-2">
              {models.map((m) => (
                <Button key={m.value} size="small" variant={s.model === m.value ? "primary" : "tertiary"} onClick={() => this.setState({ model: m.value })}>
                  {m.label}
                </Button>
              ))}
            </HStack>
          )}
          <Form className="mt-2">
            {s.provider === "custom" && (
              <>
                <Input
                  field="customBaseUrl"
                  label="Base URL (OpenAI-compatible)"
                  placeholder="https://llm.example.com/v1"
                  value={s.customBaseUrl}
                  onChange={(customBaseUrl) => this.setState({ customBaseUrl })}
                />
                <Input field="customModel" label="Model name" value={s.customModel} onChange={(customModel) => this.setState({ customModel })} />
              </>
            )}
            <Input
              field="apiKey"
              label="API key"
              value={s.apiKey}
              onChange={(apiKey) => this.setState({ apiKey })}
              placeholder={s.hasKey ? "•••••••••••••••• enter a new key to replace the saved one" : "paste the provider API key"}
              autoComplete="off"
            />
            {s.hasKey ? (
              <p style={{ fontSize: 12.5, fontWeight: 650, color: "var(--colors-green-600)", display: "flex", alignItems: "center", gap: 6 }}>
                <span
                  style={{ width: 8, height: 8, borderRadius: "50%", background: "var(--colors-green-600)", boxShadow: "0 0 6px var(--colors-green-600)" }}
                />
                A key is saved for this site
                <span className="text-muted" style={{ fontWeight: 400 }}>
                  — write-only; it is never shown or sent back to any browser
                </span>
              </p>
            ) : (
              <p className="text-muted" style={{ fontSize: 12 }}>
                No key saved yet. Keys are write-only: stored server-side, never shown or sent back to any browser.
              </p>
            )}
          </Form>
          <Button variant="primary" disabled={s.busy} onClick={this.saveSettings}>
            Save AI settings
          </Button>
        </div>

        <div>
          <h2>Vora, per product</h2>
          <p className="text-muted" style={{ fontSize: 13 }}>
            The agent is always named Vora. Tune what it says about itself and how it interviews for each product; General uses the default.
          </p>
          <VStack spacing={4} className="mt-2">
            {s.agents.map((agent, idx) => (
              <div key={`${agent.productId ?? "default"}`} style={{ border: "1px solid var(--colors-gray-200)", borderRadius: 10, padding: "12px 14px" }}>
                <HStack spacing={2} align="center">
                  <strong style={{ fontSize: 13 }}>{this.productName(agent.productId)}</strong>
                  <Toggle active={agent.enabled} onToggle={(enabled) => this.setAgent(idx, { enabled })} />
                </HStack>
                <Form className="mt-2">
                  <Input
                    field={`agent-desc-${idx}`}
                    label="Description (shown to users when the chat opens)"
                    value={agent.description}
                    maxLength={500}
                    onChange={(description) => this.setAgent(idx, { description })}
                    placeholder="I help you turn a rough thought into an idea the team can evaluate."
                  />
                  <TextArea
                    field={`agent-inst-${idx}`}
                    label="Instructions (how Vora interviews for this product)"
                    value={agent.instructions}
                    onChange={(instructions) => this.setAgent(idx, { instructions })}
                    placeholder="Ask about the problem before solutions. Always cover workflow, owner, and success measure…"
                  />
                </Form>
                <Button size="small" variant="secondary" onClick={() => this.saveAgent(agent)}>
                  Save agent
                </Button>
              </div>
            ))}
          </VStack>
        </div>
      </VStack>
    )
  }
}
