import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api/auth'
import { storage } from '@/utils/storage'
import type { User, LoginRequest, RegisterRequest, AuthResponse } from '@/api/types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const hasToken = ref<boolean>(!!storage.getToken())
  const registrationEnabled = ref<boolean>(true)

  const isAuthenticated = computed(() => hasToken.value)
  const isAdmin = computed(() => !!user.value?.is_site_admin)

  async function init(): Promise<void> {
    const token = storage.getToken()
    hasToken.value = !!token
    if (token) {
      try {
        await loadUser()
      } catch {
        logout()
      }
    }
    try {
      const res = await authApi.registrationStatus()
      registrationEnabled.value = res.enabled
    } catch {
      registrationEnabled.value = false
    }
  }

  async function login(data: LoginRequest): Promise<void> {
    const res = await authApi.login(data)
    setSession(res)
  }

  async function register(data: RegisterRequest): Promise<void> {
    const res = await authApi.register(data)
    setSession(res)
  }

  async function loadUser(): Promise<User> {
    const u = await authApi.me()
    user.value = u
    return u
  }

  function logout(): void {
    storage.removeToken()
    hasToken.value = false
    user.value = null
  }

  function setSession(res: AuthResponse): void {
    storage.setToken(res.token)
    hasToken.value = true
    user.value = res.user
  }

  return {
    user,
    hasToken,
    registrationEnabled,
    isAuthenticated,
    isAdmin,
    init,
    login,
    register,
    loadUser,
    logout,
    setSession
  }
})
