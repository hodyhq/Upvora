import React from "react"
import { Button, Form } from "@fider/components"
import { HStack, VStack } from "@fider/components/layout"
import { actions, notify, Fider, Failure } from "@fider/services"
import { AdminBasePage } from "../components/AdminBasePage"
import { ColorInput } from "../components/ColorInput"

const ACCENT_SURFACES: { key: string; label: string; hint: string }[] = [
  { key: "buttons", label: "Buttons & CTA", hint: "Primary buttons, Share an Idea" },
  { key: "votes", label: "Vote controls", hint: "Voted state on vote blocks" },
  { key: "links", label: "Links", hint: "Text links everywhere" },
  { key: "header", label: "Header accent", hint: "Active-nav underline" },
]

interface ThemeState {
  primaryMode: "default" | "custom"
  primary: string
  accents: { [key: string]: string } // key -> hex; missing = follow brand
  defaultTheme: string
  error?: Failure
  busy: boolean
}

export default class ManageThemePage extends AdminBasePage<any, ThemeState> {
  public id = "p-admin-theme"
  public name = "theme"
  public title = "Theme"
  public subtitle = "Colors as settings, not stylesheets — token overrides survive every redesign"

  constructor(props: any) {
    super(props)
    const t = Fider.session.tenant
    this.state = {
      primaryMode: t.themePrimary ? "custom" : "default",
      primary: t.themePrimary || "#F97316",
      accents: { ...(t.themeAccents ?? {}) },
      defaultTheme: t.defaultTheme || "light",
      busy: false,
    }
  }

  private setAccent = (key: string, hex: string) => {
    this.setState((s) => {
      const accents = { ...s.accents }
      if (hex === "") {
        delete accents[key]
      } else {
        accents[key] = hex
      }
      return { ...s, accents }
    })
  }

  private save = async () => {
    this.setState({ busy: true })
    const result = await actions.updateTenantTheme({
      primary: this.state.primaryMode === "default" ? "" : this.state.primary,
      accents: this.state.accents,
      defaultTheme: this.state.defaultTheme,
    })
    this.setState({ busy: false })
    if (result.ok) {
      notify.success("Theme saved — reloading to apply.")
      setTimeout(() => location.reload(), 600)
    } else {
      this.setState({ error: result.error })
    }
  }

  public content() {
    const isAdmin = Fider.session.user.isAdministrator
    return (
      <Form error={this.state.error}>
        <VStack spacing={6}>
          <div>
            <h2>Brand color</h2>
            <p>One color drives buttons, votes, glows, links, and eyebrows everywhere.</p>
            <HStack spacing={2} className="mt-2 c-theme-row">
              <Button
                variant={this.state.primaryMode === "default" ? "primary" : "tertiary"}
                size="small"
                onClick={() => this.setState({ primaryMode: "default" })}
              >
                Upvora orange (default)
              </Button>
              <Button
                variant={this.state.primaryMode === "custom" ? "primary" : "tertiary"}
                size="small"
                onClick={() => this.setState({ primaryMode: "custom" })}
              >
                Custom
              </Button>
              {this.state.primaryMode === "custom" && <ColorInput value={this.state.primary} onChange={(primary) => this.setState({ primary })} />}
            </HStack>
            <p className="text-muted mt-1">Light and dark shades are derived automatically for both appearances.</p>
          </div>

          <div>
            <h2>Default appearance</h2>
            <HStack spacing={2} className="c-theme-row">
              {["light", "dark", "system"].map((mode) => (
                <Button
                  key={mode}
                  variant={this.state.defaultTheme === mode ? "primary" : "tertiary"}
                  size="small"
                  onClick={() => this.setState({ defaultTheme: mode })}
                >
                  {mode === "system" ? "Match visitor's system" : mode.charAt(0).toUpperCase() + mode.slice(1)}
                </Button>
              ))}
            </HStack>
            <p className="text-muted mt-1">What visitors see before they pick. Their own toggle always wins.</p>
          </div>

          <div>
            <h2>Accents per function</h2>
            <p>Leave everything on Brand, or point a surface at its own color.</p>
            <VStack spacing={2} className="mt-2">
              {ACCENT_SURFACES.map((surface) => {
                const value = this.state.accents[surface.key] ?? ""
                return (
                  <HStack key={surface.key} spacing={2} align="center" className="c-theme-row">
                    <span style={{ minWidth: 140, fontWeight: 650, fontSize: 13 }}>{surface.label}</span>
                    <Button variant={value === "" ? "primary" : "tertiary"} size="small" onClick={() => this.setAccent(surface.key, "")}>
                      Brand (default)
                    </Button>
                    <Button variant={value !== "" ? "primary" : "tertiary"} size="small" onClick={() => this.setAccent(surface.key, value || "#3B82F6")}>
                      Custom
                    </Button>
                    {value !== "" && <ColorInput value={value} onChange={(hex) => this.setAccent(surface.key, hex)} />}
                    <span className="text-muted c-theme-row__hint" style={{ fontSize: 12 }}>
                      {surface.hint}
                    </span>
                  </HStack>
                )
              })}
            </VStack>
          </div>

          <div>
            <Button variant="primary" onClick={this.save} disabled={this.state.busy || !isAdmin}>
              Save theme
            </Button>
            <p className="text-muted mt-2">
              The theme writes CSS tokens on the page — the UI reads tokens, so your colors survive redesigns. Custom CSS (Advanced) still loads last and wins;
              keep it for structural tweaks. Product colors join the same system.
            </p>
          </div>
        </VStack>
      </Form>
    )
  }
}
