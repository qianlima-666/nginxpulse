<template>
  <div class="logs-layout">
    <header class="page-header">
    <div class="page-title">
        <span class="title-chip">{{ t('logs.title') }}</span>
        <p class="title-sub">{{ t('logs.subtitle') }}</p>
      </div>
      <div class="header-actions header-actions-tech">
        <HeaderToolbar class="header-toolbar-tech">
          <template #primary>
            <div class="site-select-pill">
              <span class="site-label">{{ t('common.website') }}</span>
              <WebsiteSelect
                v-model="currentWebsiteId"
                class="website-select-compact"
                :websites="websites"
                :loading="websitesLoading"
                id="logs-website-selector"
                label=""
              />
            </div>
          </template>
          <template #utility>
            <SystemNotifications />
            <ThemeToggle />
          </template>
        </HeaderToolbar>
      </div>
    </header>

    <div class="card logs-control-box">
      <div class="logs-control-content">
        <div class="control-row">
          <div class="search-box">
            <InputText
              v-model="searchInput"
              class="search-input"
              :placeholder="t('logs.searchPlaceholder')"
              @keyup.enter="applySearch"
            />
            <div class="search-actions">
              <Button
                class="search-btn"
                severity="primary"
                icon="pi pi-search"
                :label="t('common.search')"
                @click="applySearch"
              />
              <Button
                class="reparse-btn action-btn"
                outlined
                severity="danger"
                icon="pi pi-refresh"
                :label="reparseButtonLabel"
                :disabled="!currentWebsiteId || isParsingBusy"
                @click="openReparseDialog"
              />
              <Button
                class="export-btn action-btn"
                outlined
                severity="success"
                icon="pi pi-file-export"
                :label="exportButtonLabel"
                :loading="exportLoading"
                :disabled="!currentWebsiteId || exportLoading"
                :title="isDemoMode ? t('logs.exportBlocked') : undefined"
                @click="handleExport"
              />
            </div>
          </div>
        <div class="filter-row filter-row-fields">
          <div class="filter-row-left">
            <div class="status-code-container">
              <label for="status-code">{{ t('logs.statusCode') }}</label>
              <InputText
                v-model="statusCodeFilter"
                inputId="status-code"
                class="status-code-input"
                :placeholder="t('logs.statusCodePlaceholder')"
              />
              <div class="status-code-quick">
                <button
                  type="button"
                  class="status-quick-btn"
                  :class="{ active: statusClassPreset === '' && !statusCodeFilter.trim() }"
                  @click="setStatusCodePreset('')"
                >
                  {{ t('common.all') }}
                </button>
                <button
                  type="button"
                  class="status-quick-btn"
                  :class="{ active: statusClassPreset === '2xx' }"
                  @click="setStatusCodePreset('2xx')"
                >
                  2xx
                </button>
                <button
                  type="button"
                  class="status-quick-btn"
                  :class="{ active: statusClassPreset === '3xx' }"
                  @click="setStatusCodePreset('3xx')"
                >
                  3xx
                </button>
                <button
                  type="button"
                  class="status-quick-btn"
                  :class="{ active: statusClassPreset === '4xx' }"
                  @click="setStatusCodePreset('4xx')"
                >
                  4xx
                </button>
                <button
                  type="button"
                  class="status-quick-btn"
                  :class="{ active: statusClassPreset === '5xx' }"
                  @click="setStatusCodePreset('5xx')"
                >
                  5xx
                </button>
              </div>
            </div>
            <div class="date-range-container">
              <label>{{ t('logs.dateRange') }}</label>
              <div class="date-range-inputs">
                <Dropdown
                  v-model="dateRangePreset"
                  class="date-range-select"
                  :options="dateRangePresetOptions"
                  optionLabel="label"
                  optionValue="value"
                />
              <DatePicker
                v-model="dateRange"
                class="date-range-picker"
                dateFormat="yy-mm-dd"
                selectionMode="range"
                showTime
                hourFormat="24"
                :hideOnRangeSelection="true"
                showButtonBar
                :showClear="true"
                :manualInput="false"
                :placeholder="`${t('logs.dateStart')} - ${t('logs.dateEnd')}`"
              />
            </div>
          </div>
          </div>
          <div class="filter-row-right">
            <div class="sort-field-container">
              <label for="sort-field">{{ t('logs.sortField') }}</label>
              <Dropdown
                inputId="sort-field"
                v-model="sortField"
                class="sort-select"
                :options="sortFieldOptions"
                optionLabel="label"
                optionValue="value"
              />
            </div>
            <div class="sort-order-container">
              <label for="sort-order">{{ t('logs.sortOrder') }}</label>
              <Dropdown
                inputId="sort-order"
                v-model="sortOrder"
                class="sort-select"
                :options="sortOrderOptions"
                optionLabel="label"
                optionValue="value"
              />
            </div>
            <div class="page-size-container">
              <label for="page-size">{{ t('logs.pageSize') }}</label>
              <Dropdown
                inputId="page-size"
                v-model="pageSize"
                class="sort-select"
                :options="pageSizeOptions"
                optionLabel="label"
                optionValue="value"
              />
            </div>
            <button
              class="advanced-toggle"
              type="button"
              :aria-expanded="advancedFiltersOpen"
              @click="advancedFiltersOpen = !advancedFiltersOpen"
            >
              <i class="ri-filter-3-line" aria-hidden="true"></i>
              <span>{{ advancedFiltersOpen ? t('logs.collapseFilters') : t('logs.advancedFilters') }}</span>
            </button>
          </div>
        </div>
        </div>
        <transition name="filter-collapse">
          <div v-if="advancedFiltersOpen" class="filter-row filter-row-toggles">
            <div class="filter-input-container filter-input-container-location">
              <label for="location-filter">{{ t('overview.ipLocation') }}</label>
              <InputText
                v-model.trim="locationFilter"
                inputId="location-filter"
                class="filter-text-input"
                :placeholder="t('overview.locationPlaceholder')"
                @keyup.enter="applyLocationFilterNow"
                @compositionstart="handleLocationFilterCompositionStart"
                @compositionend="handleLocationFilterCompositionEnd"
              />
            </div>
            <div class="filter-toggle-container">
              <Checkbox v-model="excludeInternal" inputId="exclude-internal" binary />
              <label for="exclude-internal">{{ t('logs.excludeInternal') }}</label>
            </div>
            <div class="filter-toggle-container">
              <Checkbox v-model="pageviewOnly" inputId="pageview-only" binary />
              <label for="pageview-only">{{ t('logs.excludeNoPv') }}</label>
            </div>
            <div class="filter-toggle-container">
              <Checkbox v-model="excludeSpider" inputId="exclude-spider" binary />
              <label for="exclude-spider">{{ t('logs.excludeSpider') }}</label>
            </div>
            <div class="filter-toggle-container">
              <Checkbox v-model="excludeForeign" inputId="exclude-foreign" binary />
              <label for="exclude-foreign">{{ t('logs.excludeForeign') }}</label>
            </div>
          </div>
        </transition>
      </div>
    </div>
    <div v-if="ipParsing || parsingPending || ipGeoParsing || ipGeoPending" class="logs-ip-notice">
      <div v-if="ipParsing">{{ t('logs.ipParsing', { progress: ipParsingProgressLabel }) }}</div>
      <div v-else-if="parsingPending">{{ t('logs.backfillParsing', { progress: parsingPendingProgressLabel }) }}</div>
      <div v-if="ipGeoParsing || ipGeoPending">{{ ipGeoParsingMessage }}</div>
    </div>

    <div class="card logs-table-box">
      <div class="logs-table-wrapper">
        <div v-if="loading" class="logs-table-overlay" role="status" aria-live="polite">
          <div class="logs-table-overlay-card">
            <span class="logs-table-overlay-spinner" aria-hidden="true"></span>
            <span>{{ t('common.loading') }}</span>
          </div>
        </div>
        <DataTable
          class="logs-table"
          :value="logs"
          scrollable
          scrollHeight="flex"
          :resizableColumns="true"
          columnResizeMode="fit"
          :rowHover="true"
          :stripedRows="true"
          :emptyMessage="''"
          :tableStyle="{ minWidth: '1460px' }"
          @row-click="openLogDetail"
        >
          <Column field="time" :header="t('logs.time')" :style="{ width: '180px' }">
            <template #body="{ data }">
              <span :title="data.time">{{ data.time }}</span>
            </template>
          </Column>
          <Column field="ip" :header="t('common.ip')" :style="{ width: '140px' }">
            <template #body="{ data }">
              <span :title="data.ip">{{ data.ip }}</span>
            </template>
          </Column>
          <Column field="location" :header="t('common.location')" :style="{ width: '160px' }">
            <template #body="{ data }">
              <span :title="data.location">{{ data.location }}</span>
            </template>
          </Column>
          <Column field="request" :header="t('logs.request')" :style="{ width: '240px' }">
            <template #body="{ data }">
              <span :title="data.request">{{ data.request }}</span>
            </template>
          </Column>
          <Column field="statusCode" :header="t('common.status')" :style="{ width: '110px' }">
            <template #body="{ data }">
              <span :style="{ color: statusColor(data.statusCode) }">{{ data.statusCode }}</span>
            </template>
          </Column>
          <Column field="trafficText" :header="t('common.traffic')" :style="{ width: '130px' }">
            <template #body="{ data }">
              <span :title="data.trafficTitle">{{ data.trafficText }}</span>
            </template>
          </Column>
          <Column field="requestTimeText" :header="t('logs.duration')" :style="{ width: '130px' }">
            <template #body="{ data }">
              <span :title="data.requestTimeTitle">{{ data.requestTimeText }}</span>
            </template>
          </Column>
          <Column field="requestSizeText" :header="t('logs.requestSize')" :style="{ width: '130px' }">
            <template #body="{ data }">
              <span :title="data.requestSizeTitle">{{ data.requestSizeText }}</span>
            </template>
          </Column>
          <Column field="referer" :header="t('logs.source')" :style="{ width: '220px' }">
            <template #body="{ data }">
              <span :title="data.referer">{{ data.referer }}</span>
            </template>
          </Column>
          <Column field="browser" :header="t('common.browser')" :style="{ width: '160px' }">
            <template #body="{ data }">
              <span :title="data.browser">{{ data.browser }}</span>
            </template>
          </Column>
          <Column field="os" :header="t('common.os')" :style="{ width: '150px' }">
            <template #body="{ data }">
              <span :title="data.os">{{ data.os }}</span>
            </template>
          </Column>
          <Column field="device" :header="t('common.device')" :style="{ width: '140px' }">
            <template #body="{ data }">
              <span :title="data.device">{{ data.device }}</span>
            </template>
          </Column>
          <Column field="pageview" :header="t('common.pageview')" :style="{ width: '90px' }" bodyClass="logs-pv-cell">
            <template #body="{ data }">
              <span :style="{ color: data.pageview ? 'var(--success-color)' : 'inherit' }">
                {{ data.pageview ? '✓' : '-' }}
              </span>
            </template>
          </Column>
        </DataTable>
        <div v-if="showLogsEmptyOverlay" class="logs-empty-overlay">
          <div class="logs-empty-state logs-empty-state-overlay">
            <span class="logs-empty-state-icon"><i class="ri-file-search-line"></i></span>
            <div class="logs-empty-state-title">{{ logsEmptyTitle }}</div>
            <div class="logs-empty-state-text">{{ logsEmptyText }}</div>
            <Button
              v-if="hasActiveLogFilters"
              class="logs-empty-action"
              outlined
              severity="secondary"
              :label="t('logs.clearFiltersAction')"
              @click="clearLogFilters"
            />
          </div>
        </div>
      </div>
    </div>

    <Dialog
      v-model:visible="logDetailVisible"
      modal
      class="log-detail-dialog"
      :header="t('logs.detailTitle')"
    >
      <div class="log-detail-grid">
        <div class="log-detail-item">
          <span class="log-detail-label">{{ t('logs.time') }}</span>
          <span class="log-detail-value">{{ selectedLog?.time || t('common.none') }}</span>
        </div>
        <div class="log-detail-item">
          <span class="log-detail-label">{{ t('common.ip') }}</span>
          <span class="log-detail-value">{{ selectedLog?.ip || t('common.none') }}</span>
        </div>
        <div class="log-detail-item">
          <span class="log-detail-label">{{ t('common.location') }}</span>
          <span class="log-detail-value">{{ selectedLog?.location || t('common.none') }}</span>
          <button
            type="button"
            class="log-detail-action"
            :disabled="!selectedLog?.ip || selectedLog.ip === t('common.none')"
            @click="openIPGeoEditor"
          >
            {{ t('logs.editLocation') }}
          </button>
        </div>
        <div class="log-detail-item">
          <span class="log-detail-label">{{ t('logs.request') }}</span>
          <span class="log-detail-value">{{ selectedLog?.request || t('common.none') }}</span>
        </div>
        <div class="log-detail-item">
          <span class="log-detail-label">{{ t('common.status') }}</span>
          <span class="log-detail-value" :style="{ color: statusColor(selectedLog?.statusCode ?? '') }">
            {{ selectedLog?.statusCode ?? t('common.none') }}
          </span>
        </div>
        <div class="log-detail-item">
          <span class="log-detail-label">{{ t('common.traffic') }}</span>
          <span class="log-detail-value" :title="selectedLog?.trafficTitle">
            {{ selectedLog?.trafficText || t('common.none') }}
          </span>
        </div>
        <div class="log-detail-item">
          <span class="log-detail-label">{{ t('logs.duration') }}</span>
          <span class="log-detail-value" :title="selectedLog?.requestTimeTitle">
            {{ selectedLog?.requestTimeText || t('common.none') }}
          </span>
        </div>
        <div class="log-detail-item">
          <span class="log-detail-label">{{ t('logs.requestSize') }}</span>
          <span class="log-detail-value" :title="selectedLog?.requestSizeTitle">
            {{ selectedLog?.requestSizeText || t('common.none') }}
          </span>
        </div>
        <div class="log-detail-item">
          <span class="log-detail-label">{{ t('logs.upstreamDuration') }}</span>
          <span class="log-detail-value" :title="selectedLog?.upstreamTimeTitle">
            {{ selectedLog?.upstreamTimeText || t('common.none') }}
          </span>
        </div>
        <div class="log-detail-item">
          <span class="log-detail-label">{{ t('logs.host') }}</span>
          <span class="log-detail-value">{{ selectedLog?.host || t('common.none') }}</span>
        </div>
        <div class="log-detail-item">
          <span class="log-detail-label">{{ t('logs.requestId') }}</span>
          <span class="log-detail-value">{{ selectedLog?.requestId || t('common.none') }}</span>
        </div>
        <div class="log-detail-item">
          <span class="log-detail-label">{{ t('logs.upstreamAddr') }}</span>
          <span class="log-detail-value">{{ selectedLog?.upstreamAddr || t('common.none') }}</span>
        </div>
        <div class="log-detail-item">
          <span class="log-detail-label">{{ t('logs.source') }}</span>
          <span class="log-detail-value">{{ selectedLog?.referer || t('common.none') }}</span>
        </div>
        <div class="log-detail-item">
          <span class="log-detail-label">{{ t('common.browser') }}</span>
          <span class="log-detail-value">{{ selectedLog?.browser || t('common.none') }}</span>
        </div>
        <div class="log-detail-item">
          <span class="log-detail-label">{{ t('common.os') }}</span>
          <span class="log-detail-value">{{ selectedLog?.os || t('common.none') }}</span>
        </div>
        <div class="log-detail-item">
          <span class="log-detail-label">{{ t('common.device') }}</span>
          <span class="log-detail-value">{{ selectedLog?.device || t('common.none') }}</span>
        </div>
        <div class="log-detail-item">
          <span class="log-detail-label">{{ t('common.pageview') }}</span>
          <span class="log-detail-value">
            {{ selectedLog?.pageview ? '✓' : '-' }}
          </span>
        </div>
      </div>
      <template #footer>
        <Button
          text
          severity="secondary"
          :label="t('logs.editLocation')"
          :disabled="!selectedLog?.ip || selectedLog.ip === t('common.none')"
          @click="openIPGeoEditor"
        />
        <Button :label="t('common.close')" @click="logDetailVisible = false" />
      </template>
    </Dialog>

    <Dialog
      v-model:visible="ipGeoEditorVisible"
      modal
      class="ip-geo-editor-dialog"
      :closable="!ipGeoEditorSaving && !ipGeoEditorResetting"
      :dismissableMask="!ipGeoEditorSaving && !ipGeoEditorResetting"
      :header="t('logs.editLocationTitle')"
      @hide="resetIPGeoEditorState"
    >
      <div class="reparse-dialog-body ip-geo-editor-body">
        <p class="reparse-dialog-note">{{ t('logs.editLocationHint', { ip: selectedLog?.ip || '-' }) }}</p>
        <div class="ip-geo-editor-grid">
          <div class="ip-geo-editor-field">
            <label for="ip-geo-domestic">{{ t('logs.locationDomestic') }}</label>
            <InputText
              id="ip-geo-domestic"
              v-model.trim="ipGeoFormDomestic"
              :disabled="ipGeoEditorLoading || ipGeoEditorSaving || ipGeoEditorResetting"
            />
          </div>
          <div class="ip-geo-editor-field">
            <label for="ip-geo-global">{{ t('logs.locationGlobal') }}</label>
            <InputText
              id="ip-geo-global"
              v-model.trim="ipGeoFormGlobal"
              :disabled="ipGeoEditorLoading || ipGeoEditorSaving || ipGeoEditorResetting"
            />
          </div>
        </div>
        <div class="ip-geo-editor-field">
          <label for="ip-geo-note">{{ t('logs.locationNote') }}</label>
          <textarea
            id="ip-geo-note"
            v-model.trim="ipGeoFormNote"
            class="ip-geo-editor-textarea"
            rows="4"
            :disabled="ipGeoEditorLoading || ipGeoEditorSaving || ipGeoEditorResetting"
          ></textarea>
        </div>
        <p v-if="ipGeoOverrideUpdatedAt" class="reparse-dialog-note">
          {{ t('logs.locationOverrideMeta', { time: ipGeoOverrideUpdatedAt }) }}
        </p>
        <p v-if="ipGeoEditorError" class="reparse-dialog-error">{{ ipGeoEditorError }}</p>
      </div>
      <template #footer>
        <Button
          v-if="ipGeoOverrideActive"
          text
          severity="danger"
          :label="t('logs.resetLocation')"
          :loading="ipGeoEditorResetting"
          :disabled="ipGeoEditorLoading || ipGeoEditorSaving"
          @click="handleResetIPGeoOverride"
        />
        <Button
          text
          severity="secondary"
          :label="t('common.cancel')"
          :disabled="ipGeoEditorSaving || ipGeoEditorResetting"
          @click="ipGeoEditorVisible = false"
        />
        <Button
          severity="primary"
          :label="t('common.confirm')"
          :loading="ipGeoEditorSaving"
          :disabled="ipGeoEditorLoading || ipGeoEditorResetting"
          @click="handleSaveIPGeoOverride"
        />
      </template>
    </Dialog>

    <div class="card pagination-box">
      <div class="pagination-controls">
        <Button class="page-btn" outlined :disabled="loading || currentPage <= 1" @click="prevPage">
          &lt; {{ t('logs.prevPage') }}
        </Button>
        <div class="pagination-center">
          <div class="page-info">
            <span>
              {{
                paginationExact
                  ? t('logs.pageInfo', { current: currentPage, total: totalPages })
                  : t('logs.pageInfoSimple', { current: currentPage })
              }}
            </span>
          </div>
          <div v-if="paginationExact" class="page-jump">
            <InputNumber
              v-model="pageJump"
              class="page-jump-input"
              :min="1"
              :max="totalPages || 1"
              :step="1"
              :useGrouping="false"
              :minFractionDigits="0"
              :maxFractionDigits="0"
              :placeholder="`1-${totalPages || 1}`"
              @keyup.enter="jumpToPage"
            />
            <Button class="page-btn" outlined :disabled="loading" @click="jumpToPage">{{ t('logs.jump') }}</Button>
          </div>
        </div>
        <Button
          class="page-btn"
          outlined
          :disabled="loading || (paginationExact ? currentPage >= totalPages : !hasMore)"
          @click="nextPage"
        >
          {{ t('logs.nextPage') }} &gt;
        </Button>
      </div>
    </div>

    <Dialog
      v-model:visible="migrationDialogVisible"
      modal
      :closable="!migrationLoading"
      :dismissableMask="!migrationLoading"
      class="reparse-dialog migration-dialog"
      :header="t('logs.migrationTitle')"
    >
      <div class="reparse-dialog-body">
        <p>{{ t('logs.migrationBody') }}</p>
        <p class="reparse-dialog-note">{{ t('logs.migrationNote') }}</p>
        <p v-if="migrationError" class="reparse-dialog-error">{{ migrationError }}</p>
      </div>
      <template #footer>
        <Button
          text
          severity="secondary"
          :label="t('logs.migrationCancel')"
          :disabled="migrationLoading"
          @click="migrationDialogVisible = false"
        />
        <Button
          severity="danger"
          :label="migrationButtonLabel"
          :loading="migrationLoading"
          @click="confirmMigration"
        />
      </template>
    </Dialog>

    <Dialog
      v-model:visible="reparseDialogVisible"
      modal
      :closable="!reparseLoading"
      :dismissableMask="!reparseLoading"
      class="reparse-dialog logs-reparse-dialog"
      :header="reparseDialogTitle"
      :style="{ width: '640px', maxWidth: 'calc(100vw - 32px)' }"
      :breakpoints="{ '768px': 'calc(100vw - 24px)' }"
    >
      <div class="reparse-dialog-body">
        <template v-if="reparseDialogMode === 'blocked'">
          <p>{{ t(reparseDialogBlockedMessageKey) }}</p>
        </template>
        <template v-else>
          <div class="reparse-mode-switch" role="tablist" :aria-label="t('logs.reparseTitle')">
            <button
              type="button"
              class="reparse-mode-btn"
              :class="{ active: reparseMode === 'full' }"
              :disabled="reparseLoading"
              @click="reparseMode = 'full'"
            >
              {{ t('logs.reparseModeFull') }}
            </button>
            <button
              type="button"
              class="reparse-mode-btn"
              :class="{ active: reparseMode === 'range' }"
              :disabled="reparseLoading"
              @click="reparseMode = 'range'"
            >
              {{ t('logs.reparseModeRange') }}
            </button>
          </div>
          <template v-if="reparseMode === 'full'">
            <p>
              {{ t('logs.reparseConfirm', { name: currentWebsiteLabel }) }}
            </p>
            <p class="reparse-dialog-note">{{ t('logs.reparseNote') }}</p>
          </template>
          <template v-else>
            <p>
              {{ t('logs.reparseRangeConfirm', { name: currentWebsiteLabel }) }}
            </p>
            <div class="reparse-range-grid">
              <div class="reparse-range-field">
                <label for="reparse-range-start">{{ t('logs.reparseRangeStart') }}</label>
                <DatePicker
                  id="reparse-range-start"
                  v-model="reparseRangeStartDate"
                  class="reparse-range-picker"
                  dateFormat="yy-mm-dd"
                  showTime
                  hourFormat="24"
                  showButtonBar
                  :showClear="true"
                  :manualInput="false"
                  :disabled="reparseLoading"
                />
              </div>
              <div class="reparse-range-field">
                <label for="reparse-range-end">{{ t('logs.reparseRangeEnd') }}</label>
                <DatePicker
                  id="reparse-range-end"
                  v-model="reparseRangeEndDate"
                  class="reparse-range-picker"
                  dateFormat="yy-mm-dd"
                  showTime
                  hourFormat="24"
                  showButtonBar
                  :showClear="true"
                  :manualInput="false"
                  :disabled="reparseLoading"
                />
              </div>
            </div>
            <p class="reparse-dialog-note">{{ t('logs.reparseRangeNote') }}</p>
          </template>
        </template>
        <p v-if="reparseError" class="reparse-dialog-error">{{ reparseError }}</p>
      </div>
      <template #footer>
        <template v-if="reparseDialogMode === 'blocked'">
          <Button :label="t('logs.reparseAcknowledge')" @click="reparseDialogVisible = false" />
        </template>
        <template v-else>
          <Button
            text
            severity="secondary"
            :label="t('logs.reparseCancel')"
            :disabled="reparseLoading"
            @click="reparseDialogVisible = false"
          />
          <Button
            severity="danger"
            :label="t('logs.reparseSubmit')"
            :loading="reparseLoading"
            @click="confirmReparse"
          />
        </template>
      </template>
    </Dialog>

    <Dialog
      v-model:visible="exportDialogVisible"
      modal
      class="reparse-dialog export-dialog"
      :header="t('logs.exportDialogTitle')"
    >
      <div class="reparse-dialog-body">
        <template v-if="exportJob">
          <p>
            {{ t('logs.exportCurrent') }}
            <span class="export-status">{{ exportStatusLabel }}</span>
          </p>
          <div v-if="exportProgressPercent !== null" class="export-progress">
            <div class="export-progress-bar" :style="{ width: `${exportProgressPercent}%` }"></div>
          </div>
          <p v-if="exportProgressText" class="reparse-dialog-note">{{ exportProgressText }}</p>
        </template>
        <p v-if="exportJobError" class="reparse-dialog-error">{{ exportJobError }}</p>
        <p class="reparse-dialog-note">{{ t('logs.exportHistory') }}</p>
        <DataTable
          :value="exportJobs"
          :loading="exportJobsLoading"
          scrollable
          scrollHeight="240px"
          class="export-table"
          ref="exportHistoryTableRef"
        >
          <Column :header="t('logs.time')">
            <template #body="{ data }">
              {{ formatExportJobTime(data.created_at) }}
            </template>
          </Column>
          <Column :header="t('logs.exportStatus')">
            <template #body="{ data }">
              {{ formatExportStatus(data.status) }}
            </template>
          </Column>
          <Column :header="t('logs.exportProgressLabel')">
            <template #body="{ data }">
              {{ formatExportJobProgress(data) }}
            </template>
          </Column>
          <Column :header="t('logs.exportFile')">
            <template #body="{ data }">
              {{ data.fileName || '-' }}
            </template>
          </Column>
          <Column :header="t('common.action')">
            <template #body="{ data }">
              <Button
                text
                severity="secondary"
                :disabled="data.status !== 'success'"
                :label="t('logs.exportDownload')"
                @click="handleDownloadHistory(data)"
              />
              <Button
                text
                severity="secondary"
                :disabled="!canRetryExport(data)"
                :title="isDemoMode ? t('logs.exportBlocked') : undefined"
                :label="t('logs.exportRetry')"
                @click="handleRetryHistory(data)"
              />
            </template>
          </Column>
        </DataTable>
        <p v-if="exportHistoryHasMore" class="reparse-dialog-note">{{ t('logs.exportHistoryScrollHint') }}</p>
        <p v-if="exportHistoryLoadingMore" class="reparse-dialog-note">{{ t('logs.exportHistoryLoadingMore') }}</p>
      </div>
      <template #footer>
        <Button
          v-if="exportJob && (exportJob.status === 'running' || exportJob.status === 'pending')"
          text
          severity="secondary"
          :label="t('logs.exportCancel')"
          :loading="exportCancelLoading"
          @click="cancelExportJob"
        />
        <Button :label="t('common.close')" @click="exportDialogVisible = false" />
      </template>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, onMounted, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';
