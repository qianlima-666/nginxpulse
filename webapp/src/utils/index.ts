import type { WebsiteInfo } from '@/api/types';

export const formatDate = (date: Date): string => {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

export const formatTraffic = (traffic: number): string => {
  if (traffic < 1024) {
    return `${traffic.toFixed(2)} B`;
  }
  if (traffic < 1024 * 1024) {
    return `${(traffic / 1024).toFixed(2)} KB`;
  }
  if (traffic < 1024 * 1024 * 1024) {
    return `${(traffic / (1024 * 1024)).toFixed(2)} MB`;
  }
  if (traffic < 1024 * 1024 * 1024 * 1024) {
    return `${(traffic / (1024 * 1024 * 1024)).toFixed(2)} GB`;
  }
  return `${(traffic / (1024 * 1024 * 1024 * 1024)).toFixed(2)} TB`;
};

export const saveUserPreference = (key: string, value: string): void => {
  localStorage.setItem(key, value);
};

export const getUserPreference = <T extends string>(key: string, defaultValue: T): T => {
  const saved = localStorage.getItem(key);
  return (saved || defaultValue) as T;
};

export const getWebsiteDisplayName = (website: WebsiteInfo): string => {
  return website.displayName?.trim() || website.name;
};

export const getWebsiteSourceBadge = (website: WebsiteInfo, t: (key: string, params?: Record<string, string | number>) => string): string => {
  const label = website.sourceType === 'remote' ? t('common.remoteSource') : t('common.localSource');
  if (website.sourceType === 'remote' && website.sourceIds && website.sourceIds.length > 0) {
    return `${label} · ${website.sourceIds.join(', ')}`;
  }
  return label;
};

export const getWebsiteDisplayTags = (
  website: WebsiteInfo,
  t: (key: string, params?: Record<string, string | number>) => string
): string[] => {
  const sourceTag = `${t('common.logSource')}${t('common.colon')}${getWebsiteSourceBadge(website, t)}`;
  const customTags = (website.tags || []).map((tag) => tag.trim()).filter((tag) => tag.length > 0);
  return [sourceTag, ...customTags];
};

export const getWebsiteMobileLabel = (
  website: WebsiteInfo,
  t: (key: string, params?: Record<string, string | number>) => string
): string => {
  return `${getWebsiteDisplayName(website)} · ${getWebsiteSourceBadge(website, t)}`;
};

export {
  getInitialServerStatusEnabled,
  getMobileBasePathWithSlash,
  getWebBasePath,
  getWebBasePathWithSlash,
} from './base-path';
