import { get, post, del } from './request'
import type { BranchProtection, CreateProtectionRequest } from '@/api/types'

function protectionPath(owner: string, name: string, protId?: number): string {
  const base = `/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/protection`
  return protId != null ? `${base}/${protId}` : base
}

export const protectionApi = {
  list(owner: string, name: string): Promise<{ items: BranchProtection[] }> {
    return get<{ items: BranchProtection[] }>(protectionPath(owner, name))
  },

  create(owner: string, name: string, data: CreateProtectionRequest): Promise<BranchProtection> {
    return post<BranchProtection>(protectionPath(owner, name), data)
  },

  del(owner: string, name: string, protId: number): Promise<unknown> {
    return del<unknown>(protectionPath(owner, name, protId))
  }
}
