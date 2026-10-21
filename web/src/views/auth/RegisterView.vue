<script setup lang="ts">
import { reactive, ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ElMessage } from 'element-plus'

const auth = useAuthStore()
const router = useRouter()

const form = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: ''
})

const loading = ref(false)
const error = ref('')
const hidePassword = ref(true)
const hideConfirm = ref(true)

const passwordMismatch = computed(
  () => form.confirmPassword.length > 0 && form.password !== form.confirmPassword
)

const canSubmit = computed(
  () =>
    form.username.length > 0 &&
    form.email.length > 0 &&
    form.password.length >= 8 &&
    form.password === form.confirmPassword
)

async function onSubmit() {
  if (!form.username || !form.email || !form.password) return
  if (form.password !== form.confirmPassword) {
    error.value = '两次输入的密码不一致'
    return
  }
  loading.value = true
  error.value = ''

  try {
    await auth.register({
      username: form.username.trim(),
      email: form.email.trim(),
      password: form.password
    })
    ElMessage.success('注册成功')
    router.push('/repos')
  } catch (err: any) {
    error.value = err.message || '注册失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="register-box">
    <div class="brand-row">
      <div class="brand-icon">
        <svg class="brand-logo" viewBox="0 0 512 512" width="22" height="22" aria-hidden="true">
          <g transform="rotate(-45, 256, 256)">
            <path d="M 60,170 C 125,108 195,108 256,170 C 317,232 387,232 452,170"
                  stroke="#4ADE80" stroke-width="28" fill="none" stroke-linecap="round"/>
            <path d="M 50,256 C 115,194 185,194 256,256 C 327,318 397,318 462,256"
                  stroke="#22C55E" stroke-width="28" fill="none" stroke-linecap="round"/>
            <path d="M 40,342 C 105,280 175,280 256,342 C 337,404 407,404 472,342"
                  stroke="#16A34A" stroke-width="28" fill="none" stroke-linecap="round"/>
          </g>
        </svg>
      </div>
      <h1>注册 gitf 账号</h1>
    </div>
    <p class="subtle">创建账号开始管理你的代码</p>

    <el-form :model="form" @submit.prevent="onSubmit" label-position="top" class="form">
      <el-form-item label="用户名">
        <el-input
          v-model="form.username"
          autocomplete="username"
          autofocus
          size="large"
          placeholder="字母、数字、下划线、连字符"
        />
      </el-form-item>

      <el-form-item label="邮箱">
        <el-input
          v-model="form.email"
          autocomplete="email"
          size="large"
          placeholder="you@example.com"
        />
      </el-form-item>

      <el-form-item label="密码">
        <el-input
          v-model="form.password"
          :type="hidePassword ? 'password' : 'text'"
          autocomplete="new-password"
          size="large"
          placeholder="至少 8 位"
          @keyup.enter="onSubmit"
        >
          <template #suffix>
            <el-icon class="password-toggle" @click="hidePassword = !hidePassword">
              <View v-if="hidePassword" />
              <Hide v-else />
            </el-icon>
          </template>
        </el-input>
      </el-form-item>

      <el-form-item label="确认密码">
        <el-input
          v-model="form.confirmPassword"
          :type="hideConfirm ? 'password' : 'text'"
          autocomplete="new-password"
          size="large"
          placeholder="再次输入密码"
        >
          <template #suffix>
            <el-icon class="password-toggle" @click="hideConfirm = !hideConfirm">
              <View v-if="hideConfirm" />
              <Hide v-else />
            </el-icon>
          </template>
        </el-input>
        <transition name="error">
          <div v-if="passwordMismatch" class="hint hint-err">
            <el-icon><Warning /></el-icon> 两次输入的密码不一致
          </div>
        </transition>
      </el-form-item>

      <transition name="error">
        <el-alert
          v-if="error"
          :title="error"
          type="error"
          :closable="false"
          show-icon
          class="error-alert"
        />
      </transition>

      <el-button
        type="primary"
        native-type="submit"
        class="submit-btn"
        size="large"
        :loading="loading"
        :disabled="!canSubmit"
      >
        <span v-if="!loading">注 册</span>
        <span v-else>注册中…</span>
      </el-button>
    </el-form>

    <div class="switch">
      已有账号？
      <router-link to="/login">立即登录</router-link>
    </div>
  </div>
</template>

<style scoped lang="scss">
.register-box {
  .brand-row {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 4px;
  }

  .brand-icon {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 8px;
    background: var(--accent-light);
    color: var(--accent);
  }

  h1 {
    font-size: 20px;
    font-weight: 600;
    letter-spacing: -0.01em;
    margin: 0;
  }

  .subtle {
    font-size: 13px;
    color: var(--text-secondary);
    margin: 0 0 24px 42px;
  }

  .form {
    margin: 0;
  }

  :deep(.el-form-item) {
    margin-bottom: 18px;
  }

  :deep(.el-form-item__label) {
    font-size: 13px;
    font-weight: 500;
    color: var(--text);
    margin-bottom: 6px;
  }

  .hint {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    margin-top: 6px;
    line-height: 1.4;
  }

  .hint-err {
    color: var(--danger);
  }

  .error-alert {
    margin: 0 0 16px;
    border-radius: var(--radius);
  }

  .submit-btn {
    width: 100%;
    height: 40px;
    font-size: 14px;
    font-weight: 500;
    margin-top: 4px;
  }

  .password-toggle {
    cursor: pointer;
    color: var(--text-muted);
    transition: color 0.15s;
  }

  .password-toggle:hover {
    color: var(--text-secondary);
  }

  .switch {
    margin-top: 18px;
    text-align: center;
    font-size: 13px;
    color: var(--text-secondary);

    a {
      color: var(--accent);
      font-weight: 500;
      transition: color 0.15s;

      &:hover {
        color: var(--accent-hover);
      }
    }
  }
}

.error-enter-active,
.error-leave-active {
  transition: opacity 0.2s, transform 0.2s;
}
.error-enter-from,
.error-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
