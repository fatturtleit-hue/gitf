import { get, post, patch, del } from './request'
import type { User, PaginatedResponse } from '@/api/types'

export const userApi = {
  list(): Promise<PaginatedResponse<User>> {
    return get<PaginatedResponse<User>>('/admin/users')
  },

  create(data: { username: string; email: string; password: string; is_admin?: boolean }): Promise<User> {
    return post<User>('/admin/users', data)
  },

  get(id: number): Promise<User> {
    return get<User>(`/admin/users/${id}`)
  },

  update(id: number, data: Partial<{ username?: string; email?: string; password?: string; is_admin?: boolean }>): Promise<User> {
    return patch<User>(`/admin/users/${id}`, data)
  },

  del(id: number): Promise<unknown> {
    return del<unknown>(`/admin/users/${id}`)
  }
}
