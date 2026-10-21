import { get, post, del } from './request'
import type { Release, ReleaseAsset, CreateReleaseRequest } from '@/api/types'

function releasePath(owner: string, name: string, releaseId?: number): string {
  const base = `/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/releases`
  return releaseId != null ? `${base}/${releaseId}` : base
}

export const releaseApi = {
  list(owner: string, name: string): Promise<{ items: Release[] }> {
    return get<{ items: Release[] }>(releasePath(owner, name))
  },

  create(owner: string, name: string, data: CreateReleaseRequest): Promise<Release> {
    return post<Release>(releasePath(owner, name), data)
  },

  get(owner: string, name: string, releaseId: number): Promise<Release> {
    return get<Release>(releasePath(owner, name, releaseId))
  },

  del(owner: string, name: string, releaseId: number): Promise<unknown> {
    return del<unknown>(releasePath(owner, name, releaseId))
  },

  uploadAsset(owner: string, name: string, releaseId: number, file: File): Promise<ReleaseAsset> {
    const form = new FormData()
    form.append('file', file)
    return post<ReleaseAsset>(`${releasePath(owner, name, releaseId)}/assets`, form, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  }
}
