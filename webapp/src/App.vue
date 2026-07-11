<template>
  <div class="app-shell" :class="{ 'setup-shell': setupRequired }">
    <SetupPage v-if="setupRequired" />
    <template v-else>
      <aside v-if="!hideSidebar" class="sidebar">
        <a
          class="brand"
          href="https://github.com/qianlima-666/nginxpulse/"
          target="_blank"
          rel="noopener noreferrer"
          aria-label="Open NginxPulse GitHub repository"
        >
          <div class="brand-mark" aria-hidden="true">
            <span class="brand-initials">NP</span>
            <svg class="brand-pulse" viewBox="0 0 32 16" role="presentation" aria-hidden="true">
              <path
                d="M1 8H7L10 3L14 13L18 8H31"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              ></path>
            </svg>
          </div>
          <div class="brand-text">
            <div class="brand-title">NginxPulse</div>
            <div class="brand-sub">{{ t('app.brand.subtitle') }}</div>
          </div>
        </a>
      <nav class="menu">
        <RouterLink to="/" class="menu-item" :class="{ active: isActive('/') }">{{ t('app.menu.overview') }}</RouterLink>
        <RouterLink to="/daily" class="menu-item" :class="{ active: isActive('/daily') }">{{ t('app.menu.daily') }}</RouterLink>
        <RouterLink to="/realtime" class="menu-item" :class="{ active: isActive('/realtime') }">{{ t('app.menu.realtime') }}</RouterLink>
        <RouterLink to="/logs" class="menu-item" :class="{ active: isActive('/logs') }">{{ t('app.menu.logs') }}</RouterLink>
        <RouterLink to="/settings" class="menu-item" :class="{ active: isActive('/settings') }">{{ t('app.menu.setup') }}</RouterLink>
      </nav>
      <div class="sidebar-language-compact" role="group" :aria-label="t('app.sidebar.language')" :key="currentLocale">
        <button
          v-for="option in languageOptions"
          :key="option.value"
          class="sidebar-language-btn"
          :class="{ active: option.value === currentLocale }"
          type="button"
          :aria-pressed="option.value === currentLocale"
          :aria-label="option.label"
          @click="currentLocale = option.value"
        >
          <i :class="['language-icon', option.icon]" aria-hidden="true"></i>
          <span>{{ option.shortLabel }}</span>
        </button>
      </div>
      <div class="sidebar-footer">
        <template v-if="isActive('/')">
          <div class="sidebar-label">{{ t('app.sidebar.recentActive') }}</div>
          <div class="sidebar-metric">
            <div class="sidebar-metric-value">{{ liveVisitorText }}</div>
            <div class="sidebar-metric-label">{{ t('app.sidebar.recentActiveHint') }}</div>
          </div>
        </template>
        <template v-else>
          <div class="sidebar-label">{{ sidebarLabel }}</div>
          <div class="sidebar-hint">{{ sidebarHint }}</div>
        </template>
        <div class="sidebar-language-toggle">
          <div class="sidebar-language-label">{{ t('app.sidebar.language') }}</div>
          <div class="sidebar-language-group" role="group" :aria-label="t('app.sidebar.language')" :key="currentLocale">
            <button
              v-for="option in languageOptions"
              :key="option.value"
              class="sidebar-language-btn"
              :class="{ active: option.value === currentLocale }"
              type="button"
              :aria-pressed="option.value === currentLocale"
              :aria-label="option.label"
              @click="currentLocale = option.value"
            >
              <i :class="['language-icon', option.icon]" aria-hidden="true"></i>
              <span>{{ option.shortLabel }}</span>
            </button>
          </div>
        </div>
        <div v-if="versionText" class="app-version">
          <span class="app-version-dot" aria-hidden="true"></span>
          <span class="app-version-current">{{ versionText }}</span>
          <a
            v-if="updateAvailable && latestVersion && latestReleaseUrl"
            class="app-version-update"
            :href="latestReleaseUrl"
            target="_blank"
            rel="noopener noreferrer"
            :title="t('app.version.viewRelease', { value: latestVersion })"
            :aria-label="t('app.version.viewRelease', { value: latestVersion })"
          >
            <i class="ri-upload-cloud-2-line" aria-hidden="true"></i>
            <span>{{ t('app.version.updateAvailable', { value: latestVersion }) }}</span>
          </a>
        </div>
      </div>
    </aside>

      <main class="main-content" :class="[mainClass, { 'parsing-lock': parsingActive }]">
        <div v-if="demoMode" class="demo-mode-banner">
          <span class="demo-mode-badge">{{ t('demo.badge') }}</span>
          <span class="demo-mode-text">
            {{ t('demo.text') }}
            <a href="https://github.com/qianlima-666/nginxpulse/" target="_blank" rel="noopener">https://github.com/qianlima-666/nginxpulse/</a>
          </span>
        </div>
        <RouterView :key="`${route.fullPath}-${currentLocale}-${accessKeyReloadToken}`" />
      </main>

      <div v-if="accessKeyRequired" class="access-gate">
        <div class="access-card">
          <div class="access-title">{{ t('access.title') }}</div>
          <div class="access-sub">{{ t('access.subtitle') }}</div>
          <form class="access-form" @submit.prevent="submitAccessKey">
            <input
              v-model="accessKeyInput"
              class="access-input"
              type="password"
              autocomplete="current-password"
              :placeholder="t('access.placeholder')"
            />
            <button class="access-submit" type="submit" :disabled="accessKeySubmitting">
              {{ accessKeySubmitting ? t('access.submitting') : t('access.submit') }}
            </button>
          </form>
          <div v-if="accessKeyErrorMessage" class="access-error">{{ accessKeyErrorMessage }}</div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue';
