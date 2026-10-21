import { get, post, del } from './request'

export interface PatToken {
  id: number
  user_id: number
  name: string
  token_prefix: string
  scopes: string[]
  last_used_at?: string
  expires_at?: string
  created_at: string
}

export const patApi = {
  list(): Promise<{ items: PatToken[] }> {
    return get<{ items: PatToken[] }>('/user/tokens')
  },

  create(data: { name: string; scopes?: string[]; expires_at?: string }): Promise<{ token: string; token_info: PatToken }> {
    return post('/user/tokens', data)
  },

  del(tokenId: number): Promise<unknown> {
    return del(`/user/tokens/${tokenId}`)
  }
}
