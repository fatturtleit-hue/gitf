import { get, post, put, del } from './request'
import type {
  Organization,
  OrgMember,
  Team,
  TeamMember,
  TeamRepo,
  CreateOrgRequest,
  UpdateOrgRequest,
  CreateTeamRequest,
  PaginatedResponse
} from '@/api/types'

function orgPath(name: string): string {
  return `/orgs/${encodeURIComponent(name)}`
}

export const orgApi = {
  list(): Promise<PaginatedResponse<Organization>> {
    return get<PaginatedResponse<Organization>>('/orgs')
  },

  create(data: CreateOrgRequest): Promise<Organization> {
    return post<Organization>('/orgs', data)
  },

  get(name: string): Promise<Organization> {
    return get<Organization>(orgPath(name))
  },

  update(name: string, data: UpdateOrgRequest): Promise<Organization> {
    return put<Organization>(orgPath(name), data)
  },

  delete(name: string): Promise<unknown> {
    return del<unknown>(orgPath(name))
  },

  leave(name: string): Promise<unknown> {
    return post<unknown>(`${orgPath(name)}/members/leave`)
  },

  listMembers(name: string): Promise<{ items: OrgMember[] }> {
    return get<{ items: OrgMember[] }>(`${orgPath(name)}/members`)
  },

  addMember(name: string, username: string, role?: string): Promise<OrgMember> {
    return post<OrgMember>(`${orgPath(name)}/members`, { username, role: role || 'member' })
  },

  removeMember(name: string, userId: number): Promise<unknown> {
    return del<unknown>(`${orgPath(name)}/members/${userId}`)
  },

  updateMember(name: string, userId: number, role: string): Promise<OrgMember> {
    return put<OrgMember>(`${orgPath(name)}/members/${userId}`, { role })
  },

  listTeams(name: string): Promise<{ items: Team[] }> {
    return get<{ items: Team[] }>(`${orgPath(name)}/teams`)
  },

  createTeam(name: string, data: CreateTeamRequest): Promise<Team> {
    return post<Team>(`${orgPath(name)}/teams`, data)
  },

  deleteTeam(name: string, teamId: number): Promise<unknown> {
    return del<unknown>(`${orgPath(name)}/teams/${teamId}`)
  },

  listTeamMembers(name: string, teamId: number): Promise<{ items: TeamMember[] }> {
    return get<{ items: TeamMember[] }>(`${orgPath(name)}/teams/${teamId}/members`)
  },

  addTeamMember(name: string, teamId: number, username: string, role?: string): Promise<TeamMember> {
    return post<TeamMember>(`${orgPath(name)}/teams/${teamId}/members`, { username, role })
  },

  removeTeamMember(name: string, teamId: number, userId: number): Promise<unknown> {
    return del<unknown>(`${orgPath(name)}/teams/${teamId}/members/${userId}`)
  },

  listTeamRepos(name: string, teamId: number): Promise<{ items: TeamRepo[] }> {
    return get<{ items: TeamRepo[] }>(`${orgPath(name)}/teams/${teamId}/repos`)
  },

  addTeamRepo(name: string, teamId: number, repoName: string, permission: string): Promise<TeamRepo> {
    return post<TeamRepo>(`${orgPath(name)}/teams/${teamId}/repos`, { repo_name: repoName, permission })
  },

  removeTeamRepo(name: string, teamId: number, teamRepoId: number): Promise<unknown> {
    return del<unknown>(`${orgPath(name)}/teams/${teamId}/repos/${teamRepoId}`)
  }
}
