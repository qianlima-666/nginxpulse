<template>
  <div class="select-group">
    <label v-if="labelText" class="select-label" :for="selectId">{{ labelText }}</label>
    <Dropdown
      :inputId="selectId"
      v-model="selectedValue"
      class="website-dropdown"
      :options="websites"
      optionLabel="name"
      optionValue="id"
      :disabled="disabled"
      :placeholder="placeholderText"
      :loading="loading"
      :emptyMessage="emptyText"
      :filter="true"
      filterBy="name,sourceLabel,remoteSourceId,customLabel"
      :filterPlaceholder="filterPlaceholderText"
    >
      <template #option="slotProps">
        <div class="website-option">
          <span class="website-option-name">{{ slotProps.option.name }}</span>
          <div class="website-option-tags">
            <span class="website-option-tag">{{ sourceLabel(slotProps.option) }}</span>
            <span v-if="remoteSourceId(slotProps.option)" class="website-option-tag website-option-tag-muted">
              {{ remoteSourceId(slotProps.option) }}
            </span>
            <span v-if="slotProps.option.autoDiscoverHosts" class="website-option-tag website-option-tag-accent">
              {{ slotProps.option.customLabel || t('common.autoDiscoverHostsTag') }}
            </span>
          </div>
        </div>
      </template>
    </Dropdown>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import type { WebsiteInfo } from '@/api/types';

const props = withDefaults(
  defineProps<{
    modelValue: string;
    websites: WebsiteInfo[];
    label?: string;
    id?: string;
    loading?: boolean;
    loadingText?: string;
    emptyText?: string;
  }>(),
  {
    id: 'website-selector',
    loading: false,
  }
);

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void;
}>();

const { t } = useI18n({ useScope: 'global' });
const selectId = computed(() => props.id || 'website-selector');
const disabled = computed(() => props.loading || props.websites.length === 0);
const labelText = computed(() => props.label ?? t('common.website'));
const loadingText = computed(() => props.loadingText ?? t('common.loading'));
const emptyText = computed(() => props.emptyText ?? t('common.emptyWebsite'));
const filterPlaceholderText = computed(() => t('common.searchWebsite'));
const selectedValue = computed({
  get: () => props.modelValue,
  set: (value: string) => emit('update:modelValue', value),
});
const placeholderText = computed(() => {
  if (props.loading) {
    return loadingText.value;
  }
  if (!props.websites.length) {
    return emptyText.value;
  }
  return t('common.selectWebsite');
});

const sourceLabel = (site: WebsiteInfo) => (site.sourceType === 'remote' ? t('common.remoteSource') : t('common.localSource'));
const remoteSourceId = (site: WebsiteInfo) => site.remoteSourceId || site.sourceIds?.[0] || '';
</script>

<style scoped>
.website-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  width: 100%;
}

.website-option-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.website-option-tags {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.35rem;
}

.website-option-tag {
  border-radius: 999px;
  background: var(--surface-100, #f1f5f9);
  color: var(--text-color-secondary, #64748b);
  font-size: 0.72rem;
  line-height: 1;
  padding: 0.25rem 0.45rem;
  white-space: nowrap;
}

.website-option-tag-muted {
  max-width: 9rem;
  overflow: hidden;
  text-overflow: ellipsis;
}

.website-option-tag-accent {
  background: rgba(59, 130, 246, 0.12);
  color: #2563eb;
}
</style>
