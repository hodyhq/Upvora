import "../../Scorecard/Scorecard.scss"

import React from "react"
import { Button, Form, Input, Toggle } from "@fider/components"
import { HStack, VStack } from "@fider/components/layout"
import { Product } from "@fider/models"
import { actions, Failure, Fider } from "@fider/services"
import { AdminBasePage } from "../components/AdminBasePage"

interface ManageProductsPageProps {
  products: Product[]
  counts: { [id: string]: number }
}

interface ManageProductsPageState {
  products: Product[]
  isAdding: boolean
  editingId: number | null
  draftName: string
  draftSlug: string
  draftDescription: string
  draftColorMode: "default" | "custom"
  draftColor: string
  draftSortOrder: number
  draftIsActive: boolean
  error?: Failure
  busy: boolean
}

const slugify = (s: string): string =>
  s
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 60)

export default class ManageProductsPage extends AdminBasePage<ManageProductsPageProps, ManageProductsPageState> {
  public bare = true
  public id = "p-admin-products"
  public name = "products"
  public title = "Products"
  public subtitle = "Separate boards and roadmaps per product — one install, one team, one sign-in"

  constructor(props: ManageProductsPageProps) {
    super(props)
    this.state = {
      products: props.products ?? [],
      isAdding: false,
      editingId: null,
      draftName: "",
      draftSlug: "",
      draftDescription: "",
      draftColorMode: "default",
      draftColor: "#38BDF8",
      draftSortOrder: 10,
      draftIsActive: true,
      busy: false,
    }
  }

  private openAdd = () => {
    this.setState({
      isAdding: true,
      editingId: null,
      draftName: "",
      draftSlug: "",
      draftDescription: "",
      draftColorMode: "default",
      draftColor: "#38BDF8",
      draftSortOrder: (this.state.products.length + 1) * 10,
      draftIsActive: true,
      error: undefined,
    })
  }

  private openEdit = (p: Product) => {
    this.setState({
      isAdding: true,
      editingId: p.id,
      draftName: p.name,
      draftSlug: p.slug,
      draftDescription: p.description,
      draftColorMode: p.color === "" ? "default" : "custom",
      draftColor: p.color || "#38BDF8",
      draftSortOrder: p.sortOrder,
      draftIsActive: p.isActive,
      error: undefined,
    })
  }

  private cancel = () => this.setState({ isAdding: false, editingId: null, error: undefined })

  private setName = (name: string) => {
    this.setState((s) => {
      const slug = s.editingId === null && (s.draftSlug === "" || s.draftSlug === slugify(s.draftName)) ? slugify(name) : s.draftSlug
      return { ...s, draftName: name, draftSlug: slug }
    })
  }

