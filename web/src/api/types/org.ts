export interface Organization {
  id: number
  name: string
  display_name: string
  description: string
  created_at: string
}

export interface OrgMember {
  id: number
  user_id: number
  username: string
  email: string
  role: 'owner' | 'admin' | 'member'
  created_at: string
}

export interface Team {
  id: number
  org_id: number
  name: string
  description: string
  created_at: string
}

export interface TeamMember {
  id: number
  user_id: number
  username: string
}

export interface TeamRepo {
  id: number
  team_id: number
  team_name: string
  repo_id: number
  permission: 'read' | 'write' | 'admin'
}

export interface CreateOrgRequest {
  name: string
  display_name: string
  description: string
}

export interface UpdateOrgRequest {
  display_name: string
  description: string
}

export interface CreateTeamRequest {
  name: string
  description: string
}
