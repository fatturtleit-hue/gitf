import axios, { AxiosError, type AxiosRequestConfig } from 'axios'
import { storage } from '@/utils/storage'
import { ApiErrorException, type APIError } from '@/api/types'
import router from '@/router'

const PUBLIC_AUTH_PATHS = ['/auth/login', '/auth/register']

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

request.interceptors.request.use(config => {
  const token = storage.getToken()
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

request.interceptors.response.use(
  response => response,
  (error: AxiosError<APIError>) => {
    const status = error.response?.status
    const payload = error.response?.data
    const message = payload?.message ?? error.message ?? '未知错误'
    const code = payload?.code

    if (status === 401 && error.config?.url && !PUBLIC_AUTH_PATHS.some(p => error.config!.url!.includes(p))) {
      storage.removeToken()
      const returnUrl = window.location.pathname
      router.push({ path: '/login', query: { returnUrl } })
    }

    return Promise.reject(new ApiErrorException(message, status ?? 0, code))
  }
)

export async function get<T>(path: string, config?: AxiosRequestConfig): Promise<T> {
  const res = await request.get<T>(path, config)
  return res.data
}

export async function post<T>(path: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  const res = await request.post<T>(path, data, config)
  return res.data
}

export async function put<T>(path: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  const res = await request.put<T>(path, data, config)
  return res.data
}

export async function patch<T>(path: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
  const res = await request.patch<T>(path, data, config)
  return res.data
}

export async function del<T>(path: string, config?: AxiosRequestConfig): Promise<T> {
  const res = await request.delete<T>(path, config)
  return res.data
}

export async function getText(path: string, config?: AxiosRequestConfig): Promise<string> {
  const res = await request.get<string>(path, { ...config, responseType: 'text' })
  return res.data
}

export default request
