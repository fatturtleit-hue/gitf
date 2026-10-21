export interface User {
  id: number
  username: string
  email: string
  is_site_admin?: boolean
  is_default_admin?: boolean
  setup_completed?: boolean
  created_at: string
}

export interface AuthResponse {
  token: string
  user: User
}

export interface RegisterRequest {
  username: string
  email: string
  password: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface APIError {
  code: string
  message: string
  details?: unknown
}

export class ApiErrorException extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly code?: string
  ) {
    super(message)
    this.name = 'ApiErrorException'
  }
}