import { RouterLink, RouterView, useRoute } from 'vue-router';
import { usePrimeVue } from 'primevue/config';
import { useI18n } from 'vue-i18n';
import { fetchAppStatus, fetchVersionInfo } from '@/api';
import { ACCESS_KEY_STORAGE, saveAccessKey, setAccessKeyExpireDays } from '@/api/client';
import { getLocaleFromQuery, getStoredLocale, normalizeLocale, setLocale } from '@/i18n';
import { primevueLocales } from '@/i18n/primevue';
import SetupPage from '@/pages/SetupPage.vue';

const route = useRoute();
const primevue = usePrimeVue();
const { t, n, locale } = useI18n({ useScope: 'global' });

const ACCESS_KEY_EVENT = 'nginxpulse:access-key-required';

const sidebarLabel = computed(() => {
  const key = route.meta.sidebarLabelKey as string;
  return key ? t(key) : '';
});
const sidebarHint = computed(() => {
  const key = route.meta.sidebarHintKey as string;
  return key ? t(key) : '';
});
const mainClass = computed(() => (route.meta.mainClass as string) || '');
const hideSidebar = computed(() => shouldHideSidebar(route.query));

const isActive = (path: string) => route.path === path;

const isDark = ref(localStorage.getItem('darkMode') === 'true');
const parsingActive = ref(false);
const liveVisitorCount = ref<number | null>(null);
const demoMode = ref(false);
const migrationRequired = ref(false);
const setupRequired = ref(false);
const appVersion = ref('');
const accessKeyRequired = ref(false);
const accessKeySubmitting = ref(false);
const accessKeyInput = ref(localStorage.getItem(ACCESS_KEY_STORAGE) || '');
const accessKeyErrorKey = ref<string | null>(null);
const accessKeyErrorText = ref('');
const accessKeyReloadToken = ref(0);
const latestVersion = ref('');
const latestReleaseUrl = ref('');
const updateAvailable = ref(false);

const languageOptions = computed(() => {
  const _locale = locale.value;
  return [
    { value: 'zh-CN', label: t('language.zh'), shortLabel: t('language.zhShort'), icon: 'ri-translate-2' },
    { value: 'en-US', label: t('language.en'), shortLabel: t('language.enShort'), icon: 'ri-global-line' },
  ];
});