  private save = async () => {
    const s = this.state
    const color = s.draftColorMode === "default" ? "" : s.draftColor
    this.setState({ busy: true })
    if (s.editingId === null) {
      const result = await actions.createProduct({
        name: s.draftName,
        slug: s.draftSlug,
        description: s.draftDescription,
        color,
        sortOrder: s.draftSortOrder,
      })
      this.setState({ busy: false })
      if (!result.ok) {
        this.setState({ error: result.error })
        return
      }
      this.setState({
        isAdding: false,
        products: [...this.state.products, result.data as Product].sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id),
      })
    } else {
      const result = await actions.updateProduct(s.editingId, {
        name: s.draftName,
        description: s.draftDescription,
        color,
        sortOrder: s.draftSortOrder,
        isActive: s.draftIsActive,
      })
      this.setState({ busy: false })
      if (!result.ok) {
        this.setState({ error: result.error })
        return
      }
      this.setState({
        isAdding: false,
        editingId: null,
        products: this.state.products
          .map((p) =>
            p.id === s.editingId
              ? { ...p, name: s.draftName, description: s.draftDescription, color, sortOrder: s.draftSortOrder, isActive: s.draftIsActive }
              : p
          )
          .sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id),
      })
    }
  }

  private toggleActive = async (p: Product) => {
    const result = await actions.updateProduct(p.id, {
      name: p.name,
      description: p.description,
      color: p.color,
      sortOrder: p.sortOrder,
      isActive: !p.isActive,
    })
    if (result.ok) {
      this.setState({ products: this.state.products.map((x) => (x.id === p.id ? { ...x, isActive: !p.isActive } : x)) })
    }
  }

  private remove = async (p: Product) => {
    const ideas = this.props.counts?.[String(p.id)] ?? 0
    const prompt =
      ideas > 0
        ? `Delete "${p.name}"? Its ${ideas} idea${ideas === 1 ? "" : "s"} won't be deleted — they move to General (unassigned).`
        : `Delete "${p.name}"? It has no ideas.`
    if (!window.confirm(prompt)) return
    const result = await actions.deleteProduct(p.id)
    if (result.ok) {
      this.setState({ products: this.state.products.filter((x) => x.id !== p.id) })
    } else {
      this.setState({ error: result.error })
    }
  }

  public content() {
    return (
      <VStack spacing={4}>
        {this.state.isAdding && this.renderForm()}

        <div className="c-scorecard__panel">
          <div className="c-scorecard__toolbar">
            <div className="c-scorecard__group-head" style={{ marginBottom: 0, paddingBottom: 9 }}>
              <span className="c-scorecard__group-label">Product catalogue</span>
              <span className="c-scorecard__group-hint">
                Each product gets its own board and roadmap at /p/&lt;slug&gt; — statuses, tags, and members stay shared
              </span>
            </div>
            <div className="c-scorecard__toolbar-right">
              {!this.state.isAdding && (
                <Button variant="primary" onClick={this.openAdd}>
                  + Add a product
                </Button>
              )}
            </div>
          </div>
          <div className="c-scorecard__table-wrap">
            <table className="c-scorecard__table c-scorecard__table--static c-status-table text-sm">
              <thead>
                <tr>
                  <th>Product</th>
                  <th>Slug</th>
                  <th>Ideas</th>
                  <th>Color</th>
                  <th>Active</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {this.state.products.map((p) => (
                  <tr key={p.id} style={p.isActive ? undefined : { opacity: 0.55 }}>
                    <td className="c-status-table__labelcell">
                      <span className="c-status-table__pill" style={{ "--st": p.color || "var(--colors-primary-base)" } as React.CSSProperties}>
                        <span className="c-status-table__dot" />
                        {p.name}
                      </span>
                    </td>
                    <td>
                      <code>/p/{p.slug}</code>
                    </td>
                    <td className="c-status-table__flag">{this.props.counts?.[String(p.id)] ?? 0}</td>
                    <td>{p.color === "" ? <span className="c-status-table__sys">brand</span> : <code>{p.color}</code>}</td>
                    <td>
                      <Toggle active={p.isActive} onToggle={() => this.toggleActive(p)} />
                    </td>
                    <td>
                      <HStack spacing={2}>
                        <Button variant="tertiary" size="small" onClick={() => this.openEdit(p)}>
                          Edit
                        </Button>
                        <Button variant="danger" size="small" onClick={() => this.remove(p)}>
                          Delete
                        </Button>
                      </HStack>
                    </td>
                  </tr>
                ))}
                {this.state.products.length === 0 && (
                  <tr>
                    <td colSpan={6} className="text-muted" style={{ padding: 16 }}>
                      No products yet — the board works exactly like a single-product site until you add one.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>

        {(this.props.counts?.["0"] ?? 0) > 0 && this.state.products.length > 0 && (
          <p className="text-sm text-muted">
            <strong>General</strong> (unassigned) holds {this.props.counts["0"]} idea{this.props.counts["0"] === 1 ? "" : "s"} — visible on the all-products
            board.
          </p>
        )}
      </VStack>
    )
  }

  private renderForm() {
    const editing = this.state.editingId !== null
    return (
      <div className="c-scorecard__panel">
        <div className="c-scorecard__form">
          <div className="c-scorecard__form-head">
            <span className="c-scorecard__group-label">{editing ? "Edit product" : "Add a product"}</span>
            <span className="c-scorecard__group-hint">
              {editing ? `URL: /p/${this.state.draftSlug} (slug is permanent)` : "The slug becomes the public URL"}
            </span>
          </div>
          <Form error={this.state.error}>
            <VStack spacing={4}>
              <Input field="name" label="Name" maxLength={60} value={this.state.draftName} onChange={this.setName} />
              {!editing && (
                <Input
                  field="slug"
                  label="Slug (public URL: /p/<slug>)"
                  maxLength={60}
                  value={this.state.draftSlug}
                  onChange={(v) => this.setState({ draftSlug: slugify(v) })}
                />
              )}
              <Input
                field="description"
                label="Description (optional)"
                maxLength={200}
                value={this.state.draftDescription}
                onChange={(v) => this.setState({ draftDescription: v })}
              />
              <div className="c-form-field">
                <label>Color</label>
                <HStack spacing={2}>
                  <Button
                    variant={this.state.draftColorMode === "default" ? "primary" : "tertiary"}
                    size="small"
                    onClick={() => this.setState({ draftColorMode: "default" })}
                  >
                    Default (brand)
                  </Button>
                  <Button
                    variant={this.state.draftColorMode === "custom" ? "primary" : "tertiary"}
                    size="small"
                    onClick={() => this.setState({ draftColorMode: "custom" })}
                  >
                    Custom
                  </Button>
                  {this.state.draftColorMode === "custom" && (
                    <input
                      type="color"
                      value={this.state.draftColor}
                      onChange={(e) => this.setState({ draftColor: e.target.value })}
                      style={{ width: 36, height: 32, border: 0, background: "none", cursor: "pointer" }}
                    />
                  )}
                </HStack>
              </div>
              <Input
                field="sortOrder"
                label="Sort order (lower shows first)"
                value={String(this.state.draftSortOrder)}
                onChange={(v) => this.setState({ draftSortOrder: parseInt(v || "0", 10) || 0 })}
              />
              <HStack spacing={2}>
                <Button variant="primary" onClick={this.save} disabled={this.state.busy || !Fider.session.user.isAdministrator}>
                  {editing ? "Save changes" : "Add product"}
                </Button>
                <Button variant="tertiary" onClick={this.cancel}>
                  Cancel
                </Button>
              </HStack>
            </VStack>
          </Form>
        </div>
      </div>
    )
  }
}