import Dialog from 'primevue/dialog';
import DataTable from 'primevue/datatable';
import Column from 'primevue/column';
import {
  fetchLogs,
  fetchWebsites,
  cancelLogsExport,
  downloadLogsExport,
  fetchIPGeoOverride,
  reparseAllLogs,
  reparseLogs,
  resetIPGeoOverride,
  listLogsExportJobs,
  saveIPGeoOverride,
  startLogsExport,
  fetchLogsExportStatus,
  retryLogsExport,
} from '@/api';
import type { IPGeoOverrideResponse, LogsExportJob, WebsiteInfo } from '@/api/types';
import { formatTraffic, getUserPreference, saveUserPreference } from '@/utils';
import { formatBrowserLabel, formatDeviceLabel, formatLocationLabel, formatOSLabel, formatRefererLabel } from '@/i18n/mappings';
import { normalizeLocale } from '@/i18n';
import HeaderToolbar from '@/components/HeaderToolbar.vue';
import SystemNotifications from '@/components/SystemNotifications.vue';
import ThemeToggle from '@/components/ThemeToggle.vue';
import WebsiteSelect from '@/components/WebsiteSelect.vue';

type LogRow = {
  time: string;
  ip: string;
  location: string;
  request: string;
  statusCode: number | string;
  trafficText: string;
  trafficTitle: string;
  requestTimeText: string;
  requestTimeTitle: string;
  requestSizeText: string;
  requestSizeTitle: string;
  upstreamTimeText: string;
  upstreamTimeTitle: string;
  host: string;
  requestId: string;
  upstreamAddr: string;
  domesticLocation: string;
  globalLocation: string;
  locationNote: string;
  locationOverrideActive: boolean;
  referer: string;
  browser: string;
  os: string;
  device: string;
  pageview: boolean;
};