const currentLocale = computed({
  get: () => normalizeLocale(locale.value),
  set: (value: string) => setLocale(normalizeLocale(value)),
});

const previewUpdateQuery = computed(() => {
  const value = route.query.previewUpdate;
  if (Array.isArray(value)) {
    return value[0];
  }
  return value;
});

const previewVersionQuery = computed(() => {
  const value = route.query.previewVersion;
  if (Array.isArray(value)) {
    return value[0];
  }
  return value;
});

const applyTheme = (value: boolean) => {
  if (value) {
    document.body.classList.add('dark-mode');
    document.documentElement.classList.add('dark-mode');
    localStorage.setItem('darkMode', 'true');
  } else {
    document.body.classList.remove('dark-mode');
    document.documentElement.classList.remove('dark-mode');
    localStorage.setItem('darkMode', 'false');
  }
};

const toggleTheme = () => {
  isDark.value = !isDark.value;
};

onMounted(() => {
  applyTheme(isDark.value);
  refreshAppStatus();
  window.addEventListener(ACCESS_KEY_EVENT, handleAccessKeyEvent);
});

onBeforeUnmount(() => {
  window.removeEventListener(ACCESS_KEY_EVENT, handleAccessKeyEvent);
});

watch(isDark, (value) => {
  applyTheme(value);
});

watch(locale, (value) => {
  const normalized = normalizeLocale(value);
  primevue.config.locale = primevueLocales[normalized];
});

watch([previewUpdateQuery, previewVersionQuery], () => {
  void refreshVersionInfo();
});

provide('theme', {
  isDark,
  toggle: toggleTheme,
});

provide('setParsingActive', (value: boolean) => {
  parsingActive.value = value;
});

provide('setLiveVisitorCount', (value: number | null) => {
  liveVisitorCount.value = value;
});

provide('demoMode', demoMode);
provide('migrationRequired', migrationRequired);

async function refreshAppStatus() {
  try {
    const status = await fetchAppStatus();
    demoMode.value = Boolean(status.demo_mode);
    migrationRequired.value = Boolean(status.migration_required);
    setupRequired.value = Boolean(status.setup_required);
    setAccessKeyExpireDays(status.access_key_expire_days);
    appVersion.value = status.version ?? '';
    accessKeyRequired.value = false;
    accessKeyErrorKey.value = null;
    accessKeyErrorText.value = '';
    const hasStoredLocale = getStoredLocale() !== null;
    const hasQueryLocale = getLocaleFromQuery() !== null;
    if (!hasStoredLocale && !hasQueryLocale && status.language) {
      setLocale(normalizeLocale(status.language), false);
    }
    void refreshVersionInfo();
  } catch (error) {
    const message = error instanceof Error ? error.message : t('common.requestFailed');
    if (message.toLowerCase().includes('key') || message.includes('密钥')) {
      accessKeyRequired.value = true;
      setAccessKeyErrorMessage(message);
    } else {
      console.error('获取系统状态失败:', error);
    }
  }
}

async function refreshVersionInfo() {
  if (!appVersion.value) {
    latestVersion.value = '';
    latestReleaseUrl.value = '';
    updateAvailable.value = false;
    return;
  }

  const previewVersion = resolvePreviewVersion();
  if (previewVersion) {
    latestVersion.value = previewVersion;
    latestReleaseUrl.value = `https://github.com/qianlima-666/nginxpulse/releases/tag/${previewVersion}`;
    updateAvailable.value = true;
    return;
  }

  try {
    const info = await fetchVersionInfo();
    latestVersion.value = info.latest_version ?? '';
    latestReleaseUrl.value = info.latest_release_url ?? '';
    updateAvailable.value = Boolean(info.update_available && info.latest_version && info.latest_release_url);
  } catch (error) {
    console.warn('获取版本更新信息失败:', error);
  }
}

