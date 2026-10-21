export interface BranchProtection {
  id: number
  repo_id: number
  pattern: string
  require_approvals: number
  allow_force_push: boolean
  allow_delete: boolean
  required_status_checks: string
  enforce_admins: boolean
}

export interface CreateProtectionRequest {
  pattern: string
  require_approvals?: number
  allow_force_push?: boolean
  allow_delete?: boolean
  required_status_checks?: string
  enforce_admins?: boolean
}