type LogRowClickEvent = {
  data: LogRow;
  originalEvent?: MouseEvent;
};

const websites = ref<WebsiteInfo[]>([]);
const websitesLoading = ref(true);
const currentWebsiteId = ref('');
const route = useRoute();
const routeWebsiteIdOverride = ref('');

const searchInput = ref('');
const searchFilter = ref('');
const locationFilter = ref('');
const isLocationFilterComposing = ref(false);
const excludeInternal = ref(false);
const pageviewOnly = ref(false);
const excludeSpider = ref(false);
const excludeForeign = ref(false);
const statusCodeFilter = ref('');
const timeStart = ref<string>('');
const timeEnd = ref<string>('');
const dateRange = ref<Date[] | null>(null);
const dateRangePreset = ref('custom');
let updatingDatePreset = false;
let updatingDateRange = false;
const sortField = ref(getUserPreference('logsSortField', 'timestamp'));
const sortOrder = ref(getUserPreference('logsSortOrder', 'desc'));
const pageSize = ref(Number(getUserPreference('logsPageSize', '100')));
const advancedFiltersOpen = ref(false);
const currentPage = ref(1);
const totalPages = ref(0);
const hasMore = ref(false);
const paginationExact = ref(true);
const pageJump = ref<number | null>(null);
const reparseDialogVisible = ref(false);
const reparseLoading = ref(false);
const reparseError = ref('');
const reparseDialogMode = ref<'confirm' | 'blocked'>('confirm');
const reparseMode = ref<'full' | 'range'>('full');
const reparseRangeStartDate = ref<Date | null>(null);
const reparseRangeEndDate = ref<Date | null>(null);
const reparseDialogBlockedMessageKey = ref<'logs.reparseBlocked' | 'logs.exportBlocked'>(
  'logs.reparseBlocked'
);
const migrationDialogVisible = ref(false);
const migrationLoading = ref(false);
const migrationError = ref('');
const exportLoading = ref(false);
const exportDialogVisible = ref(false);
const exportJob = ref<LogsExportJob | null>(null);
const exportJobError = ref('');
const exportJobs = ref<LogsExportJob[]>([]);
const exportJobsLoading = ref(false);
const exportCancelLoading = ref(false);
let exportPollTimer: ReturnType<typeof setInterval> | null = null;
const exportHistoryPageSize = 20;
const exportHistoryPage = ref(1);
const exportHistoryHasMore = ref(false);
const exportHistoryLoadingMore = ref(false);
const exportHistoryTableRef = ref<InstanceType<typeof DataTable> | null>(null);
let exportHistoryScrollHandler: ((event: Event) => void) | null = null;
let exportHistoryTimer: ReturnType<typeof setInterval> | null = null;
const locationFilterDebounceMs = 450;
let locationFilterDebounceTimer: ReturnType<typeof setTimeout> | null = null;
let suppressLocationFilterWatch = false;
const demoMode = inject<{ value: boolean } | null>('demoMode', null);
const migrationRequired = inject<{ value: boolean } | null>('migrationRequired', null);
const migrationAckKey = 'pgMigrationAck';

const { t, n, locale } = useI18n({ useScope: 'global' });
const currentLocale = computed(() => normalizeLocale(locale.value));

const sortFieldOptions = computed(() => [
  { value: 'timestamp', label: t('logs.time') },
  { value: 'ip', label: t('common.ip') },
  { value: 'url', label: t('common.url') },
  { value: 'status_code', label: t('common.status') },
  { value: 'bytes_sent', label: t('common.traffic') },
  { value: 'request_time_ms', label: t('logs.duration') },
  { value: 'request_length', label: t('logs.requestSize') },
  { value: 'upstream_response_time_ms', label: t('logs.upstreamDuration') },
]);
const sortOrderOptions = computed(() => [
  { value: 'desc', label: t('logs.sortDesc') },
  { value: 'asc', label: t('logs.sortAsc') },
]);
const pageSizeOptions = [50, 100, 200, 500].map((value) => ({ value, label: `${value}` }));
const dateRangePresetOptions = computed(() => [
  { value: 'today', label: t('common.today') },
  { value: 'yesterday', label: t('common.yesterday') },
  { value: 'last7Days', label: t('common.last7Days') },
  { value: 'last30Days', label: t('common.last30Days') },
  { value: 'all', label: t('common.all') },
  { value: 'custom', label: t('logs.dateRangeCustom') },
]);

const rawLogs = ref<Array<Record<string, any>>>([]);
const loading = ref(false);
const ipParsing = ref(false);
const ipParsingProgress = ref<number | null>(null);
const ipParsingEstimatedRemainingSeconds = ref<number | null>(null);
const ipGeoParsing = ref(false);
const ipGeoPending = ref(false);
const ipGeoProgress = ref<number | null>(null);
const ipGeoEstimatedRemainingSeconds = ref<number | null>(null);
const parsingPending = ref(false);
const parsingPendingProgress = ref<number | null>(null);
const logDetailVisible = ref(false);
const selectedLog = ref<LogRow | null>(null);
const ipGeoEditorVisible = ref(false);
const ipGeoEditorLoading = ref(false);
const ipGeoEditorSaving = ref(false);
const ipGeoEditorResetting = ref(false);
const ipGeoEditorError = ref('');
const ipGeoFormDomestic = ref('');
const ipGeoFormGlobal = ref('');
const ipGeoFormNote = ref('');
const ipGeoOverrideActive = ref(false);
const ipGeoOverrideUpdatedAt = ref('');
const progressPollIntervalMs = 3000;
let progressPollTimer: ReturnType<typeof setInterval> | null = null;
let progressPollInFlight = false;

const ipParsingProgressText = computed(() => {
  if (ipParsingProgress.value === null) {
    return '';
  }
  if (ipParsingEstimatedRemainingSeconds.value) {
    const duration = formatDurationSeconds(ipParsingEstimatedRemainingSeconds.value);
    return t('parsing.progressWithRemaining', { value: ipParsingProgress.value, duration });
  }
  return t('parsing.progress', { value: ipParsingProgress.value });
});
const ipParsingProgressLabel = computed(() => {
  if (!ipParsingProgressText.value) {
    return '';
  }
  return currentLocale.value === 'zh-CN'
    ? `（${ipParsingProgressText.value}）`
    : ` (${ipParsingProgressText.value})`;
});

const ipGeoProgressText = computed(() => {
  if (ipGeoProgress.value === null) {
    return '';
  }
  return t('parsing.progress', { value: ipGeoProgress.value });
});
const ipGeoProgressLabel = computed(() => {
  if (!ipGeoProgressText.value) {
    return '';
  }
  return currentLocale.value === 'zh-CN'
    ? `（${ipGeoProgressText.value}）`
    : ` (${ipGeoProgressText.value})`;
});
const ipGeoRemainingLabel = computed(() => {
  if (ipGeoEstimatedRemainingSeconds.value === null) {
    return '';
  }
  return formatDurationSeconds(ipGeoEstimatedRemainingSeconds.value);
});
const ipGeoParsingMessage = computed(() => {
  if (ipGeoProgressLabel.value && ipGeoRemainingLabel.value) {
    return t('logs.ipGeoParsingProgress', {
      progress: ipGeoProgressLabel.value,
      remaining: ipGeoRemainingLabel.value,
    });
  }
  if (ipGeoProgressLabel.value) {
    return t('logs.ipGeoParsingProgressOnly', { progress: ipGeoProgressLabel.value });
  }
  return t('logs.ipGeoParsing');
});

const parsingPendingProgressText = computed(() => {
  if (parsingPendingProgress.value === null) {
    return '';
  }
  return t('parsing.progress', { value: parsingPendingProgress.value });
});
const parsingPendingProgressLabel = computed(() => {
  if (!parsingPendingProgressText.value) {
    return '';
  }
  return currentLocale.value === 'zh-CN'
    ? `（${parsingPendingProgressText.value}）`
    : ` (${parsingPendingProgressText.value})`;
});

