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
  webSearchEnabled: boolean
}

interface ManageAIPageProps {
  agents: AgentConfig[]
  enabled: boolean
  provider: string
  model: string
  customBaseUrl: string
  customModel: string
  hasKey: boolean
  webSearchEnabled: boolean
  webSearchProvider: string
  webSearchBaseUrl: string
  webSearchHasKey: boolean
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
  keyRevealed: boolean
  webSearchEnabled: boolean
  webSearchProvider: string
  webSearchApiKey: string
  webSearchBaseUrl: string
  webSearchHasKey: boolean
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
      agents.unshift({ id: 0, productId: null, description: "", instructions: "", enabled: true, webSearchEnabled: false })
    }
    for (const p of Fider.session.tenant.products ?? []) {
      if (!agents.some((a) => a.productId === p.id)) {
        agents.push({ id: 0, productId: p.id, description: "", instructions: "", enabled: false, webSearchEnabled: false })
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
      keyRevealed: false,
      webSearchEnabled: props.webSearchEnabled,
      webSearchProvider: props.webSearchProvider || "serper",
      webSearchApiKey: "",
      webSearchBaseUrl: props.webSearchBaseUrl,
      webSearchHasKey: props.webSearchHasKey,
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
      webSearchEnabled: this.state.webSearchEnabled,
      webSearchProvider: this.state.webSearchProvider,
      webSearchApiKey: this.state.webSearchApiKey,
      webSearchBaseUrl: this.state.webSearchBaseUrl,
    })
    this.setState({ busy: false })
    if (result.ok) {
      notify.success("AI settings saved.")
      this.setState((prev) => ({
        apiKey: "",
        hasKey: prev.hasKey || prev.apiKey.trim() !== "",
        webSearchApiKey: "",
        webSearchHasKey: prev.webSearchHasKey || prev.webSearchApiKey.trim() !== "",
      }))
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
      webSearchEnabled: agent.webSearchEnabled,
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
    // switching provider hides any revealed key again
    this.setState({ provider, model, keyRevealed: false })
  }

  // revealKey fetches the stored provider key so a custom-AI admin can see what
  // they saved (handy for self-hosted / local endpoints). Admin-only endpoint;
  // the server only returns it for the custom provider.
  private revealKey = async () => {
    const result = await actions.getAIProviderKey()
    if (result.ok && result.data.apiKey) {
      this.setState({ apiKey: result.data.apiKey, keyRevealed: true })
    } else {
      notify.error(result.error?.errors?.[0]?.message || "Couldn't reveal the saved key.")
    }
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
              <p style={{ fontSize: 12.5, fontWeight: 650, color: "var(--colors-green-600)", display: "flex", alignItems: "center", gap: 6, flexWrap: "wrap" }}>
                <span
                  style={{ width: 8, height: 8, borderRadius: "50%", background: "var(--colors-green-600)", boxShadow: "0 0 6px var(--colors-green-600)" }}
                />
                A key is saved for this site
                {s.provider === "custom" && !s.keyRevealed && (
                  <button
                    type="button"
                    onClick={this.revealKey}
                    style={{
                      background: "none",
                      border: "none",
                      padding: 0,
                      cursor: "pointer",
                      color: "var(--colors-primary-base)",
                      fontWeight: 650,
                      fontSize: 12.5,
                    }}
                  >
                    reveal saved key
                  </button>
                )}
                {s.keyRevealed && (
                  <span className="text-muted" style={{ fontWeight: 400 }}>
                    — shown above; it will not be re-hidden until you reload
                  </span>
                )}
              </p>
            ) : (
              <p className="text-muted" style={{ fontSize: 12 }}>
                No key saved yet. Keys are stored server-side. For hosted providers (Claude / OpenAI) they are write-only; for a Custom provider you can reveal
                the saved key here.
              </p>
            )}
          </Form>
          <Button variant="primary" disabled={s.busy} onClick={this.saveSettings}>
            Save AI settings
          </Button>
        </div>

        <div>
          <h2>Web search</h2>
          <p className="text-muted" style={{ fontSize: 13 }}>
            Let Vora look things up on the web to ground ideas in current information. Vora decides when to search based on each product&apos;s instructions
            below — tell her when it helps (e.g. &quot;search for existing tools before proposing one&quot;). Enable it per product in the cards below.
          </p>
          <HStack spacing={2} align="center" className="mt-2">
            <Toggle
              active={s.webSearchEnabled}
              onToggle={(webSearchEnabled) => this.setState({ webSearchEnabled })}
              label={s.webSearchEnabled ? "On" : "Off"}
            />
            <span className="text-muted" style={{ fontSize: 12 }}>
              master switch for web search
            </span>
          </HStack>
          {s.webSearchEnabled && (
            <>
              <HStack spacing={2} className="mt-2">
                {[
                  { value: "serper", label: "Serper.dev" },
                  { value: "searxng", label: "SearXNG (self-hosted)" },
                ].map((p) => (
                  <Button
                    key={p.value}
                    size="small"
                    variant={s.webSearchProvider === p.value ? "primary" : "tertiary"}
                    onClick={() => this.setState({ webSearchProvider: p.value })}
                  >
                    {p.label}
                  </Button>
                ))}
              </HStack>
              <Form className="mt-2">
                {s.webSearchProvider === "serper" ? (
                  <>
                    <Input
                      field="webSearchApiKey"
                      label="Serper.dev API key"
                      value={s.webSearchApiKey}
                      onChange={(webSearchApiKey) => this.setState({ webSearchApiKey })}
                      placeholder={s.webSearchHasKey ? "•••••••••••••••• enter a new key to replace the saved one" : "paste your Serper.dev API key"}
                      autoComplete="off"
                    />
                    <p className="text-muted" style={{ fontSize: 12 }}>
                      Get a free key at{" "}
                      <a href="https://serper.dev" target="_blank" rel="noopener noreferrer" style={{ color: "var(--colors-primary-base)" }}>
                        serper.dev
                      </a>{" "}
                      (2,500 free searches). {s.webSearchHasKey && <span style={{ color: "var(--colors-green-600)", fontWeight: 650 }}>● A key is saved.</span>}
                    </p>
                  </>
                ) : (
                  <>
                    <Input
                      field="webSearchBaseUrl"
                      label="SearXNG instance URL"
                      value={s.webSearchBaseUrl}
                      onChange={(webSearchBaseUrl) => this.setState({ webSearchBaseUrl })}
                      placeholder="https://searxng.example.com"
                    />
                    <p className="text-muted" style={{ fontSize: 12 }}>
                      Free &amp; open-source — self-host{" "}
                      <a href="https://docs.searxng.org" target="_blank" rel="noopener noreferrer" style={{ color: "var(--colors-primary-base)" }}>
                        SearXNG
                      </a>{" "}
                      and point Vora at it. Enable its JSON format (<code>search.formats: [html, json]</code>). No API key or per-query cost.
                    </p>
                  </>
                )}
              </Form>
              <Button variant="primary" disabled={s.busy} onClick={this.saveSettings}>
                Save AI settings
              </Button>
            </>
          )}
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
                    placeholder="Ask about the problem before solutions. Always cover workflow, owner, and success measure. Search the web when it helps ground the idea…"
                  />
                </Form>
                <HStack spacing={2} align="center" className="mt-2">
                  <Toggle
                    active={agent.webSearchEnabled}
                    onToggle={(webSearchEnabled) => this.setAgent(idx, { webSearchEnabled })}
                    disabled={!s.webSearchEnabled}
                  />
                  <span className="text-muted" style={{ fontSize: 12 }}>
                    {s.webSearchEnabled ? "let Vora search the web for this product" : "web search off site-wide — enable it above first"}
                  </span>
                </HStack>
                <Button size="small" variant="secondary" className="mt-2" onClick={() => this.saveAgent(agent)}>
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
