<template>
  <div class="validation-result" v-if="errors.length > 0 || warnings.length > 0">
    <div class="validation-result__header">
      <el-icon :class="isValid ? 'is-success' : 'is-error'">
        <CircleCheck v-if="isValid" />
        <CircleClose v-else />
      </el-icon>
      <span class="validation-result__summary">{{ summary }}</span>
    </div>

    <el-collapse v-model="expandedPanels" class="validation-result__body">
      <el-collapse-item v-if="errors.length > 0" name="errors">
        <template #title>
          <div class="validation-result__panel-title">
            <el-icon class="is-error"><CircleClose /></el-icon>
            <span>{{ $t('db.teValidationErrors') }} ({{ errors.length }})</span>
          </div>
        </template>
        <div class="validation-result__list">
          <div v-for="(error, idx) in errors" :key="idx" class="validation-result__item is-error">
            <div class="validation-result__item-header">
              <el-tag size="small" type="danger">{{ error.code }}</el-tag>
              <span class="validation-result__item-location">
                {{ error.location.type }}: {{ error.location.name }}
                <span v-if="error.location.field">.{{ error.location.field }}</span>
              </span>
            </div>
            <div class="validation-result__item-message">{{ error.message }}</div>
            <div v-if="error.suggestion" class="validation-result__item-suggestion">
              <el-icon><InfoFilled /></el-icon>
              {{ error.suggestion }}
            </div>
          </div>
        </div>
      </el-collapse-item>

      <el-collapse-item v-if="warnings.length > 0" name="warnings">
        <template #title>
          <div class="validation-result__panel-title">
            <el-icon class="is-warning"><Warning /></el-icon>
            <span>{{ $t('db.teValidationWarnings') }} ({{ warnings.length }})</span>
          </div>
        </template>
        <div class="validation-result__list">
          <div v-for="(warning, idx) in warnings" :key="idx" class="validation-result__item is-warning">
            <div class="validation-result__item-header">
              <el-tag size="small" type="warning">{{ warning.code }}</el-tag>
              <span class="validation-result__item-location">
                {{ warning.location.type }}: {{ warning.location.name }}
                <span v-if="warning.location.field">.{{ warning.location.field }}</span>
              </span>
            </div>
            <div class="validation-result__item-message">{{ warning.message }}</div>
            <div v-if="warning.suggestion" class="validation-result__item-suggestion">
              <el-icon><InfoFilled /></el-icon>
              {{ warning.suggestion }}
            </div>
          </div>
        </div>
      </el-collapse-item>
    </el-collapse>
  </div>

  <div class="validation-result validation-result--success" v-else>
    <el-icon class="is-success"><CircleCheck /></el-icon>
    <span>{{ $t('db.teValidationPass') }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { CircleCheck, CircleClose, Warning, InfoFilled } from '@element-plus/icons-vue';
import type { ValidationError } from '../types/schema';

const { t } = useI18n();

interface Props {
  errors: ValidationError[];
  warnings: ValidationError[];
}

const props = defineProps<Props>();

const expandedPanels = ref(['errors', 'warnings']);

const isValid = computed(() => props.errors.length === 0);

const summary = computed(() => {
  if (isValid.value && props.warnings.length === 0) {
    return t('db.teValidationPass');
  }
  const parts = [];
  if (props.errors.length > 0) {
    parts.push(t('db.teValidationSummaryErrors', { count: props.errors.length }));
  }
  if (props.warnings.length > 0) {
    parts.push(t('db.teValidationSummaryWarnings', { count: props.warnings.length }));
  }
  return parts.join(', ');
});
</script>

<style scoped lang="scss">
.validation-result {
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  overflow: hidden;

  &--success {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 16px;
    background: var(--el-color-success-light-9);
    border-color: var(--el-color-success-light-5);
    color: var(--el-color-success);
    font-weight: 500;
  }

  &__header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 16px;
    background: var(--el-fill-color-light);
    border-bottom: 1px solid var(--el-border-color-light);
    font-weight: 600;

    .is-success { color: var(--el-color-success); }
    .is-error { color: var(--el-color-danger); }
  }

  &__body {
    :deep(.el-collapse-item__header) {
      padding: 0 16px;
      height: 40px;
    }
    :deep(.el-collapse-item__content) {
      padding: 0;
    }
  }

  &__panel-title {
    display: flex;
    align-items: center;
    gap: 8px;
    .is-error { color: var(--el-color-danger); }
    .is-warning { color: var(--el-color-warning); }
  }

  &__list {
    padding: 0 16px 12px;
  }

  &__item {
    padding: 10px 12px;
    border-radius: 4px;
    margin-bottom: 8px;

    &.is-error {
      background: var(--el-color-danger-light-9);
      border-left: 3px solid var(--el-color-danger);
    }

    &.is-warning {
      background: var(--el-color-warning-light-9);
      border-left: 3px solid var(--el-color-warning);
    }

    &-header {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 6px;
    }

    &-location {
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }

    &-message {
      font-size: 13px;
      margin-bottom: 4px;
    }

    &-suggestion {
      display: flex;
      align-items: center;
      gap: 4px;
      font-size: 12px;
      color: var(--el-color-primary);
      margin-top: 4px;
    }
  }
}
</style>
