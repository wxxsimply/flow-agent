<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import AppSidebar from './components/AppSidebar.vue'
import DesktopTitlebar from './components/DesktopTitlebar.vue'
import HistoryRail from './components/HistoryRail.vue'
import SettingsDrawer from './components/SettingsDrawer.vue'
import StudioView from './components/StudioView.vue'
import LoginView from './components/LoginView.vue'
import { fetchAuthStatus, fetchConfigStatus, logout } from './api'
import { getToken } from './auth'
import { useStudioRunState } from './composables/useStudioRunState'
import { useStudioCreativeOptions } from './composables/useStudioCreativeOptions'
import type { ConfigStatus, UserInfo } from './constants'
import { stackProfileLabel } from './constants'

const config = ref<ConfigStatus | null>(null)
const authEnabled = ref(false)
const loggedIn = ref(true)
const user = ref<UserInfo | null>(null)
const studioLoadRunId = ref<string | null>(null)
const settingsOpen = ref(false)
const studioRef = ref<InstanceType<typeof StudioView> | null>(null)

const { currentRunId } = useStudioRunState()
const { initOptions, refreshFromConfig } = useStudioCreativeOptions()
const desktopWindowClass = ref(false)

const envBadge = computed(() => {
  if (!config.value) return { text: '检测环境中…', className: 'badge' }
  const stackName = config.value.stack_profile || config.value.stack_default || 'micro-movie-cap5'
  const tierLabel = stackProfileLabel(stackName, config.value.available_stacks)
  const vendor = config.value.active_media_provider_label
  if (config.value.media_configured && config.value.ffmpeg_available) {
    const mode = config.value.desktop_mode ? '桌面' : 'Web'
    const vendorPart = vendor ? ` · ${vendor}` : ''
    return {
      text: `${mode} · ${tierLabel}${vendorPart} · 就绪`,
      className: 'badge ok',
    }
  }
  const vendorHint = vendor ? ` · ${vendor}` : ''
  return { text: `${tierLabel}${vendorHint} · 需配置 API / FFmpeg`, className: 'badge warn' }
})

async function refreshConfig() {
  try {
    config.value = await fetchConfigStatus()
    if (config.value.auth_enabled) authEnabled.value = true
    desktopWindowClass.value = !!config.value.desktop_window_controls
    await refreshFromConfig()
    if (
      config.value.desktop_mode &&
      !config.value.media_configured &&
      !settingsOpen.value
    ) {
      settingsOpen.value = true
    }
  } catch {
    config.value = null
  }
}

function onSettingsSaved() {
  void refreshConfig()
}

const outputDirLocked = computed(() => !!currentRunId.value)

async function refreshAuth() {
  try {
    const st = await fetchAuthStatus()
    authEnabled.value = st.auth_enabled
    if (!st.auth_enabled) {
      loggedIn.value = true
      return
    }
    loggedIn.value = st.logged_in || !!getToken()
  } catch {
    authEnabled.value = false
    loggedIn.value = true
  }
}

function onLoggedIn() {
  loggedIn.value = true
}

function onLogout() {
  logout()
  loggedIn.value = false
  user.value = null
}

function onOpenRun(runId: string) {
  if (studioLoadRunId.value === runId) {
    studioLoadRunId.value = null
    void nextTick(() => {
      studioLoadRunId.value = runId
    })
    return
  }
  studioLoadRunId.value = runId
}

function onRunDeleted(runId: string) {
  if (currentRunId.value === runId) {
    studioRef.value?.startNewProject()
  }
  studioLoadRunId.value = null
}

onMounted(async () => {
  await initOptions()
  await refreshAuth()
  await refreshConfig()
})
</script>

<template>
  <LoginView v-if="authEnabled && !loggedIn" @logged-in="onLoggedIn" />
  <div v-else class="desktop-shell" :class="{ 'desktop-shell-active': desktopWindowClass }">
    <DesktopTitlebar v-if="desktopWindowClass" />
    <div class="app studio-layout" :class="{ 'desktop-window': desktopWindowClass }">
    <HistoryRail @open-run="onOpenRun" @deleted="onRunDeleted" @open-settings="settingsOpen = true" />
    <AppSidebar
      :env-badge="envBadge"
      :auth-enabled="authEnabled"
      :output-dir-locked="outputDirLocked"
      @logout="onLogout"
    />
    <main class="main">
      <StudioView ref="studioRef" :load-run-id="studioLoadRunId" />
    </main>
    <SettingsDrawer v-model:open="settingsOpen" :config="config" @settings-saved="onSettingsSaved" />
    </div>
  </div>
</template>
