import { get, post, del } from './request'

export interface Webhook {
  id: number
  repo_id: number
  url: string
  events: string[]
  content_type: string
  secret?: string
  active: boolean
  created_at: string
}

export interface WebhookDelivery {
  id: number
  hook_id: number
  event: string
  status: string
  response_code: number
  duration_ms: number
  created_at: string
}

function hookPath(owner: string, name: string, hookId?: number): string {
  const base = `/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}/hooks`
  return hookId != null ? `${base}/${hookId}` : base
}

export const webhookApi = {
  list(owner: string, name: string): Promise<{ items: Webhook[] }> {
    return get<{ items: Webhook[] }>(hookPath(owner, name))
  },

  create(owner: string, name: string, data: { url: string; events: string[]; content_type?: string; secret?: string }): Promise<Webhook> {
    return post<Webhook>(hookPath(owner, name), data)
  },

  del(owner: string, name: string, hookId: number): Promise<unknown> {
    return del<unknown>(hookPath(owner, name, hookId))
  },

  listDeliveries(owner: string, name: string, hookId: number): Promise<{ items: WebhookDelivery[] }> {
    return get<{ items: WebhookDelivery[] }>(`${hookPath(owner, name, hookId)}/deliveries`)
  },

  redeliver(owner: string, name: string, hookId: number, deliveryId: number): Promise<unknown> {
    return post(`${hookPath(owner, name, hookId)}/deliveries/${deliveryId}`, {})
  }
}
