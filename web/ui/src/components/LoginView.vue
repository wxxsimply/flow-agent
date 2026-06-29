<script setup lang="ts">
import { ref } from 'vue'
import { loginWithSMS, sendSMSCode } from '../api'

const emit = defineEmits<{ loggedIn: [] }>()

const phone = ref('')
const code = ref('')
const sending = ref(false)
const logging = ref(false)
const error = ref('')
const hint = ref('开发模式默认验证码：123456')

async function sendCode() {
  error.value = ''
  sending.value = true
  try {
    await sendSMSCode(phone.value)
    hint.value = '验证码已发送，请查收短信'
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    sending.value = false
  }
}

async function login() {
  error.value = ''
  logging.value = true
  try {
    await loginWithSMS(phone.value, code.value)
    emit('loggedIn')
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    logging.value = false
  }
}
</script>

<template>
  <div class="login-wrap">
    <div class="login-card">
      <h1>FlowAgent 登录</h1>
      <p class="muted">使用手机号验证码登录，查看历史创作记录</p>
      <label class="field">
        <span>手机号</span>
        <input v-model="phone" type="tel" maxlength="11" placeholder="11 位手机号" />
      </label>
      <label class="field">
        <span>验证码</span>
        <div class="code-row">
          <input v-model="code" type="text" maxlength="6" placeholder="6 位验证码" />
          <button class="ghost" :disabled="sending || !phone" @click="sendCode">
            {{ sending ? '发送中…' : '获取验证码' }}
          </button>
        </div>
      </label>
      <p v-if="hint" class="hint muted">{{ hint }}</p>
      <p v-if="error" class="error">{{ error }}</p>
      <button class="primary login-btn" :disabled="logging || !phone || !code" @click="login">
        {{ logging ? '登录中…' : '登录' }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.login-wrap {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg, #0f1115);
}
.login-card {
  width: min(400px, 92vw);
  padding: 2rem;
  border-radius: 12px;
  background: var(--panel, #1a1d24);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.35);
}
.login-card h1 {
  margin: 0 0 0.5rem;
  font-size: 1.5rem;
}
.field {
  display: block;
  margin-top: 1rem;
}
.field span {
  display: block;
  margin-bottom: 0.35rem;
  font-size: 0.85rem;
  color: #9aa3b2;
}
.field input {
  width: 100%;
  padding: 0.6rem 0.75rem;
  border-radius: 8px;
  border: 1px solid #333;
  background: #12151a;
  color: inherit;
}
.code-row {
  display: flex;
  gap: 0.5rem;
}
.code-row input {
  flex: 1;
}
.hint {
  margin-top: 0.75rem;
  font-size: 0.8rem;
}
.error {
  color: #f87171;
  margin-top: 0.75rem;
}
.login-btn {
  width: 100%;
  margin-top: 1.25rem;
}
</style>
