<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { fetchConfigStatus, fetchMediaAdapters, fetchUserPrefs, saveUserPrefs, validateOutputDir } from '../api'
import type {
  ConfigStatus,
  CustomMediaProvider,
  MediaAdapterInfo,
  ProviderUserCreds,
  ProvidersUserCreds,
  StackSummary,
} from '../constants'
import { MEDIA_ADAPTER_OPTIONS, STUDIO_STACK } from '../constants'
import OutputDirPicker from './OutputDirPicker.vue'

const props = defineProps<{
  config: ConfigStatus | null
  embedded?: boolean
}>()

const emit = defineEmits<{
  workspaceSaved: [path: string]
  settingsSaved: []
}>()

const defaultOutputDir = ref('')
const stackProfile = ref(STUDIO_STACK)
const stacks = ref<StackSummary[]>([])
const requiredHints = ref<string[]>([])
const mediaConfigured = ref(false)
const customMediaProviders = ref<CustomMediaProvider[]>([])
const activeMediaProviderId = ref('')
const adapterOptions = ref<MediaAdapterInfo[]>(MEDIA_ADAPTER_OPTIONS)
const providers = ref<ProvidersUserCreds>({
  volcengine: {},
  dashscope: {},
  kling: {},
  gemini: {},
  openai: {},
  deepseek: {},
})
const saveStatus = ref('')

function emptyCreds(): ProviderUserCreds {
  return { api_key: '', base_url: '', access_key: '', secret_key: '', app_id: '', region: '' }
}

function initProvidersFromPrefs(p: ProvidersUserCreds | undefined, legacy?: ProviderUserCreds) {
  providers.value = {
    volcengine: { ...emptyCreds(), ...p?.volcengine, ...legacy },
    dashscope: { ...emptyCreds(), ...p?.dashscope },
    kling: { ...emptyCreds(), ...p?.kling },
    gemini: { ...emptyCreds(), ...p?.gemini },
    openai: { ...emptyCreds(), ...p?.openai },
    deepseek: { ...emptyCreds(), ...p?.deepseek },
  }
}

const selectedStack = computed(() =>
  stacks.value.find((s) => s.name === stackProfile.value),
)

function adapterInfo(adapter: string): MediaAdapterInfo | undefined {
  return adapterOptions.value.find((a) => a.id === adapter)
}

function adapterNeedsSecret(adapter: string): boolean {
  return !!adapterInfo(adapter)?.needs_secret_key
}

function adapterDefaultBase(adapter: string): string {
  return adapterInfo(adapter)?.default_base_url || ''
}

