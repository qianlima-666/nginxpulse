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
      filterBy="name"
      :filterPlaceholder="filterPlaceholderText"
    />
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
</script>
