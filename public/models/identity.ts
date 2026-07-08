import { Status } from "./post"

export interface Tenant {
  id: number
  name: string
  cname: string
  subdomain: string
  locale: string
  invitation: string
  welcomeMessage: string
  welcomeHeader: string
  descriptionTemplate: string
  shareIdeaInstructions: string
  status: TenantStatus
  isPrivate: boolean
  logoBlobKey: string
  allowedSchemes: string
  isEmailAuthAllowed: boolean
  isFeedEnabled: boolean
  isModerationEnabled: boolean
  isPro: boolean
  // Custom status catalogue for this tenant; populated by the server on every
  // request. Undefined for old Fider builds / unmigrated tenants — runtime
  // code must fall back to the hardcoded PostStatus enum in that case.
  statuses?: Status[]
  // Scorecard feature toggle + band thresholds + admin-configurable field
  // catalogue. When isScorecardEnabled is false, nothing renders.
  isScorecardEnabled: boolean
  scorecardBandStrong: number
  scorecardBandGood: number
  scorecardBandRefine: number
  scorecardBandLow: number
  scorecardBandStrongLabel: string
  scorecardBandGoodLabel: string
  scorecardBandRefineLabel: string
  scorecardBandLowLabel: string
  scorecardBandNoneLabel: string
  scorecardTriggerStatusSlug: string
  scorecardFields?: ScorecardField[]
}

// One entry in a choice-typed field's list. bucket maps the choice onto a
// scorecard dashboard view (used by the header status field's tabs).
export interface ScorecardFieldChoice {
  value: string
  color?: string
  bucket?: "new" | "review" | "executive"
}

export interface ScorecardField {
  id: number
  key: string
  label: string
  groupKey: string
  type: "text" | "note" | "date" | "number" | "url" | "choice" | "score" | "user"
  choices?: ScorecardFieldChoice[]
  weight?: number
  question?: string
  sortOrder: number
  isSystem: boolean
  isActive: boolean
  createdAt: string
  updatedAt: string
}

export enum TenantStatus {
  Active = 1,
  Pending = 2,
  Locked = 3,
  Disabled = 4,
}

export interface User {
  id: number
  name: string
  email?: string
  role: UserRole
  status: UserStatus
  isTrusted: boolean
  avatarURL: string
}

export interface UserNames {
  id: number
  name: string
}

export enum UserAvatarType {
  Letter = "letter",
  Gravatar = "gravatar",
  Custom = "custom",
}

export enum UserStatus {
  Active = "active",
  Deleted = "deleted",
  Blocked = "blocked",
}

export enum UserRole {
  Visitor = "visitor",
  Collaborator = "collaborator",
  Administrator = "administrator",
}

export const isCollaborator = (role: UserRole): boolean => {
  return role === UserRole.Collaborator || role === UserRole.Administrator
}

export const requiresModeration = (user: User): boolean => {
  return user.role === UserRole.Visitor && !user.isTrusted
}

export interface CurrentUser {
  id: number
  name: string
  email: string
  avatarType: UserAvatarType
  avatarBlobKey: string
  avatarURL: string
  role: UserRole
  status: UserStatus
  isAdministrator: boolean
  isCollaborator: boolean
  isTrusted: boolean
}
