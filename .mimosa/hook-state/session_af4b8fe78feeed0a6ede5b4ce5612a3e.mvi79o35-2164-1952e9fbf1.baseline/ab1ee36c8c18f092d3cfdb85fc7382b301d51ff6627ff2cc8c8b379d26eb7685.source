import { post, get } from './request'
import type { AuthResponse, LoginRequest, RegisterRequest, User } from '@/api/types'

export const authApi = {
  login(data: LoginRequest): Promise<AuthResponse> {
    return post<AuthResponse>('/auth/login', data)
  },
  register(data: RegisterRequest): Promise<AuthResponse> {
    return post<AuthResponse>('/auth/register', data)
  },
  me(): Promise<User> {
    return get<User>('/user')
  },
  registrationStatus(): Promise<{ enabled: boolean }> {
    return get<{ enabled: boolean }>('/auth/register/status')
  }
}
