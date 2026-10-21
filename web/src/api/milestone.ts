import { get, post } from './request'
import type { Milestone } from '@/api/types'

function milestonePath(owner: string, name: string): string {
  return `/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/milestones`
}

export const milestoneApi = {
  list(owner: string, name: string): Promise<{ items: Milestone[] }> {
    return get<{ items: Milestone[] }>(milestonePath(owner, name))
  },

  create(owner: string, name: string, data: { title: string; description?: string; due_date?: string }): Promise<Milestone> {
    return post<Milestone>(milestonePath(owner, name), data)
  },

  close(owner: string, name: string, title: string): Promise<Milestone> {
    return post<Milestone>(`${milestonePath(owner, name)}/close`, { title })
  }
}
