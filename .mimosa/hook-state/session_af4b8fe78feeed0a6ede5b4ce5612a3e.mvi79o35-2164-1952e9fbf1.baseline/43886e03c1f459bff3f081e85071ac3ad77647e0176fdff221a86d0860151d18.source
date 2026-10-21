import { get, post, del } from './request'
import type { Label } from '@/api/types'

function labelPath(owner: string, name: string, labelId?: number): string {
  const base = `/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/labels`
  return labelId != null ? `${base}/${labelId}` : base
}

export const labelApi = {
  list(owner: string, name: string): Promise<{ items: Label[] }> {
    return get<{ items: Label[] }>(labelPath(owner, name))
  },

  create(owner: string, name: string, data: { name: string; color?: string }): Promise<Label> {
    return post<Label>(labelPath(owner, name), data)
  },

  del(owner: string, name: string, labelId: number): Promise<unknown> {
    return del<unknown>(labelPath(owner, name, labelId))
  }
}