function newMediaProviderId(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID()
  }
  return `mp-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}

function addMediaProvider() {
  const adapter = adapterOptions.value[0]?.id || 'volcengine'
  const entry: CustomMediaProvider = {
    id: newMediaProviderId(),
    label: adapterInfo(adapter)?.label || '媒体供应商',
    adapter,
    api_key: '',
    secret_key: '',
    base_url: adapterDefaultBase(adapter),
    image_model: '',
    video_model: '',
  }
  customMediaProviders.value.push(entry)
  if (!activeMediaProviderId.value) {
    activeMediaProviderId.value = entry.id
  }
}

function removeMediaProvider(id: string) {
  customMediaProviders.value = customMediaProviders.value.filter((e) => e.id !== id)
  if (activeMediaProviderId.value === id) {
    activeMediaProviderId.value = customMediaProviders.value[0]?.id || ''
  }
}

function onAdapterChange(entry: CustomMediaProvider) {
  const info = adapterInfo(entry.adapter)
  if (info && (!entry.label || adapterOptions.value.some((a) => a.label === entry.label))) {
    entry.label = info.label
  }
  if (!entry.base_url?.trim() && info?.default_base_url) {
    entry.base_url = info.default_base_url
  }
}

watch(stackProfile, () => {
  const hints = selectedStack.value?.required_hints
  if (hints?.length) requiredHints.value = hints
})

async function refreshStatus() {
  try {
    const cfg = await fetchConfigStatus()
    stacks.value = cfg.available_stacks || []
    if (cfg.stack_profile) stackProfile.value = cfg.stack_profile
    requiredHints.value = cfg.required_providers || selectedStack.value?.required_hints || []
    mediaConfigured.value = cfg.media_configured
  } catch {
    // ignore
  }
}

onMounted(async () => {
  if (props.config?.available_stacks?.length) {
    stacks.value = props.config.available_stacks
  }
  try {
    const adapters = await fetchMediaAdapters()
    if (adapters.length) adapterOptions.value = adapters
  } catch {
    // use constants fallback
  }
  try {
    const prefs = await fetchUserPrefs()
    defaultOutputDir.value = prefs.workspace_dir || prefs.default_output_dir || ''
    if (prefs.stack_profile) stackProfile.value = prefs.stack_profile
    initProvidersFromPrefs(prefs.providers, prefs.volcengine)
    customMediaProviders.value = (prefs.custom_media_providers || []).map((e) => ({ ...e }))
    activeMediaProviderId.value = prefs.active_media_provider_id || customMediaProviders.value[0]?.id || ''
    if (prefs.required_providers?.length) {
      requiredHints.value = prefs.required_providers
    }
    if (prefs.media_configured != null) mediaConfigured.value = prefs.media_configured
  } catch {
    // ignore
  }
  await refreshStatus()
})

async function saveSettings() {
  saveStatus.value = ''
  const path = defaultOutputDir.value.trim()
  if (path && !props.config?.desktop_mode) {
    try {
      defaultOutputDir.value = await validateOutputDir(path, 'workspace')
    } catch (err) {
      saveStatus.value = err instanceof Error ? err.message : String(err)
      return
    }
  }
  try {
    const saved = await saveUserPrefs({
      workspace_dir: defaultOutputDir.value.trim(),
      default_output_dir: defaultOutputDir.value.trim(),
      stack_profile: stackProfile.value,
      providers: providers.value,
      volcengine: providers.value.volcengine,
      custom_media_providers: customMediaProviders.value,
      active_media_provider_id: activeMediaProviderId.value,
    })
    defaultOutputDir.value = saved.workspace_dir || saved.default_output_dir || defaultOutputDir.value
    if (saved.stack_profile) stackProfile.value = saved.stack_profile
    if (saved.providers) providers.value = saved.providers
    if (saved.custom_media_providers) {
      customMediaProviders.value = saved.custom_media_providers.map((e) => ({ ...e }))
    }
    if (saved.active_media_provider_id) {
      activeMediaProviderId.value = saved.active_media_provider_id
    }
    if (saved.required_providers?.length) {
      requiredHints.value = saved.required_providers
    }
    if (saved.media_configured != null) mediaConfigured.value = saved.media_configured
    emit('workspaceSaved', defaultOutputDir.value)
    emit('settingsSaved')
    await refreshStatus()
    saveStatus.value = '已保存'
  } catch (err) {
    saveStatus.value = err instanceof Error ? err.message : String(err)
  }
}
</script>

<template>
  <section class="view active" :class="{ embedded }">
    <header v-if="!embedded" class="topbar">
      <div>
        <h1>环境设置</h1>
        <p class="muted">选择样片/成片方案，配置媒体供应商（出图 + 视频同源）与旁白 TTS。</p>
      </div>
    </header>

    <div class="settings-grid">
      <div class="card">
        <h3>视频方案</h3>
        <label class="cred-field">
          <span class="cred-label">样片 / 成片</span>
          <select v-model="stackProfile" class="cred-input">
            <option v-for="st in stacks" :key="st.name" :value="st.name">
              {{ st.label || st.name }} — {{ st.cost_hint || st.description }}
            </option>
            <option v-if="stacks.length === 0" :value="stackProfile">{{ stackProfile }}</option>
          </select>
        </label>
        <p v-if="selectedStack" class="muted stack-hint">
          {{ selectedStack.label || '方案' }} · {{ selectedStack.cost_hint }} · 目标 ~{{
            selectedStack.target_duration_sec
          }}s · 预算/时长由方案决定，出图/视频由下方「当前供应商」决定
        </p>
      </div>
      <div class="card">
        <h3>媒体 API</h3>
        <p class="muted" :class="{ ok: mediaConfigured }">
          {{ mediaConfigured ? '当前供应商密钥已就绪' : '请配置并选定一个媒体供应商' }}
        </p>
        <ul v-if="requiredHints.length" class="hint-list muted">
          <li v-for="(h, i) in requiredHints" :key="i">{{ h }}</li>
        </ul>
      </div>
      <div class="card">
        <h3>FFmpeg</h3>
        <p class="muted">{{ config?.ffmpeg_available ? 'FFmpeg 可用' : '未检测到 FFmpeg' }}</p>
      </div>
    </div>

    <div class="card settings-volc">
      <div class="media-header">
        <h3>媒体供应商（出图 + 视频）</h3>
        <button type="button" class="secondary-btn" @click="addMediaProvider">+ 添加供应商</button>
      </div>
      <p class="muted media-intro">
        选定一个供应商作为当前出图与 i2v 同源配置；可添加多条并在其间切换。
      </p>

      <div v-if="customMediaProviders.length === 0" class="muted empty-hint">
        暂无供应商，点击「添加供应商」开始配置。
      </div>

      <article
        v-for="entry in customMediaProviders"
        :key="entry.id"
        class="media-card"
        :class="{ active: entry.id === activeMediaProviderId }"
      >
        <div class="media-card-head">
          <label class="active-radio">
            <input v-model="activeMediaProviderId" type="radio" :value="entry.id" />
            <span>{{ entry.id === activeMediaProviderId ? '当前' : '设为当前' }}</span>
          </label>
          <button type="button" class="link-btn" @click="removeMediaProvider(entry.id)">删除</button>
        </div>

        <label class="cred-field">
          <span class="cred-label">显示名称</span>
          <input v-model="entry.label" type="text" class="cred-input" placeholder="如 Sora 代理" />
        </label>

        <label class="cred-field">
          <span class="cred-label">类型</span>
          <select
            v-model="entry.adapter"
            class="cred-input"
            @change="onAdapterChange(entry)"
          >
            <option v-for="opt in adapterOptions" :key="opt.id" :value="opt.id">
              {{ opt.label }}
            </option>
          </select>
        </label>

        <label class="cred-field">
          <span class="cred-label">{{ adapterNeedsSecret(entry.adapter) ? 'Access Key' : 'API Key' }}</span>
          <input
            v-model="entry.api_key"
            type="password"
            class="cred-input"
            autocomplete="off"
          />
        </label>

        <label v-if="adapterNeedsSecret(entry.adapter)" class="cred-field">
          <span class="cred-label">Secret Key</span>
          <input
            v-model="entry.secret_key"
            type="password"
            class="cred-input"
            autocomplete="off"
          />
        </label>

        <label class="cred-field">
          <span class="cred-label">Base URL（可选）</span>
          <input
            v-model="entry.base_url"
            type="text"
            class="cred-input"
            :placeholder="adapterDefaultBase(entry.adapter) || '留空则使用默认'"
            autocomplete="off"
          />
        </label>

        <label class="cred-field">
          <span class="cred-label">出图 Model（可选）</span>
          <input v-model="entry.image_model" type="text" class="cred-input" autocomplete="off" />
        </label>

        <label class="cred-field">
          <span class="cred-label">视频 Model（可选）</span>
          <input v-model="entry.video_model" type="text" class="cred-input" autocomplete="off" />
        </label>
      </article>
    </div>

    <div class="card settings-volc">
      <h3>火山旁白 TTS（OpenSpeech）</h3>
      <p class="muted media-intro">与上方媒体供应商独立；样片/成片旁白均使用此处配置。</p>
      <label class="cred-field">
        <span class="cred-label">TTS Access Token</span>
        <input
          v-model="providers.volcengine!.access_key"
          type="password"
          class="cred-input"
          autocomplete="off"
        />
      </label>
      <label class="cred-field">
        <span class="cred-label">TTS App ID</span>
        <input v-model="providers.volcengine!.app_id" type="text" class="cred-input" autocomplete="off" />
      </label>
    </div>

    <div class="card settings-volc">
      <h3>DeepSeek（文案扩写，可选）</h3>
      <label class="cred-field">
        <span class="cred-label">API Key</span>
        <input v-model="providers.deepseek!.api_key" type="password" class="cred-input" autocomplete="off" />
      </label>
      <label class="cred-field">
        <span class="cred-label">Base URL（可选）</span>
        <input v-model="providers.deepseek!.base_url" type="text" class="cred-input" autocomplete="off" />
      </label>
    </div>

    <div class="card settings-output">
      <h3>默认工作区</h3>
      <p class="muted">新建项目时将自动填入此路径，每个项目会保存在其子文件夹中。</p>
      <OutputDirPicker v-model="defaultOutputDir" :desktop-mode="config?.desktop_mode" />
      <div class="settings-output-actions">
        <button type="button" class="primary" @click="saveSettings">保存设置</button>
        <span v-if="saveStatus" class="muted">{{ saveStatus }}</span>
      </div>
    </div>

    <div class="card help">
      <h3>说明</h3>
      <ol>
        <li v-if="config?.desktop_mode">
          API 密钥保存在本机用户目录，换电脑需重新填写；不会读取仓库内的
          <code>providers.local.yaml</code>
        </li>
        <li v-else>
          也可在 <code>config/providers.local.yaml</code> 预填密钥；设置页填写会覆盖同名字段
        </li>
        <li>样片/成片仅控制预算与时长；出图/视频由「当前媒体供应商」决定</li>
        <li>未安装 FFmpeg 时运行 <code>scripts/setup-ffmpeg.ps1</code></li>
      </ol>
    </div>
  </section>
</template>

<style scoped>
.embedded {
  padding-top: 12px;
}
.settings-volc {
  margin-bottom: 14px;
}
.media-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.media-header h3 {
  margin: 0;
}
.media-intro {
  margin: 8px 0 12px;
  font-size: 12px;
}
.empty-hint {
  padding: 12px 0;
  font-size: 13px;
}
.media-card {
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 12px 14px;
  margin-top: 12px;
  background: rgba(255, 255, 255, 0.02);
}
.media-card.active {
  border-color: var(--ok);
}
.media-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}
.active-radio {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  cursor: pointer;
}
.secondary-btn {
  background: transparent;
  color: var(--text);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 6px 12px;
  font: inherit;
  font-size: 12px;
  cursor: pointer;
}
.link-btn {
  background: none;
  border: none;
  color: var(--muted);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
  text-decoration: underline;
}
.cred-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 12px;
}
.cred-label {
  font-size: 12px;
  color: var(--muted);
}
.cred-input {
  background: #0b101a;
  color: var(--text);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 10px 12px;
  font: inherit;
  font-size: 13px;
}
.settings-output {
  margin-bottom: 14px;
}
.settings-output-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 12px;
}
.stack-hint {
  margin: 10px 0 0;
  font-size: 12px;
}
.hint-list {
  margin: 8px 0 0;
  padding-left: 18px;
  font-size: 12px;
}
.muted.ok {
  color: var(--ok);
}
</style>
