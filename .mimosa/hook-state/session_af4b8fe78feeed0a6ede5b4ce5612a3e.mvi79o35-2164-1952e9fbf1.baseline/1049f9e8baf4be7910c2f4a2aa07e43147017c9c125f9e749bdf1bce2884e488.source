import { get, post } from './request'
import type { PaginatedResponse } from '@/api/types'

export interface PullRequest {
  id: number
  repo_id: number
  number: number
  title: string
  body: string
  head_branch: string
  base_branch: string
  author_id: number
  author_username: string
  status: 'open' | 'closed' | 'merged'
  created_at: string
  updated_at: string
}

export interface PRReview {
  id: number
  pull_request_id: number
  reviewer_id: number
  reviewer_username: string
  state: 'approved' | 'changes_requested' | 'commented'
  body: string
  created_at: string
}

function prPath(owner: string, name: string, number?: number): string {
  const base = `/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/pulls`
  return number != null ? `${base}/${number}` : base
}

export const prApi = {
  list(owner: string, name: string, status?: string): Promise<PaginatedResponse<PullRequest>> {
    return get<PaginatedResponse<PullRequest>>(prPath(owner, name), { params: status ? { status } : {} })
  },

  create(
    owner: string,
    name: string,
    data: { title: string; body?: string; head_branch: string; base_branch: string }
  ): Promise<PullRequest> {
    return post<PullRequest>(prPath(owner, name), data)
  },

  get(owner: string, name: string, number: number): Promise<PullRequest> {
    return get<PullRequest>(prPath(owner, name, number))
  },

  getDiff(owner: string, name: string, number: number): Promise<{ diffs: { path: string; patch: string; status: string }[] }> {
    return get(`${prPath(owner, name, number)}/diff`)
  },

  merge(owner: string, name: string, number: number): Promise<unknown> {
    return post(`${prPath(owner, name, number)}/merge`, {})
  },

  close(owner: string, name: string, number: number): Promise<unknown> {
    return post(`${prPath(owner, name, number)}/close`, {})
  },

  listReviews(owner: string, name: string, number: number): Promise<{ items: PRReview[] }> {
    return get<{ items: PRReview[] }>(`${prPath(owner, name, number)}/reviews`)
  },

  createReview(
    owner: string,
    name: string,
    number: number,
    data: { state: string; body?: string }
  ): Promise<PRReview> {
    return post<PRReview>(`${prPath(owner, name, number)}/reviews`, data)
  }
}