function resolvePreviewVersion() {
  const raw = String(previewUpdateQuery.value ?? '').trim().toLowerCase();
  if (!raw || ['0', 'false', 'no', 'off'].includes(raw)) {
    return '';
  }

  const previewVersion = String(previewVersionQuery.value ?? '').trim();
  if (previewVersion) {
    return previewVersion.startsWith('v') ? previewVersion : `v${previewVersion}`;
  }

  return 'v1.6.18';
}

function handleAccessKeyEvent(event: Event) {
  const detail = (event as CustomEvent<{ message?: string }>).detail;
  accessKeyRequired.value = true;
  setAccessKeyErrorMessage(detail?.message || '');
}

function shouldHideSidebar(query: Record<string, unknown>) {
  const truthy = new Set(['1', 'true', 'yes', 'on']);
  const falsy = new Set(['0', 'false', 'no', 'off', 'hide']);
  const pick = (value: unknown) => {
    if (Array.isArray(value)) {
      return value[0];
    }
    return value;
  };

  const hideSidebar = pick(query.hideSidebar);
  if (hideSidebar !== undefined) {
    return truthy.has(String(hideSidebar).toLowerCase());
  }

  const embed = pick(query.embed);
  if (embed !== undefined) {
    return truthy.has(String(embed).toLowerCase());
  }

  const sidebar = pick(query.sidebar);
  if (sidebar !== undefined) {
    return falsy.has(String(sidebar).toLowerCase());
  }

  return false;
}

async function submitAccessKey() {
  const value = accessKeyInput.value.trim();
  if (!value) {
    accessKeyErrorKey.value = 'access.required';
    accessKeyErrorText.value = '';
    return;
  }
  accessKeySubmitting.value = true;
  saveAccessKey(value);
  try {
    await refreshAppStatus();
    if (!accessKeyRequired.value) {
      accessKeyReloadToken.value += 1;
    }
  } finally {
    accessKeySubmitting.value = false;
  }
}

function setAccessKeyErrorMessage(message: string) {
  const normalized = message.trim().toLowerCase();
  if (!message || normalized.includes('需要访问密钥') || normalized.includes('access key required')) {
    accessKeyErrorKey.value = 'access.title';
    accessKeyErrorText.value = '';
    return;
  }
  if (normalized.includes('访问密钥无效') || normalized.includes('invalid')) {
    accessKeyErrorKey.value = 'access.invalid';
    accessKeyErrorText.value = '';
    return;
  }
  if (normalized.includes('过期') || normalized.includes('expired')) {
    accessKeyErrorKey.value = 'access.expired';
    accessKeyErrorText.value = '';
    return;
  }
  accessKeyErrorKey.value = null;
  accessKeyErrorText.value = message;
}

const liveVisitorText = computed(() =>
  Number.isFinite(liveVisitorCount.value ?? NaN)
    ? n(liveVisitorCount.value as number)
    : '--'
);

const versionText = computed(() => appVersion.value || '');
const accessKeyErrorMessage = computed(() => {
  if (accessKeyErrorKey.value) {
    return t(accessKeyErrorKey.value);
  }
  return accessKeyErrorText.value;
});
</script>

<style lang="scss" scoped>
.demo-mode-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  margin-bottom: 16px;
  border-radius: var(--radius-md);
  border: 1px solid rgba(239, 68, 68, 0.2);
  background: rgba(239, 68, 68, 0.08);
  color: #991b1b;
  font-size: 13px;
  font-weight: 500;
  box-shadow: var(--shadow-soft);
}

.demo-mode-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border-radius: var(--radius-pill);
  background: rgba(239, 68, 68, 0.14);
  color: #b91c1c;
  font-weight: 700;
  font-size: 12px;
  letter-spacing: 0.4px;
}

.demo-mode-text {
  color: inherit;
  line-height: 1.5;
}