const currentWebsiteLabel = computed(() => {
  const match = websites.value.find((site) => site.id === currentWebsiteId.value);
  return match?.name || t('common.currentWebsite');
});

const isParsingBusy = computed(() => reparseLoading.value || migrationLoading.value || ipParsing.value);
const reparseButtonLabel = computed(() =>
  isParsingBusy.value ? t('logs.reparseLoading') : t('logs.reparse')
);
const migrationButtonLabel = computed(() =>
  migrationLoading.value ? t('logs.migrationLoading') : t('logs.migrationSubmit')
);
const exportButtonLabel = computed(() =>
  exportLoading.value ? t('logs.exportLoading') : t('logs.export')
);
const exportProgressPercent = computed(() => {
  const job = exportJob.value;
  if (!job || !job.total || job.total <= 0) {
    return null;
  }
  const processed = job.processed ?? 0;
  return Math.min(100, Math.max(0, Math.round((processed / job.total) * 100)));
});
const exportProgressText = computed(() => {
  const job = exportJob.value;
  if (!job) {
    return '';
  }
  if (job.total && job.total > 0) {
    return t('logs.exportProgress', { processed: job.processed ?? 0, total: job.total });
  }
  if (job.processed) {
    return t('logs.exportProgressOnly', { processed: job.processed ?? 0 });
  }
  return '';
});
const exportStatusLabel = computed(() => {
  const status = exportJob.value?.status;
  if (!status) {
    return '';
  }
  switch (status) {
    case 'pending':
      return t('logs.exportStatusPending');
    case 'running':
      return t('logs.exportStatusRunning');
    case 'success':
      return t('logs.exportStatusSuccess');
    case 'failed':
      return t('logs.exportStatusFailed');
    case 'canceled':
      return t('logs.exportStatusCanceled');
    default:
      return status;
  }
});
const hasActiveLogFilters = computed(() =>
  Boolean(currentWebsiteId.value) &&
  Boolean(
    searchFilter.value ||
    locationFilter.value.trim() ||
    statusCodeFilter.value.trim() ||
    excludeInternal.value ||
    pageviewOnly.value ||
    excludeSpider.value ||
    excludeForeign.value ||
    dateRangePreset.value !== 'last7Days'
  )
);
const logsEmptyTitle = computed(() =>
  hasActiveLogFilters.value ? t('logs.filteredEmptyTitle') : t('logs.emptyTitle')
);
const logsEmptyText = computed(() =>
  hasActiveLogFilters.value ? t('logs.filteredEmptyText') : t('logs.emptyText')
);
const showLogsEmptyOverlay = computed(() => !loading.value && logs.value.length === 0);
const statusClassPreset = computed(() => parseStatusFilter(statusCodeFilter.value).statusClass || '');
const isDemoMode = computed(() => demoMode?.value ?? false);
const reparseDialogTitle = computed(() =>
  reparseDialogMode.value === 'blocked' ? t('demo.badge') : t('logs.reparseTitle')
);

function getRouteQueryValue(key: string) {
  const raw = route.query[key];
  if (Array.isArray(raw)) {
    return String(raw[0] || '').trim();
  }
  return String(raw || '').trim();
}

function isDateTimeValue(value: string) {
  return /^\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}$/.test(value || '');
}

function getIPFilterFromSearch(value: string) {
  const normalized = String(value || '').trim();
  if (!normalized) {
    return '';
  }
  const ipv4Pattern =
    /^(25[0-5]|2[0-4]\d|1?\d?\d)(\.(25[0-5]|2[0-4]\d|1?\d?\d)){3}$/;
  if (ipv4Pattern.test(normalized)) {
    return normalized;
  }
  const ipv6Pattern = /^[0-9a-fA-F:.]+$/;
  if (normalized.includes(':') && ipv6Pattern.test(normalized)) {
    return normalized;
  }
  return '';
}

function normalizeProgress(value: unknown): number | null {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return null;
  }
  return Math.min(100, Math.max(0, Math.round(value)));
}

function normalizeSeconds(value: unknown): number | null {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return null;
  }
  const normalized = Math.round(value);
  if (normalized <= 0) {
    return null;
  }
  return normalized;
}

function formatDurationSeconds(seconds: number) {
  const total = Math.max(0, Math.floor(seconds));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = total % 60;
  if (hours > 0) {
    return t('overview.durationHoursMinutes', { hours, minutes });
  }
  if (minutes > 0) {
    return t('overview.durationMinutesSeconds', { minutes, seconds: secs });
  }
  return t('overview.durationSeconds', { seconds: secs });
}

function formatMilliseconds(value: unknown) {
  const normalized = typeof value === 'number' ? value : Number(value);
  if (!Number.isFinite(normalized) || normalized <= 0) {
    return t('common.none');
  }
  return `${Math.round(normalized)} ms`;
}

function formatBytesTitle(value: number) {
  return t('common.bytes', { value: n(value) });
}

function parseStatusFilter(value: string) {
  const trimmed = value.trim().toLowerCase();
  if (!trimmed) {
    return {};
  }
  if (/^[2-5]\d{2}$/.test(trimmed)) {
    return { statusCode: trimmed };
  }
  const classMatch = trimmed.match(/^([2-5])(?:xx|x{2,3}|\*{2,3})$/);
  if (classMatch?.[1]) {
    return { statusClass: `${classMatch[1]}xx` };
  }
  return {};
}

function resolveStatusParams() {
  return parseStatusFilter(statusCodeFilter.value);
}

function setStatusCodePreset(preset: string) {
  statusCodeFilter.value = preset;
}

function clearLocationFilterDebounce() {
  if (locationFilterDebounceTimer) {
    clearTimeout(locationFilterDebounceTimer);
    locationFilterDebounceTimer = null;
  }
}

function setLocationFilterSilently(value: string) {
  suppressLocationFilterWatch = true;
  locationFilter.value = value;
}

function scheduleLocationFilterSearch() {
  clearLocationFilterDebounce();
  locationFilterDebounceTimer = setTimeout(() => {
    locationFilterDebounceTimer = null;
    currentPage.value = 1;
    loadLogs();
  }, locationFilterDebounceMs);
}

function applyLocationFilterNow() {
  clearLocationFilterDebounce();
  currentPage.value = 1;
  loadLogs();
}

function handleLocationFilterCompositionStart() {
  isLocationFilterComposing.value = true;
  clearLocationFilterDebounce();
}

function handleLocationFilterCompositionEnd() {
  isLocationFilterComposing.value = false;
  scheduleLocationFilterSearch();
}

function clearLogFilters() {
  searchInput.value = '';
  searchFilter.value = '';
  clearLocationFilterDebounce();
  setLocationFilterSilently('');
  statusCodeFilter.value = '';
  excludeInternal.value = false;
  pageviewOnly.value = false;
  excludeSpider.value = false;
  excludeForeign.value = false;
  advancedFiltersOpen.value = false;
  currentPage.value = 1;
  updatingDatePreset = true;
  dateRangePreset.value = 'last7Days';
  applyDatePreset('last7Days');
  updatingDatePreset = false;
  saveUserPreference('logsDatePreset', dateRangePreset.value || '');
  loadLogs();
}

function buildExportParams() {
  const { statusCode, statusClass } = resolveStatusParams();
  const ipFilter = getIPFilterFromSearch(searchFilter.value);
  const params: Record<string, unknown> = {
    id: currentWebsiteId.value,
    website_id: currentWebsiteId.value,
    page: currentPage.value,
    pageSize: pageSize.value,
    sortField: sortField.value,
    sortOrder: sortOrder.value,
    lang: currentLocale.value,
  };
  if (searchFilter.value) {
    params.filter = searchFilter.value;
  }
  if (statusClass) {
    params.statusClass = statusClass;
  }
  if (statusCode) {
    params.statusCode = statusCode;
  }
  if (excludeInternal.value) {
    params.excludeInternal = true;
  }
  if (ipFilter) {
    params.ipFilter = ipFilter;
  }
  if (locationFilter.value.trim()) {
    params.locationFilter = locationFilter.value.trim();
  }
  if (pageviewOnly.value) {
    params.pageviewOnly = true;
  }
  if (excludeSpider.value) {
    params.excludeSpider = true;
  }
  if (excludeForeign.value) {
    params.excludeForeign = true;
  }
  if (timeStart.value) {
    params.timeStart = timeStart.value;
  }
  if (timeEnd.value) {
    params.timeEnd = timeEnd.value;
  }
  return params;
}

function extractExportFileName(disposition?: string) {
  if (!disposition) {
    return '';
  }
  const utf8Match = disposition.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf8Match?.[1]) {
    try {
      return decodeURIComponent(utf8Match[1]);
    } catch {
      return utf8Match[1];
    }
  }
  const quotedMatch = disposition.match(/filename=\"([^\"]+)\"/i);
  if (quotedMatch?.[1]) {
    return quotedMatch[1];
  }
  const fallbackMatch = disposition.match(/filename=([^;]+)/i);
  return fallbackMatch?.[1]?.trim() || '';
}

function formatExportTimestamp() {
  const now = new Date();
  const pad = (value: number) => `${value}`.padStart(2, '0');
  return `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}_${pad(
    now.getHours()
  )}${pad(now.getMinutes())}${pad(now.getSeconds())}`;
}

