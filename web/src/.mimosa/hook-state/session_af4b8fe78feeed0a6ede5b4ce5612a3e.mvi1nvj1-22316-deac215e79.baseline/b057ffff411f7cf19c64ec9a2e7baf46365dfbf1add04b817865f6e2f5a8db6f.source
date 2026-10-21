import { get, post, patch } from './request'
import type { Issue, Comment, CreateIssueRequest, PaginatedResponse } from '@/api/types'

function issuePath(owner: string, name: string, number?: number): string {
  const base = `/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/issues`
  return number != null ? `${base}/${number}` : base
}

export const issueApi = {
  list(
    owner: string,
    name: string,
    params?: {
      status?: string
      label?: string
      milestone?: string
      assignee?: string
      q?: string
      page?: number
      per_page?: number
    }
  ): Promise<PaginatedResponse<Issue>> {
    return get<PaginatedResponse<Issue>>(issuePath(owner, name), { params })
  },

  create(owner: string, name: string, data: CreateIssueRequest): Promise<Issue> {
    return post<Issue>(issuePath(owner, name), data)
  },

  get(owner: string, name: string, number: number): Promise<Issue> {
    return get<Issue>(issuePath(owner, name, number))
  },

  update(owner: string, name: string, number: number, data: Partial<CreateIssueRequest>): Promise<Issue> {
    return patch<Issue>(issuePath(owner, name, number), data)
  },

  close(owner: string, name: string, number: number): Promise<Issue> {
    return post<Issue>(`${issuePath(owner, name, number)}/close`, {})
  },

  reopen(owner: string, name: string, number: number): Promise<Issue> {
    return post<Issue>(`${issuePath(owner, name, number)}/reopen`, {})
  },

  listComments(owner: string, name: string, number: number): Promise<{ items: Comment[] }> {
    return get<{ items: Comment[] }>(`${issuePath(owner, name, number)}/comments`)
  },

  addComment(owner: string, name: string, number: number, body: string): Promise<Comment> {
    return post<Comment>(`${issuePath(owner, name, number)}/comments`, { body })
  }
}
