import axios from 'axios';
import { i18n } from '@/i18n';
import { getWebBasePathWithSlash } from '@/utils';

export const ACCESS_KEY_STORAGE = 'nginxpulse_access_key';
export const SETUP_PASSWORD_STORAGE = 'nginxpulse_setup_password';
const ACCESS_KEY_ISSUED_AT_STORAGE = 'nginxpulse_access_key_issued_at';
const ACCESS_KEY_EXPIRE_DAYS_STORAGE = 'nginxpulse_access_key_expire_days';
const ACCESS_KEY_HEADER = 'X-NginxPulse-Key';
const SETUP_PASSWORD_HEADER = 'X-NginxPulse-Setup-Password';
const ACCESS_KEY_EVENT = 'nginxpulse:access-key-required';
const DEFAULT_ACCESS_KEY_EXPIRE_DAYS = 7;

let accessKeyExpireDays = DEFAULT_ACCESS_KEY_EXPIRE_DAYS;
const savedExpireDays = Number(localStorage.getItem(ACCESS_KEY_EXPIRE_DAYS_STORAGE));
if (Number.isFinite(savedExpireDays) && savedExpireDays > 0) {
  accessKeyExpireDays = Math.floor(savedExpireDays);
}

export function setAccessKeyExpireDays(days?: number | null) {
  if (typeof days === 'number' && Number.isFinite(days) && days > 0) {
    accessKeyExpireDays = Math.floor(days);
    localStorage.setItem(ACCESS_KEY_EXPIRE_DAYS_STORAGE, String(accessKeyExpireDays));
    return;
  }
  accessKeyExpireDays = DEFAULT_ACCESS_KEY_EXPIRE_DAYS;
  localStorage.setItem(ACCESS_KEY_EXPIRE_DAYS_STORAGE, String(accessKeyExpireDays));
}

export function saveAccessKey(value: string) {
  const normalized = value.trim();
  if (!normalized) {
    clearAccessKey();
    return;
  }
  localStorage.setItem(ACCESS_KEY_STORAGE, normalized);
  localStorage.setItem(ACCESS_KEY_ISSUED_AT_STORAGE, String(Date.now()));
}

export function clearAccessKey() {
  localStorage.removeItem(ACCESS_KEY_STORAGE);
  localStorage.removeItem(ACCESS_KEY_ISSUED_AT_STORAGE);
}

export function saveSetupPassword(value: string) {
  const normalized = value.trim();
  if (!normalized) {
    localStorage.removeItem(SETUP_PASSWORD_STORAGE);
    return;
  }
  localStorage.setItem(SETUP_PASSWORD_STORAGE, normalized);
}

function ensureAccessKeyIssuedAt() {
  const raw = localStorage.getItem(ACCESS_KEY_ISSUED_AT_STORAGE);
  const issuedAt = Number(raw);
  if (Number.isFinite(issuedAt) && issuedAt > 0) {
    return issuedAt;
  }
  const now = Date.now();
  localStorage.setItem(ACCESS_KEY_ISSUED_AT_STORAGE, String(now));
  return now;
}

function isAccessKeyExpired() {
  const issuedAt = ensureAccessKeyIssuedAt();
  const maxAgeMs = accessKeyExpireDays * 24 * 60 * 60 * 1000;
  return Date.now() >= issuedAt + maxAgeMs;
}

const client = axios.create({
  baseURL: getWebBasePathWithSlash(),
  timeout: 15000,
  headers: {
    'X-Requested-With': 'XMLHttpRequest',
  },
});

client.interceptors.request.use((config) => {
  const accessKey = localStorage.getItem(ACCESS_KEY_STORAGE);
  if (accessKey) {
    if (isAccessKeyExpired()) {
      clearAccessKey();
      window.dispatchEvent(
        new CustomEvent(ACCESS_KEY_EVENT, {
          detail: { message: i18n.global.t('access.expired') },
        })
      );
      return config;
    }
    config.headers[ACCESS_KEY_HEADER] = accessKey;
  }
  const setupPassword = localStorage.getItem(SETUP_PASSWORD_STORAGE);
  if (setupPassword) {
    config.headers[SETUP_PASSWORD_HEADER] = setupPassword;
  }
  return config;
});

client.interceptors.response.use(
  (response) => response,
  (error) => {
    const status = error?.response?.status;
    const fallback = i18n.global.t('common.requestFailed');
    const message = error?.response?.data?.error || error?.message || fallback;
    if (status === 401) {
      const normalized = String(message).toLowerCase();
      const eventName = normalized.includes('系统配置密码') || normalized.includes('settings password')
        ? 'nginxpulse:setup-password-required'
        : ACCESS_KEY_EVENT;
      window.dispatchEvent(
        new CustomEvent(eventName, {
          detail: { message },
        })
      );
    }
    return Promise.reject(new Error(message));
  }
);

export default client;