.demo-mode-text a {
  color: inherit;
  text-decoration: underline;
  text-underline-offset: 3px;
}

.access-gate {
  position: fixed;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 24px;
  background: rgba(15, 23, 42, 0.35);
  backdrop-filter: blur(10px);
  z-index: 50;
}

.access-card {
  width: min(420px, 100%);
  background: var(--panel);
  border-radius: var(--radius-xl);
  border: 1px solid var(--border);
  box-shadow: var(--shadow);
  padding: 28px;
  text-align: center;
}

.access-title {
  font-size: 20px;
  font-weight: 700;
  margin-bottom: 6px;
}

.access-sub {
  font-size: 13px;
  color: var(--muted);
  margin-bottom: 18px;
}

.access-form {
  display: grid;
  gap: 12px;
}

.access-input {
  width: 100%;
  padding: 12px 14px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  background: var(--input-bg);
  color: var(--text);
  font-size: 14px;
  outline: none;
}

.access-input:focus {
  border-color: rgba(var(--primary-color-rgb), 0.6);
  box-shadow: 0 0 0 3px rgba(var(--primary-color-rgb), 0.15);
}

.access-submit {
  border: none;
  border-radius: var(--radius-md);
  padding: 12px 14px;
  font-size: 14px;
  font-weight: 600;
  color: #fff;
  background: linear-gradient(135deg, var(--primary) 0%, var(--primary-strong) 100%);
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  box-shadow: var(--shadow-soft);
}

.access-submit:hover {
  transform: translateY(-1px);
}

.access-submit:disabled {
  cursor: default;
  opacity: 0.75;
  transform: none;
}

.access-error {
  margin-top: 12px;
  font-size: 12px;
  color: var(--error-color);
}

.app-version {
  margin-top: 14px;
  font-size: 11px;
  color: var(--muted);
  letter-spacing: 0.02em;
  display: flex;
  gap: 6px;
  align-items: center;
  flex-wrap: wrap;
}

.app-version-current {
  color: var(--text);
  font-weight: 600;
}

.app-version-dot {
  width: 6px;
  height: 6px;
  border-radius: var(--radius-pill);
  background: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-soft);
}

.app-version-update {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: var(--radius-pill);
  border: 1px solid rgba(var(--primary-color-rgb), 0.22);
  background: linear-gradient(135deg, rgba(var(--primary-color-rgb), 0.1), rgba(var(--primary-color-rgb), 0.04));
  color: var(--primary);
  font-size: 10px;
  font-weight: 700;
  text-decoration: none;
  box-shadow: 0 10px 18px rgba(var(--primary-color-rgb), 0.12);
  transition: border-color 0.2s ease, background 0.2s ease, box-shadow 0.2s ease, color 0.2s ease;
}

.app-version-update:hover {
  border-color: rgba(var(--primary-color-rgb), 0.34);
  background: linear-gradient(135deg, rgba(var(--primary-color-rgb), 0.16), rgba(var(--primary-color-rgb), 0.07));
  box-shadow: 0 12px 22px rgba(var(--primary-color-rgb), 0.16);
}

.app-version-update:focus-visible {
  outline: 2px solid rgba(var(--primary-color-rgb), 0.28);
  outline-offset: 2px;
}

.app-version-update i {
  font-size: 12px;
}

:global(body.dark-mode) .app-version-current {
  color: #dbe5f4;
}

:global(body.dark-mode) .app-version-update {
  border-color: rgba(90, 162, 255, 0.28);
  background: linear-gradient(135deg, rgba(90, 162, 255, 0.18), rgba(90, 162, 255, 0.08));
  color: #91c3ff;
  box-shadow: 0 12px 24px rgba(15, 23, 42, 0.3);
}

:global(body.dark-mode) .app-version-update:hover {
  border-color: rgba(90, 162, 255, 0.42);
  background: linear-gradient(135deg, rgba(90, 162, 255, 0.26), rgba(90, 162, 255, 0.12));
}
</style>
