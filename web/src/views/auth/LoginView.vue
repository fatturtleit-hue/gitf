<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ElMessage } from 'element-plus'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const form = reactive({
  username: '',
  password: ''
})

const loading = ref(false)
const error = ref('')
const hidePassword = ref(true)

async function onSubmit() {
  if (!form.username || !form.password) return
  loading.value = true
  error.value = ''

  try {
    await auth.login({ username: form.username.trim(), password: form.password.trim() })
    ElMessage.success('登录成功')
    const returnUrl = route.query.returnUrl as string
    router.push(returnUrl || '/repos')
  } catch (err: any) {
    error.value = err.message || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-box">
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
      <h1>登录 gitf</h1>
    </div>
    <p class="subtle">欢迎回来，请输入您的账号信息</p>

    <el-form :model="form" @submit.prevent="onSubmit" label-position="top" class="form">
      <el-form-item label="用户名">
        <el-input
          v-model="form.username"
          autocomplete="username"
          autofocus
          size="large"
          placeholder="请输入用户名"
        />
      </el-form-item>

      <el-form-item label="密码">
        <el-input
          v-model="form.password"
          :type="hidePassword ? 'password' : 'text'"
          autocomplete="current-password"
          size="large"
          placeholder="请输入密码"
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
        :disabled="!form.username || !form.password"
      >
        <span v-if="!loading">登 录</span>
        <span v-else>登录中…</span>
      </el-button>
    </el-form>

    <div v-if="auth.registrationEnabled" class="switch">
      还没有账号？
      <router-link to="/register">立即注册</router-link>
    </div>
  </div>
</template>

<style scoped lang="scss">
.login-box {
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

// smooth error message appearance
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