function formatExportJobTime(value?: string) {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

function formatExportStatus(status?: string) {
  if (!status) {
    return '-';
  }
  switch (status) {
    case 'pending':
      return t('logs.exportStatusPending');
    case 'running':
      return t('logs.exportStatusRunning');
    case 'success':
      return t('logs.exportStatusSuccess');
    case 'failed':
      return t('logs.exportStatusFailed');
    case 'canceled':
      return t('logs.exportStatusCanceled');
    default:
      return status;
  }
}

function formatExportJobProgress(job: LogsExportJob) {
  const processed = job.processed ?? 0;
  const total = job.total ?? 0;
  if (total > 0) {
    return `${processed}/${total}`;
  }
  if (processed > 0) {
    return `${processed}`;
  }
  return '-';
}

function canRetryExport(job: LogsExportJob) {
  return job.status === 'failed' || job.status === 'canceled';
}

function formatDateTimeValue(date: Date) {
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(
    date.getHours()
  )}:${pad(date.getMinutes())}`;
}

function parseDateFromTime(value: string) {
  const match = value.match(/^(\d{4})-(\d{2})-(\d{2})(?:\s(\d{2}):(\d{2}))?$/);
  if (!match) {
    return null;
  }
  const year = Number(match[1]);
  const month = Number(match[2]) - 1;
  const day = Number(match[3]);
  const hour = match[4] ? Number(match[4]) : 0;
  const minute = match[5] ? Number(match[5]) : 0;
  if (
    !Number.isFinite(year) ||
    !Number.isFinite(month) ||
    !Number.isFinite(day) ||
    !Number.isFinite(hour) ||
    !Number.isFinite(minute)
  ) {
    return null;
  }
  return new Date(year, month, day, hour, minute, 0, 0);
}

function buildDateRangeFromTime(start: string, end: string) {
  const startDate = start ? parseDateFromTime(start) : null;
  const endDate = end ? parseDateFromTime(end) : null;
  if (!startDate && !endDate) {
    return null;
  }
  if (startDate && endDate) {
    return [startDate, endDate];
  }
  return [startDate || endDate] as Date[];
}

function startOfDay(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate(), 0, 0, 0);
}

function endOfDay(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate(), 23, 59, 59);
}

function applyDatePreset(preset: string) {
  const now = new Date();
  switch (preset) {
    case 'today': {
      timeStart.value = formatDateTimeValue(startOfDay(now));
      timeEnd.value = formatDateTimeValue(now);
      break;
    }
    case 'yesterday': {
      const base = new Date(now);
      base.setDate(base.getDate() - 1);
      timeStart.value = formatDateTimeValue(startOfDay(base));
      timeEnd.value = formatDateTimeValue(endOfDay(base));
      break;
    }
    case 'last7Days': {
      const start = new Date(now);
      start.setDate(start.getDate() - 6);
      timeStart.value = formatDateTimeValue(startOfDay(start));
      timeEnd.value = formatDateTimeValue(now);
      break;
    }
    case 'last30Days': {
      const start = new Date(now);
      start.setDate(start.getDate() - 29);
      timeStart.value = formatDateTimeValue(startOfDay(start));
      timeEnd.value = formatDateTimeValue(now);
      break;
    }
    case 'all': {
      timeStart.value = '';
      timeEnd.value = '';
      break;
    }
    default:
      break;
  }
}

async function handleExport() {
  if (!currentWebsiteId.value || exportLoading.value) {
    return;
  }
  if (isDemoMode.value) {
    openDemoBlockedDialog('logs.exportBlocked');
    return;
  }
  stopExportPolling();
  exportLoading.value = true;
  exportDialogVisible.value = true;
  exportJobError.value = '';
  exportJob.value = null;
  try {
    const start = await startLogsExport(buildExportParams());
    exportJob.value = {
      id: start.job_id,
      status: start.status,
      fileName: start.fileName,
      website_id: currentWebsiteId.value,
    };
    startExportPolling();
    await refreshExportJobs();
  } catch (error) {
    console.error('导出日志失败:', error);
    exportJobError.value = error instanceof Error ? error.message : t('logs.exportError');
    exportLoading.value = false;
  } finally {
  }
}

async function refreshExportJobs() {
  if (!currentWebsiteId.value) {
    exportJobs.value = [];
    return;
  }
  exportJobsLoading.value = true;
  try {
    const response = await listLogsExportJobs(currentWebsiteId.value, 1, exportHistoryPageSize);
    exportJobs.value = response.jobs || [];
    exportHistoryPage.value = 1;
    exportHistoryHasMore.value = Boolean(response.has_more);
    updateExportHistoryPolling();
  } catch (error) {
    console.debug('读取导出任务失败:', error);
  } finally {
    exportJobsLoading.value = false;
  }
}

async function loadMoreExportJobs() {
  if (!currentWebsiteId.value || exportHistoryLoadingMore.value || !exportHistoryHasMore.value) {
    return;
  }
  exportHistoryLoadingMore.value = true;
  try {
    const nextPage = exportHistoryPage.value + 1;
    const response = await listLogsExportJobs(currentWebsiteId.value, nextPage, exportHistoryPageSize);
    const incoming = response.jobs || [];
    if (incoming.length > 0) {
      const existing = new Set(exportJobs.value.map((job) => job.id));
      const merged = exportJobs.value.slice();
      for (const job of incoming) {
        if (!existing.has(job.id)) {
          merged.push(job);
        }
      }
      exportJobs.value = merged;
    }
    exportHistoryPage.value = nextPage;
    exportHistoryHasMore.value = Boolean(response.has_more);
    updateExportHistoryPolling();
  } catch (error) {
    console.debug('加载更多导出任务失败:', error);
  } finally {
    exportHistoryLoadingMore.value = false;
  }
}

async function refreshExportJobsSilently() {
  if (!currentWebsiteId.value || exportHistoryPage.value > 1) {
    return;
  }
  try {
    const response = await listLogsExportJobs(currentWebsiteId.value, 1, exportHistoryPageSize);
    const latest = response.jobs || [];
    if (latest.length === 0) {
      exportJobs.value = [];
      exportHistoryHasMore.value = Boolean(response.has_more);
      return;
    }
    const existingMap = new Map(exportJobs.value.map((job) => [job.id, job]));
    const updated: LogsExportJob[] = [];
    for (const job of latest) {
      if (existingMap.has(job.id)) {
        updated.push({ ...existingMap.get(job.id)!, ...job });
      } else {
        updated.push(job);
      }
    }
    for (const job of exportJobs.value) {
      if (!updated.find((item) => item.id === job.id)) {
        updated.push(job);
      }
    }
    exportJobs.value = updated;
    exportHistoryHasMore.value = Boolean(response.has_more);
    updateExportHistoryPolling();
  } catch (error) {
    console.debug('刷新导出任务失败:', error);
  }
}

function bindExportHistoryScroll() {
  if (exportHistoryScrollHandler || !exportHistoryTableRef.value) {
    return;
  }
  const wrapper = exportHistoryTableRef.value.$el?.querySelector?.('.p-datatable-wrapper') as
    | HTMLElement
    | undefined;
  if (!wrapper) {
    return;
  }
  exportHistoryScrollHandler = () => {
    if (!exportHistoryHasMore.value || exportHistoryLoadingMore.value) {
      return;
    }
    const threshold = 40;
    if (wrapper.scrollTop + wrapper.clientHeight >= wrapper.scrollHeight - threshold) {
      loadMoreExportJobs();
    }
  };
  wrapper.addEventListener('scroll', exportHistoryScrollHandler);
}

function unbindExportHistoryScroll() {
  if (!exportHistoryScrollHandler || !exportHistoryTableRef.value) {
    exportHistoryScrollHandler = null;
    return;
  }
  const wrapper = exportHistoryTableRef.value.$el?.querySelector?.('.p-datatable-wrapper') as
    | HTMLElement
    | undefined;
  if (wrapper) {
    wrapper.removeEventListener('scroll', exportHistoryScrollHandler);
  }
  exportHistoryScrollHandler = null;
}

function startExportHistoryPolling() {
  if (exportHistoryTimer) {
    return;
  }
  exportHistoryTimer = setInterval(() => {
    refreshExportJobsSilently();
  }, 4000);
  refreshExportJobsSilently();
}

function stopExportHistoryPolling() {
  if (exportHistoryTimer) {
    clearInterval(exportHistoryTimer);
    exportHistoryTimer = null;
  }
}

function updateExportHistoryPolling() {
  if (!exportDialogVisible.value) {
    stopExportHistoryPolling();
    return;
  }
  if (exportHistoryPage.value > 1) {
    stopExportHistoryPolling();
    return;
  }
  const hasRunning = exportJobs.value.some((job) => job.status === 'running' || job.status === 'pending');
  const currentActive =
    exportJob.value?.status === 'running' || exportJob.value?.status === 'pending';
  if (hasRunning || currentActive) {
    startExportHistoryPolling();
  } else {
    stopExportHistoryPolling();
  }
}

async function refreshCurrentExportStatus() {
  if (!exportJob.value?.id) {
    return;
  }
  try {
    const status = await fetchLogsExportStatus(exportJob.value.id);
    exportJob.value = status;
    if (status.website_id && currentWebsiteId.value && status.website_id !== currentWebsiteId.value) {
      stopExportPolling();
      exportLoading.value = false;
      await refreshExportJobs();
      return;
    }
    updateExportHistoryPolling();
    if (status.status === 'success') {
      stopExportPolling();
      exportLoading.value = false;
      await downloadExportJob(status.id, status.fileName);
      await refreshExportJobs();
    } else if (status.status === 'failed' || status.status === 'canceled') {
      stopExportPolling();
      exportLoading.value = false;
      exportJobError.value = status.error || (status.status === 'canceled' ? t('logs.exportCanceled') : t('logs.exportError'));
      await refreshExportJobs();
    }
  } catch (error) {
    console.debug('读取导出状态失败:', error);
  }
}

function startExportPolling() {
  if (exportPollTimer) {
    return;
  }
  exportPollTimer = setInterval(() => {
    refreshCurrentExportStatus();
  }, 1500);
  refreshCurrentExportStatus();
}

function stopExportPolling() {
  if (exportPollTimer) {
    clearInterval(exportPollTimer);
    exportPollTimer = null;
  }
}

async function cancelExportJob() {
  if (!exportJob.value?.id || exportCancelLoading.value) {
    return;
  }
  exportCancelLoading.value = true;
  try {
    await cancelLogsExport(exportJob.value.id);
    exportJob.value = { ...exportJob.value, status: 'canceled' };
    exportJobError.value = t('logs.exportCanceled');
    exportLoading.value = false;
    stopExportPolling();
    updateExportHistoryPolling();
    await refreshExportJobs();
  } catch (error) {
    exportJobError.value = error instanceof Error ? error.message : t('logs.exportError');
  } finally {
    exportCancelLoading.value = false;
  }
}

async function downloadExportJob(jobId: string, fallbackName?: string) {
  const response = await downloadLogsExport(jobId, currentWebsiteId.value || undefined);
  const headerName = extractExportFileName(response.headers?.['content-disposition']);
  const fileName = headerName || fallbackName || `nginxpulse_logs_${formatExportTimestamp()}.xlsx`;
  const url = window.URL.createObjectURL(response.data);
  const link = document.createElement('a');
  link.href = url;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  window.URL.revokeObjectURL(url);
}

function handleDownloadHistory(job: LogsExportJob) {
  if (!job?.id || job.status !== 'success') {
    return;
  }
  if (job.website_id && currentWebsiteId.value && job.website_id !== currentWebsiteId.value) {
    return;
  }
  downloadExportJob(job.id, job.fileName);
}

async function handleRetryHistory(job: LogsExportJob) {
  if (!job?.id || exportLoading.value) {
    return;
  }
  if (isDemoMode.value) {
    openDemoBlockedDialog('logs.exportBlocked');
    return;
  }
  stopExportPolling();
  exportLoading.value = true;
  exportJobError.value = '';
  try {
    const start = await retryLogsExport(job.id);
    exportJob.value = {
      id: start.job_id,
      status: start.status,
      fileName: start.fileName,
      website_id: currentWebsiteId.value,
    };
    exportDialogVisible.value = true;
    startExportPolling();
    await refreshExportJobs();
  } catch (error) {
    exportJobError.value = error instanceof Error ? error.message : t('logs.exportError');
    exportLoading.value = false;
  }
}

function applyParsingStatus(result: Record<string, any>) {
  ipParsing.value = Boolean(result.ip_parsing);
  ipParsingProgress.value = ipParsing.value ? normalizeProgress(result.ip_parsing_progress) : null;
  ipParsingEstimatedRemainingSeconds.value = ipParsing.value
    ? normalizeSeconds(result.ip_parsing_estimated_remaining_seconds)
    : null;
  ipGeoParsing.value = Boolean(result.ip_geo_parsing);
  ipGeoPending.value = Boolean(result.ip_geo_pending);
  ipGeoProgress.value = ipGeoParsing.value || ipGeoPending.value
    ? normalizeProgress(result.ip_geo_progress)
    : null;
  ipGeoEstimatedRemainingSeconds.value = ipGeoParsing.value || ipGeoPending.value
    ? normalizeSeconds(result.ip_geo_estimated_remaining_seconds)
    : null;
  parsingPending.value = Boolean(result.parsing_pending);
  parsingPendingProgress.value = parsingPending.value
    ? normalizeProgress(result.parsing_pending_progress)
    : null;
}

function statusColor(statusCode: number | string) {
  if (typeof statusCode !== 'number') {
    return 'inherit';
  }
  if (statusCode >= 400) {
    return 'var(--error-color)';
  }
  if (statusCode >= 300) {
    return 'var(--warning-color)';
  }
  return 'var(--success-color)';
}

function openLogDetail(event: LogRowClickEvent) {
  const target = event?.originalEvent?.target;
  if (target instanceof HTMLElement && target.closest('.p-column-resizer')) {
    return;
  }
  selectedLog.value = event.data;
  logDetailVisible.value = true;
}

function resetIPGeoEditorState() {
  ipGeoEditorError.value = '';
  ipGeoFormDomestic.value = '';
  ipGeoFormGlobal.value = '';
  ipGeoFormNote.value = '';
  ipGeoOverrideActive.value = false;
  ipGeoOverrideUpdatedAt.value = '';
  ipGeoEditorLoading.value = false;
  ipGeoEditorSaving.value = false;
  ipGeoEditorResetting.value = false;
}

function formatEditorTimestamp(value?: string) {
  if (!value) {
    return '';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

function applyIPGeoOverridePayload(payload: IPGeoOverrideResponse) {
  const domestic = (
    payload.override?.domestic ||
    (payload.source !== 'pending' ? payload.domestic : '') ||
    selectedLog.value?.domesticLocation ||
    ''
  ).trim();
  const global = (
    payload.override?.global ||
    (payload.source !== 'pending' ? payload.global : '') ||
    selectedLog.value?.globalLocation ||
    ''
  ).trim();
  ipGeoFormDomestic.value = domestic;
  ipGeoFormGlobal.value = global;
  ipGeoFormNote.value = (payload.override?.note || payload.note || '').trim();
  ipGeoOverrideActive.value = Boolean(payload.overridden);
  ipGeoOverrideUpdatedAt.value = formatEditorTimestamp(payload.override?.updated_at);
}

function updateSelectedLogLocation(domestic: string, global: string, note = '', overridden = false) {
  if (!selectedLog.value) {
    return;
  }
  const domesticTrimmed = domestic.trim();
  const globalTrimmed = global.trim();
  const locationRaw = domesticTrimmed || globalTrimmed;
  selectedLog.value = {
    ...selectedLog.value,
    domesticLocation: domesticTrimmed,
    globalLocation: globalTrimmed,
    locationNote: note.trim(),
    locationOverrideActive: overridden,
    location: formatLocationLabel(locationRaw, currentLocale.value, t) || t('common.none'),
  };
}

async function openIPGeoEditor() {
  if (!selectedLog.value?.ip || selectedLog.value.ip === t('common.none')) {
    return;
  }
  resetIPGeoEditorState();
  ipGeoEditorVisible.value = true;
  ipGeoEditorLoading.value = true;
  ipGeoFormDomestic.value = selectedLog.value.domesticLocation || '';
  ipGeoFormGlobal.value = selectedLog.value.globalLocation || '';
  ipGeoFormNote.value = selectedLog.value.locationNote || '';
  ipGeoOverrideActive.value = selectedLog.value.locationOverrideActive;
  try {
    const payload = await fetchIPGeoOverride(selectedLog.value.ip);
    applyIPGeoOverridePayload(payload);
  } catch (error) {
    ipGeoEditorError.value = error instanceof Error ? error.message : t('common.requestFailed');
  } finally {
    ipGeoEditorLoading.value = false;
  }
}

async function handleSaveIPGeoOverride() {
  if (!selectedLog.value?.ip || ipGeoEditorSaving.value) {
    return;
  }
  const domestic = ipGeoFormDomestic.value.trim();
  const global = ipGeoFormGlobal.value.trim();
  if (!domestic || !global) {
    ipGeoEditorError.value = t('logs.locationRequired');
    return;
  }
  ipGeoEditorSaving.value = true;
  ipGeoEditorError.value = '';
  try {
    const result = await saveIPGeoOverride({
      ip: selectedLog.value.ip,
      domestic,
      global,
      note: ipGeoFormNote.value.trim(),
    });
    updateSelectedLogLocation(result.domestic, result.global, result.note || '', true);
    ipGeoOverrideActive.value = true;
    ipGeoEditorVisible.value = false;
    await loadLogs();
  } catch (error) {
    ipGeoEditorError.value = error instanceof Error ? error.message : t('common.requestFailed');
  } finally {
    ipGeoEditorSaving.value = false;
  }
}

async function handleResetIPGeoOverride() {
  if (!selectedLog.value?.ip || ipGeoEditorResetting.value) {
    return;
  }
  ipGeoEditorResetting.value = true;
  ipGeoEditorError.value = '';
  try {
    const result = await resetIPGeoOverride(selectedLog.value.ip);
    updateSelectedLogLocation(result.domestic, result.global, '', false);
    ipGeoOverrideActive.value = false;
    ipGeoEditorVisible.value = false;
    await loadLogs();
  } catch (error) {
    ipGeoEditorError.value = error instanceof Error ? error.message : t('common.requestFailed');
  } finally {
    ipGeoEditorResetting.value = false;
  }
}

const logs = computed(() => {
  const emptyLabel = t('common.none');
  return rawLogs.value.map((log) => {
    const time = log.time || emptyLabel;
    const ip = log.ip || emptyLabel;
    const domesticLocation = String(log.domestic_location || '').trim();
    const globalLocation = String(log.global_location || '').trim();
    const locationRaw = domesticLocation || globalLocation || '';
    const location = formatLocationLabel(locationRaw, currentLocale.value, t) || emptyLabel;
    const method = log.method || '';
    const url = log.url || '';
    const requestText = `${method} ${url}`.trim() || emptyLabel;
    const statusCode = log.status_code ?? emptyLabel;
    const bytesSent = Number(log.bytes_sent) || 0;
    const requestLength = Number(log.request_length) || 0;
    const requestTimeMs = Number(log.request_time_ms) || 0;
    const upstreamTimeMs = Number(log.upstream_response_time_ms) || 0;
    const refererRaw = log.referer ?? '';
    const referer = formatRefererLabel(refererRaw, currentLocale.value, t) || emptyLabel;
    const host = String(log.host || '').trim() || emptyLabel;
    const requestId = String(log.request_id || '').trim() || emptyLabel;
    const upstreamAddr = String(log.upstream_addr || '').trim() || emptyLabel;
    const browserRaw = log.user_browser ?? '';
    const browser = formatBrowserLabel(browserRaw, t) || emptyLabel;
    const osRaw = log.user_os ?? '';
    const os = formatOSLabel(osRaw, t) || emptyLabel;
    const deviceRaw = log.user_device ?? '';
    const device = formatDeviceLabel(deviceRaw, t) || emptyLabel;
    const pageview = Boolean(log.pageview_flag);
    return {
      time,
      ip,
      location,
      request: requestText,
      statusCode,
      trafficText: formatTraffic(bytesSent),
      trafficTitle: formatBytesTitle(bytesSent),
      requestTimeText: formatMilliseconds(requestTimeMs),
      requestTimeTitle: requestTimeMs > 0 ? `${requestTimeMs} ms` : emptyLabel,
      requestSizeText: requestLength > 0 ? formatTraffic(requestLength) : emptyLabel,
      requestSizeTitle: requestLength > 0 ? formatBytesTitle(requestLength) : emptyLabel,
      upstreamTimeText: formatMilliseconds(upstreamTimeMs),
      upstreamTimeTitle: upstreamTimeMs > 0 ? `${upstreamTimeMs} ms` : emptyLabel,
      host,
      requestId,
      upstreamAddr,
      domesticLocation,
      globalLocation,
      locationNote: '',
      locationOverrideActive: false,
      referer,
      browser,
      os,
      device,
      pageview,
    }
  });
});

onMounted(() => {
  initPreferences();
  applyRouteOverrides();
  loadWebsites();
});

onUnmounted(() => {
  clearLocationFilterDebounce();
  stopProgressPolling();
  stopExportPolling();
  stopExportHistoryPolling();
  unbindExportHistoryScroll();
});

watch(exportDialogVisible, (visible) => {
  if (visible) {
    refreshExportJobs();
    setTimeout(() => bindExportHistoryScroll(), 0);
  } else {
    stopExportHistoryPolling();
    unbindExportHistoryScroll();
  }
});

watch(logDetailVisible, (visible) => {
  if (!visible) {
    ipGeoEditorVisible.value = false;
  }
});

watch(dateRangePreset, (preset) => {
  if (updatingDatePreset) {
    return;
  }
  updatingDatePreset = true;
  applyDatePreset(preset);
  updatingDatePreset = false;
  saveUserPreference('logsDatePreset', dateRangePreset.value || '');
});

watch(dateRange, (range) => {
  if (updatingDateRange) {
    return;
  }
  updatingDateRange = true;
  const [start, end] = Array.isArray(range) ? range : [];
  if (start) {
    timeStart.value = formatDateTimeValue(start);
  } else {
    timeStart.value = '';
  }
  if (end) {
    timeEnd.value = formatDateTimeValue(end);
  } else {
    timeEnd.value = '';
  }
  updatingDateRange = false;
});

watch([timeStart, timeEnd], ([start, end]) => {
  if (!updatingDatePreset) {
    if (!start && !end) {
      dateRangePreset.value = 'all';
    } else if (dateRangePreset.value !== 'custom') {
      dateRangePreset.value = 'custom';
    }
  }
  if (!updatingDateRange) {
    updatingDateRange = true;
    dateRange.value = buildDateRangeFromTime(start || '', end || '');
    updatingDateRange = false;
  }
});

watch(currentWebsiteId, (value) => {
  if (value) {
    saveUserPreference('selectedWebsite', value);
  }
  stopExportPolling();
  exportLoading.value = false;
  exportJob.value = null;
  exportJobError.value = '';
  currentPage.value = 1;
  loadLogs();
});

watch([ipParsing, parsingPending, ipGeoParsing, ipGeoPending, currentWebsiteId], ([ipActive, pendingActive, geoActive, geoPendingActive, websiteId], prev) => {
  if (!websiteId) {
    stopProgressPolling();
    return;
  }
  const wasActive = Array.isArray(prev) && Boolean(prev[0] || prev[1] || prev[2] || prev[3]);
  const isActive = Boolean(ipActive || pendingActive || geoActive || geoPendingActive);
  if (ipActive || pendingActive || geoActive || geoPendingActive) {
    startProgressPolling();
    refreshParsingStatus();
  } else {
    stopProgressPolling(wasActive);
  }
});

watch(locationFilter, (value) => {
  saveUserPreference('logsLocationFilter', value || '');
  if (suppressLocationFilterWatch) {
    suppressLocationFilterWatch = false;
    clearLocationFilterDebounce();
    return;
  }
  if (isLocationFilterComposing.value) {
    return;
  }
  scheduleLocationFilterSearch();
});

watch([sortField, sortOrder, pageSize, excludeInternal, pageviewOnly, excludeSpider, excludeForeign, statusCodeFilter, timeStart, timeEnd], () => {
  saveUserPreference('logsSortField', sortField.value);
  saveUserPreference('logsSortOrder', sortOrder.value);
  saveUserPreference('logsPageSize', String(pageSize.value));
  saveUserPreference('logsExcludeInternal', excludeInternal.value ? 'true' : 'false');
  saveUserPreference('logsPageviewOnly', pageviewOnly.value ? 'true' : 'false');
  saveUserPreference('logsExcludeSpider', excludeSpider.value ? 'true' : 'false');
  saveUserPreference('logsExcludeForeign', excludeForeign.value ? 'true' : 'false');
  saveUserPreference('logsStatusCode', statusCodeFilter.value || '');
  saveUserPreference('logsTimeStart', timeStart.value || '');
  saveUserPreference('logsTimeEnd', timeEnd.value || '');
  currentPage.value = 1;
  loadLogs();
});

function initPreferences() {
  setLocationFilterSilently(getUserPreference('logsLocationFilter', ''));
  excludeInternal.value = getUserPreference('logsExcludeInternal', 'false') === 'true';
  pageviewOnly.value = getUserPreference('logsPageviewOnly', 'false') === 'true';
  excludeSpider.value = getUserPreference('logsExcludeSpider', 'false') === 'true';
  excludeForeign.value = getUserPreference('logsExcludeForeign', 'false') === 'true';
  statusCodeFilter.value = getUserPreference('logsStatusCode', '');
  timeStart.value = getUserPreference('logsTimeStart', '');
  timeEnd.value = getUserPreference('logsTimeEnd', '');
  const savedPreset = getUserPreference('logsDatePreset', '');
  const hasSavedRange = Boolean(timeStart.value || timeEnd.value);
  if (savedPreset && !(savedPreset === 'all' && !hasSavedRange)) {
    dateRangePreset.value = savedPreset;
  } else if (!hasSavedRange) {
    dateRangePreset.value = 'last7Days';
  } else {
    dateRangePreset.value = 'custom';
  }
}

function applyRouteOverrides() {
  const websiteID = getRouteQueryValue('id');
  if (websiteID) {
    routeWebsiteIdOverride.value = websiteID;
  }

  const ipFilter = getRouteQueryValue('ipFilter');
  if (ipFilter) {
    searchInput.value = ipFilter;
    searchFilter.value = ipFilter;
  }

  const locationFilterValue = getRouteQueryValue('locationFilter');
  if (locationFilterValue) {
    setLocationFilterSilently(locationFilterValue);
  }

  const keywordFilter = getRouteQueryValue('filter');
  if (keywordFilter) {
    searchInput.value = keywordFilter;
    searchFilter.value = keywordFilter;
  }

  const statusClass = getRouteQueryValue('statusClass');
  if (statusClass) {
    statusCodeFilter.value = statusClass;
  }

  const sortFieldOverride = getRouteQueryValue('sortField');
  if (sortFieldOverride && sortFieldOptions.value.some((item) => item.value === sortFieldOverride)) {
    sortField.value = sortFieldOverride;
  }

  const sortOrderOverride = getRouteQueryValue('sortOrder');
  if (sortOrderOverride === 'asc' || sortOrderOverride === 'desc') {
    sortOrder.value = sortOrderOverride;
  }

  const start = getRouteQueryValue('timeStart');
  const end = getRouteQueryValue('timeEnd');
  if (isDateTimeValue(start) || isDateTimeValue(end)) {
    timeStart.value = isDateTimeValue(start) ? start : '';
    timeEnd.value = isDateTimeValue(end) ? end : '';
    dateRangePreset.value = 'custom';
  }
}

async function loadWebsites() {
  websitesLoading.value = true;
  try {
    const data = await fetchWebsites();
    websites.value = data || [];
    const saved = getUserPreference('selectedWebsite', '');
    if (routeWebsiteIdOverride.value && websites.value.find((site) => site.id === routeWebsiteIdOverride.value)) {
      currentWebsiteId.value = routeWebsiteIdOverride.value;
    } else if (saved && websites.value.find((site) => site.id === saved)) {
      currentWebsiteId.value = saved;
    } else if (websites.value.length > 0) {
      currentWebsiteId.value = websites.value[0].id;
    } else {
      currentWebsiteId.value = '';
    }
    maybeShowMigrationDialog();
  } catch (error) {
    console.error('初始化网站失败:', error);
    websites.value = [];
    currentWebsiteId.value = '';
  } finally {
    websitesLoading.value = false;
  }
}

async function loadLogs() {
  if (!currentWebsiteId.value) {
    return;
  }
  clearLocationFilterDebounce();
  loading.value = true;
  try {
    const { statusCode: statusCodeParam, statusClass: statusClassParam } = resolveStatusParams();
    const ipFilterParam = getIPFilterFromSearch(searchFilter.value) || undefined;
    const locationFilterParam = locationFilter.value.trim() || undefined;
    const timeStartParam = timeStart.value || undefined;
    const timeEndParam = timeEnd.value || undefined;
    const result = await fetchLogs(
      currentWebsiteId.value,
      currentPage.value,
      pageSize.value,
      sortField.value,
      sortOrder.value,
      searchFilter.value,
      undefined,
      statusClassParam,
      statusCodeParam,
      excludeInternal.value,
      ipFilterParam,
      timeStartParam,
      timeEndParam,
      locationFilterParam,
      undefined,
      pageviewOnly.value,
      undefined,
      undefined,
      excludeSpider.value,
      excludeForeign.value
    );
    rawLogs.value = result.logs || [];
    paginationExact.value = result.pagination?.exact !== false;
    hasMore.value = Boolean(result.pagination?.hasMore);
    totalPages.value = paginationExact.value ? result.pagination?.pages || 0 : 0;
    applyParsingStatus(result);
  } catch (error) {
    console.error('加载日志失败:', error);
    rawLogs.value = [];
    totalPages.value = 0;
    hasMore.value = false;
    paginationExact.value = true;
    ipParsing.value = false;
    ipParsingProgress.value = null;
    parsingPending.value = false;
    parsingPendingProgress.value = null;
  } finally {
    loading.value = false;
  }
}

async function refreshParsingStatus() {
  if (!currentWebsiteId.value || progressPollInFlight || loading.value) {
    return;
  }
  progressPollInFlight = true;
  try {
    const { statusCode: statusCodeParam, statusClass: statusClassParam } = resolveStatusParams();
    const ipFilterParam = getIPFilterFromSearch(searchFilter.value) || undefined;
    const locationFilterParam = locationFilter.value.trim() || undefined;
    const timeStartParam = timeStart.value || undefined;
    const timeEndParam = timeEnd.value || undefined;
    const result = await fetchLogs(
      currentWebsiteId.value,
      currentPage.value,
      pageSize.value,
      sortField.value,
      sortOrder.value,
      searchFilter.value,
      undefined,
      statusClassParam,
      statusCodeParam,
      excludeInternal.value,
      ipFilterParam,
      timeStartParam,
      timeEndParam,
      locationFilterParam,
      undefined,
      pageviewOnly.value,
      undefined,
      undefined,
      excludeSpider.value,
      excludeForeign.value
    );
    applyParsingStatus(result);
  } catch (error) {
    console.debug('刷新解析进度失败:', error);
  } finally {
    progressPollInFlight = false;
  }
}

function startProgressPolling() {
  if (progressPollTimer) {
    return;
  }
  progressPollTimer = setInterval(() => {
    if (ipParsing.value || parsingPending.value || ipGeoParsing.value || ipGeoPending.value) {
      refreshParsingStatus();
    }
  }, progressPollIntervalMs);
}

function stopProgressPolling(refresh = false) {
  if (progressPollTimer) {
    clearInterval(progressPollTimer);
    progressPollTimer = null;
  }
  if (refresh) {
    loadLogs();
  }
}

function applySearch() {
  searchFilter.value = searchInput.value.trim();
  currentPage.value = 1;
  loadLogs();
}

function openReparseDialog() {
  reparseError.value = '';
  if (isDemoMode.value) {
    openDemoBlockedDialog('logs.reparseBlocked');
    return;
  }
  reparseMode.value = 'full';
  const now = new Date();
  reparseRangeStartDate.value = parseDateFromTime(timeStart.value) || startOfDay(now);
  reparseRangeEndDate.value = parseDateFromTime(timeEnd.value) || now;
  reparseDialogMode.value = 'confirm';
  reparseDialogVisible.value = true;
}

function openDemoBlockedDialog(messageKey: 'logs.reparseBlocked' | 'logs.exportBlocked') {
  reparseDialogBlockedMessageKey.value = messageKey;
  reparseDialogMode.value = 'blocked';
  reparseDialogVisible.value = true;
}

function maybeShowMigrationDialog() {
  const acknowledged = getUserPreference(migrationAckKey, 'false') === 'true';
  if (
    acknowledged ||
    isDemoMode.value ||
    websites.value.length === 0 ||
    !migrationRequired?.value
  ) {
    return;
  }
  migrationError.value = '';
  migrationDialogVisible.value = true;
}

async function confirmMigration() {
  if (migrationLoading.value) {
    return;
  }
  if (websites.value.length === 0) {
    migrationError.value = t('logs.migrationError');
    return;
  }
  migrationLoading.value = true;
  migrationError.value = '';
  try {
    await reparseAllLogs();
    saveUserPreference(migrationAckKey, 'true');
    if (migrationRequired) {
      migrationRequired.value = false;
    }
    migrationDialogVisible.value = false;
    currentPage.value = 1;
    await loadLogs();
  } catch (error) {
    if (error instanceof Error) {
      migrationError.value = error.message;
    } else {
      migrationError.value = t('logs.migrationError');
    }
  } finally {
    migrationLoading.value = false;
  }
}

async function confirmReparse() {
  if (reparseDialogMode.value !== 'confirm') {
    reparseDialogVisible.value = false;
    return;
  }
  if (!currentWebsiteId.value) {
    return;
  }
  reparseLoading.value = true;
  reparseError.value = '';
  try {
    if (reparseMode.value === 'range') {
      const startAt = reparseRangeStartDate.value ? formatDateTimeValue(reparseRangeStartDate.value) : '';
      const endAt = reparseRangeEndDate.value ? formatDateTimeValue(reparseRangeEndDate.value) : '';
      if (!startAt || !endAt) {
        reparseError.value = t('logs.reparseRangeRequired');
        return;
      }
      if (startAt >= endAt) {
        reparseError.value = t('logs.reparseRangeInvalid');
        return;
      }
      await reparseLogs(currentWebsiteId.value, {
        mode: 'range',
        startAt,
        endAt,
      });
    } else {
      await reparseLogs(currentWebsiteId.value, { mode: 'full' });
    }
    reparseDialogVisible.value = false;
    currentPage.value = 1;
    await loadLogs();
  } catch (error) {
    if (error instanceof Error) {
      reparseError.value = error.message;
    } else {
      reparseError.value = t('logs.reparseError');
    }
  } finally {
    reparseLoading.value = false;
  }
}

function jumpToPage() {
  if (!paginationExact.value) {
    return;
  }
  const pageNum = pageJump.value ?? 1;
  if (!Number.isFinite(pageNum) || pageNum < 1 || pageNum > totalPages.value) {
    return;
  }
  currentPage.value = Math.trunc(pageNum);
  loadLogs();
}

function prevPage() {
  if (currentPage.value > 1) {
    currentPage.value -= 1;
    loadLogs();
  }
}

function nextPage() {
  if ((paginationExact.value && currentPage.value < totalPages.value) || (!paginationExact.value && hasMore.value)) {
    currentPage.value += 1;
    loadLogs();
  }
}

</script>

<style scoped lang="scss">
.logs-layout {
  height: calc(100vh - 32px - 24px);
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.logs-control-box {
  padding: 18px 20px;
  margin-bottom: 18px;
  position: relative;
  z-index: 30;
  --control-height: 40px;
}

.logs-layout .card:hover {
  transform: none;
  box-shadow: var(--shadow);
  border-color: var(--border);
}

.logs-ip-notice {
  padding: 10px 14px;
  margin-bottom: 18px;
  border-radius: var(--radius-sm);
  background: rgba(var(--primary-color-rgb), 0.12);
  color: var(--accent-color);
  font-size: 13px;
  font-weight: 500;
}

.logs-control-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.control-row {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  justify-content: flex-start;
  gap: 12px;
}

.logs-control-box :deep(.p-button),
.logs-control-box :deep(.p-inputtext),
.logs-control-box :deep(.p-inputnumber-input),
.logs-control-box :deep(.p-dropdown),
.logs-control-box :deep(.p-select) {
  height: var(--control-height);
}

.logs-control-box :deep(.p-dropdown-label),
.logs-control-box :deep(.p-select-label) {
  display: flex;
  align-items: center;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 0 0 auto;
  width: 100%;
  min-width: 320px;
  flex-wrap: wrap;
}

.search-input {
  flex: 1 1 380px;
  min-width: 240px;
  max-width: none;
}

.search-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: var(--control-height);
  padding: 0;
  margin-left: auto;
  border: 0;
  background: transparent;
  box-shadow: none;
  box-sizing: border-box;
  flex: 0 0 auto;
}

.search-btn {
  font-weight: 700;
  border-radius: var(--radius-sm);
  height: var(--control-height);
  min-width: 82px;
  padding: 0 14px;
  box-shadow: 0 8px 16px rgba(var(--primary-color-rgb), 0.18);
}

.action-btn {
  border-radius: var(--radius-sm);
  height: var(--control-height);
  min-width: 0;
  padding: 0 12px;
  font-weight: 700;
  box-shadow: none;
}

.filter-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.filter-collapse-enter-active,
.filter-collapse-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.filter-collapse-enter-from,
.filter-collapse-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}

.filter-row-fields {
  gap: 16px;
  margin-left: 0;
  justify-content: space-between;
  flex: 0 0 auto;
  width: 100%;
  min-width: 0;
}

.filter-row-left,
.filter-row-right {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.filter-row-left {
  flex: 1 1 520px;
  min-width: 0;
}

.filter-row-right {
  flex: 0 0 auto;
  margin-left: auto;
}

.filter-row-toggles {
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  background: var(--panel-muted);
  border: 1px solid var(--border);
  gap: 10px;
}

.filter-input-container {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1 1 260px;
  min-width: 220px;
}

.filter-input-container label {
  font-size: 12px;
  color: var(--muted);
  font-weight: 600;
  white-space: nowrap;
}

.filter-text-input {
  flex: 1 1 auto;
  min-width: 0;
  font-size: 12px;
}

.filter-toggle-container {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  border-radius: var(--radius-xs);
  background: var(--panel-muted);
  border: 1px solid var(--border);
  font-size: 12px;
  font-weight: 600;
  color: var(--text);
  flex: 0 0 auto;
  white-space: nowrap;
  min-height: var(--control-height);
}

.status-code-container {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
  flex-wrap: wrap;
  row-gap: 6px;
  white-space: normal;
}

.status-code-container label,
.sort-field-container label,
.sort-order-container label,
.page-size-container label,
.date-range-container label {
  font-size: 12px;
  color: var(--muted);
  font-weight: 600;
}

.status-code-input {
  width: 120px;
}

.status-code-input :deep(.p-inputtext) {
  font-size: 12px;
}

.status-code-quick {
  display: none;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  row-gap: 6px;
}

.status-code-container:focus-within .status-code-quick {
  display: flex;
}

.status-quick-btn {
  border: 1px solid var(--border);
  background: var(--panel);
  color: var(--text);
  border-radius: var(--radius-xs);
  padding: 6px 10px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}

.status-quick-btn:hover {
  border-color: rgba(var(--primary-color-rgb), 0.5);
  color: var(--accent-color);
}

.status-quick-btn.active {
  border-color: rgba(var(--primary-color-rgb), 0.8);
  background: rgba(var(--primary-color-rgb), 0.12);
  color: var(--accent-color);
}

.sort-field-container,
.sort-order-container,
.page-size-container {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
  white-space: nowrap;
}

.date-range-container {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
  flex-wrap: wrap;
  row-gap: 6px;
  white-space: normal;
}

.date-range-inputs {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  row-gap: 6px;
}

.date-range-select {
  min-width: 120px;
}

.date-range-select :deep(.p-dropdown-label),
.date-range-select :deep(.p-select-label) {
  font-size: 12px;
}

.date-range-picker {
  width: 300px;
}

.date-range-picker :deep(.p-inputtext) {
  width: 100%;
  font-size: 12px;
}

.sort-select {
  min-width: 120px;
}

.sort-select :deep(.p-dropdown),
.sort-select :deep(.p-select) {
  font-size: 12px;
}

.sort-select :deep(.p-dropdown-label),
.sort-select :deep(.p-select-label) {
  font-size: 12px;
}

.logs-table-box {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  position: relative;
  z-index: 1;
}

:global(.logs-page) .page-header {
  z-index: 60;
}

.logs-table-wrapper {
  --logs-table-head-offset: 52px;
  overflow: hidden;
  width: 100%;
  flex: 1;
  min-height: 0;
  position: relative;
  display: flex;
  flex-direction: column;
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  background: var(--panel);
}

.logs-table {
  background: transparent;
  border: none;
  flex: 1;
  min-height: 0;
}

.logs-table :deep(.p-datatable-wrapper) {
  flex: 1;
  min-height: 0;
}

.logs-table :deep(.p-datatable-table-container) {
  flex: 1;
  min-height: 0;
}

.logs-table :deep(.p-datatable-table) {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
  table-layout: fixed;
}

.logs-table :deep(.p-datatable-thead > tr > th),
.logs-table :deep(.p-datatable-tbody > tr > td) {
  padding: 8px 10px;
  text-align: left;
  border-bottom: 1px solid var(--border);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.logs-table :deep(.p-datatable-thead > tr > th) {
  position: sticky;
  top: 0;
  background-color: var(--panel);
  z-index: 2;
  font-weight: 600;
}

.logs-table :deep(.p-datatable-tbody > tr.p-row-odd) {
  background-color: var(--row-alt-bg);
}

.logs-table :deep(.p-datatable-tbody > tr) {
  cursor: pointer;
}

.logs-table :deep(.p-datatable-tbody > tr:hover) {
  background-color: rgba(var(--primary-color-rgb), 0.08);
}

.logs-table :deep(.p-column-resizer) {
  cursor: col-resize;
  width: 6px;
}

.logs-table :deep(.p-column-resizer:hover) {
  background-color: rgba(var(--primary-color-rgb), 0.2);
}

.logs-table-overlay {
  position: absolute;
  inset: 0;
  z-index: 5;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: inherit;
  background: color-mix(in srgb, var(--panel) 75%, transparent);
  backdrop-filter: blur(1px);
}

:global(body.dark-mode) .logs-table-overlay {
  background: color-mix(in srgb, var(--panel) 70%, transparent);
}

.logs-table-overlay-card {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  border-radius: var(--radius-pill);
  background: var(--panel);
  border: 1px solid var(--border);
  box-shadow: var(--shadow-soft);
  color: var(--muted);
  font-size: 13px;
  font-weight: 600;
}

.logs-table-overlay-spinner {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  border: 2px solid rgba(var(--primary-color-rgb), 0.25);
  border-top-color: var(--primary);
  animation: logs-spin 0.8s linear infinite;
}

.logs-empty-overlay {
  position: absolute;
  inset: var(--logs-table-head-offset) 0 0;
  z-index: 3;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 28px 24px 32px;
  pointer-events: none;
}

.logs-empty-overlay::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, rgba(247, 249, 255, 0.8), rgba(255, 255, 255, 0.92));
}

.logs-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 32px 20px;
  text-align: center;
}

.logs-empty-state-overlay {
  position: relative;
  z-index: 1;
  width: min(380px, 100%);
  padding: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  pointer-events: auto;
}

.logs-empty-state-icon {
  width: 44px;
  height: 44px;
  border-radius: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  color: var(--primary);
  background: rgba(var(--primary-color-rgb), 0.12);
}

.logs-empty-state-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--text);
}

.logs-empty-state-text {
  max-width: 300px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--muted);
}

.logs-empty-action {
  margin-top: 6px;
}

:global(body.dark-mode) .logs-empty-state-icon {
  color: #8bb8ff;
  background: rgba(var(--primary-color-rgb), 0.16);
}

:global(body.dark-mode) .logs-empty-overlay::before {
  background: linear-gradient(180deg, rgba(15, 23, 42, 0.58), rgba(15, 23, 42, 0.72));
}

:global(body.dark-mode) .logs-empty-state-overlay {
  background: transparent;
  box-shadow: none;
}

:global(body.dark-mode) .logs-empty-state-title {
  color: rgba(241, 245, 249, 0.94);
}

:global(body.dark-mode) .logs-empty-state-text {
  color: rgba(148, 163, 184, 0.92);
}

@keyframes logs-spin {
  to {
    transform: rotate(360deg);
  }
}

.logs-pv-cell {
  text-align: center;
}

.log-detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 18px;
}

.log-detail-item {
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  background: var(--panel-muted);
  border: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.log-detail-label {
  font-size: 12px;
  color: var(--muted);
  font-weight: 600;
}

.log-detail-value {
  font-size: 13px;
  color: var(--text);
  word-break: break-all;
}

.log-detail-action {
  align-self: flex-start;
  margin-top: 6px;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--accent-color);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}

.log-detail-action:disabled {
  color: var(--muted);
  cursor: not-allowed;
}

.ip-geo-editor-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.ip-geo-editor-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.ip-geo-editor-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.ip-geo-editor-field label {
  font-size: 12px;
  font-weight: 600;
  color: var(--muted);
}

.ip-geo-editor-textarea {
  width: 100%;
  min-height: 96px;
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border);
  background: var(--panel);
  color: var(--text);
  resize: vertical;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.ip-geo-editor-textarea:focus {
  outline: none;
  border-color: rgba(var(--primary-color-rgb), 0.55);
  box-shadow: 0 0 0 3px rgba(var(--primary-color-rgb), 0.14);
}

.pagination-box {
  padding: 15px 20px;
  margin-top: 15px;
}

.pagination-controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.pagination-center {
  display: flex;
  align-items: center;
  gap: 12px;
}

.page-info {
  font-size: 12px;
  color: var(--muted);
}

.page-jump {
  display: flex;
  align-items: center;
  gap: 8px;
}

.page-jump-input {
  width: 120px;
}

.page-btn {
  border-radius: var(--radius-xs);
}

.advanced-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: var(--control-height);
  padding: 0 12px;
  border-radius: var(--radius-sm);
  border: 1px dashed rgba(var(--primary-color-rgb), 0.3);
  background: rgba(var(--primary-color-rgb), 0.06);
  color: var(--accent-color);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: border-color 0.2s ease, background 0.2s ease, color 0.2s ease;
}

.advanced-toggle:hover {
  border-color: rgba(var(--primary-color-rgb), 0.5);
  background: rgba(var(--primary-color-rgb), 0.12);
}

.reparse-btn {
  white-space: nowrap;
  color: var(--error-color);
  border-color: rgba(239, 68, 68, 0.28);
  background: rgba(239, 68, 68, 0.04);
}

.reparse-btn:not(:disabled):hover {
  border-color: rgba(239, 68, 68, 0.44);
  background: rgba(239, 68, 68, 0.08);
}

.export-btn {
  white-space: nowrap;
  color: #16a34a;
  border-color: rgba(22, 163, 74, 0.28);
  background: rgba(34, 197, 94, 0.06);
  transition: background 0.2s ease, border-color 0.2s ease, color 0.2s ease;
}

.export-btn:not(:disabled):hover {
  color: #15803d;
  border-color: rgba(22, 163, 74, 0.44);
  background: rgba(34, 197, 94, 0.1);
}

body.dark-mode .search-actions {
  background: transparent;
  box-shadow: none;
}

body.dark-mode .search-btn {
  border-color: rgba(96, 165, 250, 0.6);
  background: linear-gradient(180deg, #2563eb 0%, #1d4ed8 100%);
  color: #eff6ff;
  box-shadow: 0 10px 20px rgba(37, 99, 235, 0.28);
}

body.dark-mode .search-btn:not(:disabled):hover {
  border-color: rgba(147, 197, 253, 0.72);
  background: linear-gradient(180deg, #3b82f6 0%, #2563eb 100%);
  color: #ffffff;
}

body.dark-mode .reparse-btn {
  color: #f87171;
  border-color: rgba(248, 113, 113, 0.42);
  background: rgba(248, 113, 113, 0.1);
}

body.dark-mode .reparse-btn:not(:disabled):hover {
  color: #fecaca;
  border-color: rgba(248, 113, 113, 0.68);
  background: rgba(248, 113, 113, 0.16);
}

body.dark-mode .export-btn {
  color: #4ade80;
  border-color: rgba(74, 222, 128, 0.42);
  background: rgba(34, 197, 94, 0.1);
}

body.dark-mode .export-btn:not(:disabled):hover {
  color: #bbf7d0;
  border-color: rgba(74, 222, 128, 0.68);
  background: rgba(34, 197, 94, 0.16);
}

body.dark-mode .action-btn:disabled {
  color: rgba(148, 163, 184, 0.62);
  border-color: rgba(148, 163, 184, 0.18);
  background: rgba(15, 23, 42, 0.42);
}

.logs-reparse-dialog :deep(.p-dialog-content) {
  min-height: 280px;
}

.reparse-dialog-body {
  display: flex;
  flex-direction: column;
  min-height: 100%;
  gap: 10px;
  font-size: 14px;
  color: var(--text);
}

.reparse-dialog-note {
  font-size: 13px;
  color: var(--muted);
}

.reparse-mode-switch {
  display: inline-flex;
  gap: 8px;
  padding: 4px;
  border-radius: 999px;
  background: rgba(var(--primary-color-rgb), 0.08);
  align-self: flex-start;
}

.reparse-mode-btn {
  border: none;
  background: transparent;
  color: var(--muted);
  padding: 8px 14px;
  border-radius: 999px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s ease, color 0.2s ease;
}

.reparse-mode-btn.active {
  background: var(--panel);
  color: var(--text);
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.08);
}

.reparse-mode-btn:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.reparse-range-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.reparse-range-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.reparse-range-field label {
  font-size: 12px;
  font-weight: 600;
  color: var(--muted);
}

.reparse-range-picker {
  width: 100%;
}

.reparse-range-picker :deep(.p-inputtext) {
  width: 100%;
  font-size: 13px;
}

.export-progress {
  height: 8px;
  border-radius: var(--radius-pill);
  background: rgba(59, 130, 246, 0.15);
  overflow: hidden;
}

.export-progress-bar {
  height: 100%;
  background: rgba(59, 130, 246, 0.9);
  transition: width 0.3s ease;
}

.export-status {
  margin-left: 6px;
  color: var(--muted);
  font-size: 13px;
}

.export-table {
  font-size: 13px;
}

.reparse-dialog-error {
  font-size: 13px;
  color: var(--error-color);
  font-weight: 600;
}

@media (max-width: 1800px) {
  .control-row {
    flex-direction: column;
    align-items: stretch;
    justify-content: flex-start;
  }

  .search-box {
    width: 100%;
    flex-wrap: wrap;
    min-width: 0;
    flex: 0 0 auto;
  }

  .search-input {
    flex-basis: 320px;
    min-width: 200px;
  }

  .search-actions {
    margin-left: 0;
  }

  .filter-row-fields {
    margin-left: 0;
    justify-content: space-between;
    flex: 0 0 auto;
    width: 100%;
  }
}

@media (max-width: 900px) {
  .logs-control-content {
    align-items: stretch;
  }

  .control-row {
    align-items: flex-start;
  }

  .search-box {
    width: 100%;
    flex-wrap: wrap;
  }

  .search-actions {
    width: 100%;
    margin-left: 0;
    justify-content: flex-start;
  }

  .filter-row-fields {
    margin-left: 0;
    flex-direction: column;
    align-items: stretch;
    justify-content: flex-start;
  }

  .filter-row-left,
  .filter-row-right {
    width: 100%;
    justify-content: flex-start;
  }

  .filter-row-right {
    margin-left: 0;
  }

  .filter-row {
    gap: 10px;
  }

  .pagination-controls {
    flex-direction: column;
    gap: 12px;
  }

  .log-detail-grid {
    grid-template-columns: 1fr;
  }

  .ip-geo-editor-grid {
    grid-template-columns: 1fr;
  }

  .reparse-range-grid {
    grid-template-columns: 1fr;
  }
}
</style>
