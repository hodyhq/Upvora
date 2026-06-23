import { PostStatus, Status, StatusKind } from "./post"
import { Tenant } from "./identity"

// resolveStatus returns the tenant's Status entry for the given slug if the
// tenant has a status catalogue and a matching active row; otherwise null.
// All callers should pair this with a fallback to the hardcoded PostStatus
// enum so pre-migration tenants (or tenants whose admin hasn't seeded a
// matching row) still render correctly.
export const resolveStatus = (tenant: Tenant | null | undefined, slug: string): Status | null => {
  if (!tenant || !tenant.statuses || tenant.statuses.length === 0) return null
  for (const s of tenant.statuses) {
    if (s.slug === slug && s.isActive) return s
  }
  return null
}

// isClosedKind reports whether a kind is one of the closed semantic kinds.
// Useful for filter UIs that group active vs. closed statuses.
export const isClosedKind = (kind: StatusKind): boolean =>
  kind === "closed-completed" || kind === "closed-declined" || kind === "duplicate"

// statusListFor returns the runtime status list to render. Prefers the
// tenant catalogue; falls back to PostStatus.All for tenants without one.
export const statusListFor = (tenant: Tenant | null | undefined): Array<{ value: string; label: string; closed: boolean; filterable: boolean }> => {
  if (tenant && tenant.statuses && tenant.statuses.length > 0) {
    return tenant.statuses
      .filter((s) => s.isActive)
      .sort((a, b) => a.sortOrder - b.sortOrder)
      .map((s) => ({ value: s.slug, label: s.label, closed: isClosedKind(s.kind), filterable: s.filterable }))
  }
  return PostStatus.All.map((p) => ({ value: p.value, label: p.title, closed: p.closed, filterable: p.filterable }))
}
