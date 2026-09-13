<template>
  <div class="schema-diff-view">
    <div class="schema-diff-view__header">
      <div class="schema-diff-view__summary">
        <el-tag type="success" v-if="addedTables.length">+{{ addedTables.length }} {{ $t('db.teDiffNewTables') }}</el-tag>
        <el-tag type="danger" v-if="removedTables.length">-{{ removedTables.length }} {{ $t('db.teDiffRemovedTables') }}</el-tag>
        <el-tag type="warning" v-if="modifiedTables.length">~{{ modifiedTables.length }} {{ $t('db.teDiffModifiedTables') }}</el-tag>
        <el-tag type="info" v-if="columnChanges">±{{ columnChanges }} {{ $t('db.teDiffColumnChanges') }}</el-tag>
      </div>
      <el-radio-group v-model="viewMode" size="small">
        <el-radio-button label="side">{{ $t('db.teDiffViewSide') }}</el-radio-button>
        <el-radio-button label="unified">{{ $t('db.teDiffViewUnified') }}</el-radio-button>
      </el-radio-group>
    </div>

    <div class="schema-diff-view__content">
      <!-- 新增的表 -->
      <div v-if="addedTables.length" class="schema-diff-view__section">
        <h4 class="is-success">{{ $t('db.teDiffNewTables') }}</h4>
        <div v-for="table in addedTables" :key="table.name" class="schema-diff-view__table-diff">
          <div class="schema-diff-view__table-name">+ {{ table.name }}</div>
          <div class="schema-diff-view__columns">
            <div v-for="col in table.columns" :key="col.name" class="schema-diff-view__col is-added">
              <span class="col-name">+ {{ col.name }}</span>
              <span class="col-type">{{ col.type }}{{ col.length ? `(${col.length})` : '' }}</span>
              <span class="col-attrs">
                <el-tag v-if="col.pri" size="small" type="danger">PK</el-tag>
                <el-tag v-if="col.notNull" size="small">NN</el-tag>
                <el-tag v-if="col.auto_increment" size="small" type="warning">AI</el-tag>
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- 删除的表 -->
      <div v-if="removedTables.length" class="schema-diff-view__section">
        <h4 class="is-danger">{{ $t('db.teDiffRemovedTables') }}</h4>
        <div v-for="table in removedTables" :key="table.name" class="schema-diff-view__table-diff">
          <div class="schema-diff-view__table-name is-removed">- {{ table.name }}</div>
        </div>
      </div>

      <!-- 修改的表 -->
      <div v-if="modifiedTables.length" class="schema-diff-view__section">
        <h4 class="is-warning">{{ $t('db.teDiffModifiedTables') }}</h4>
        <div v-for="mod in modifiedTables" :key="mod.tableName" class="schema-diff-view__table-diff">
          <div class="schema-diff-view__table-name is-modified">~ {{ mod.tableName }}</div>
          <div class="schema-diff-view__changes">
            <div v-for="(change, idx) in mod.changes" :key="idx" class="schema-diff-view__change">
              <el-tag size="small" :type="getChangeType(change.type)">{{ change.type }}</el-tag>
              <span class="change-before" v-if="change.before">{{ change.before }}</span>
              <span class="change-arrow">→</span>
              <span class="change-after" v-if="change.after">{{ change.after }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 列级别变更 -->
      <div v-for="colDiff in diff.columnLevel" :key="colDiff.tableName" class="schema-diff-view__section">
        <h4>{{ colDiff.tableName }} - {{ $t('db.teDiffColumnChangesTitle') }}</h4>
        <div v-for="col in colDiff.added" :key="col.name" class="schema-diff-view__col is-added">
          <span class="col-name">+ {{ col.name }}</span>
          <span class="col-type">{{ col.type }}</span>
        </div>
        <div v-for="col in colDiff.removed" :key="col.name" class="schema-diff-view__col is-removed">
          <span class="col-name">- {{ col.name }}</span>
          <span class="col-type">{{ col.type }}</span>
        </div>
        <div v-for="mod in colDiff.modified" :key="mod.columnName" class="schema-diff-view__col is-modified">
          <span class="col-name">~ {{ mod.columnName }}</span>
          <span class="col-changes">{{ mod.changes.join(', ') }}</span>
        </div>
      </div>

      <el-empty v-if="isEmpty" :description="$t('db.teDiffNoDiff')" :image-size="80" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { SchemaDiff } from '../types/schema';

interface Props {
  diff: SchemaDiff;
}

const props = defineProps<Props>();
const viewMode = ref<'side' | 'unified'>('side');

const addedTables = computed(() => props.diff.tableLevel.added);
const removedTables = computed(() => props.diff.tableLevel.removed);
const modifiedTables = computed(() => props.diff.tableLevel.modified);

const columnChanges = computed(() => {
  return props.diff.columnLevel.reduce((sum, d) => sum + d.added.length + d.removed.length + d.modified.length, 0);
});

const isEmpty = computed(() => {
  return addedTables.value.length === 0 && removedTables.value.length === 0 && modifiedTables.value.length === 0 && columnChanges.value === 0;
});

function getChangeType(type: string): 'success' | 'danger' | 'warning' | 'info' {
  switch (type) {
    case 'COMMENT': return 'info';
    case 'ENGINE': return 'warning';
    case 'CHARSET': return 'warning';
    default: return 'info';
  }
}
</script>

<style scoped lang="scss">
.schema-diff-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  border: 1px solid var(--el-border-color-light);
  border-radius: 4px;
  overflow: hidden;

  &__header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 16px;
    background: var(--el-fill-color-light);
    border-bottom: 1px solid var(--el-border-color-light);
  }

  &__summary {
    display: flex;
    gap: 8px;
  }

  &__content {
    flex: 1;
    overflow: auto;
    padding: 16px;
  }

  &__section {
    margin-bottom: 20px;

    h4 {
      margin: 0 0 12px;
      font-size: 14px;
      &.is-success { color: var(--el-color-success); }
      &.is-danger { color: var(--el-color-danger); }
      &.is-warning { color: var(--el-color-warning); }
    }
  }

  &__table-diff {
    margin-bottom: 12px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 4px;
    overflow: hidden;
  }

  &__table-name {
    padding: 8px 12px;
    font-weight: 600;
    font-size: 13px;
    background: var(--el-color-success-light-9);

    &.is-removed { background: var(--el-color-danger-light-9); }
    &.is-modified { background: var(--el-color-warning-light-9); }
  }

  &__columns {
    padding: 8px 12px;
  }

  &__col {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 4px 8px;
    border-radius: 3px;
    font-size: 13px;
    margin-bottom: 4px;

    &.is-added { background: var(--el-color-success-light-9); }
    &.is-removed { background: var(--el-color-danger-light-9); }
    &.is-modified { background: var(--el-color-warning-light-9); }

    .col-name { font-weight: 500; min-width: 120px; }
    .col-type { color: var(--el-text-color-secondary); }
    .col-attrs { display: flex; gap: 4px; }
    .col-changes { color: var(--el-text-color-regular); font-size: 12px; }
  }

  &__changes {
    padding: 8px 12px;
  }

  &__change {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 0;
    font-size: 13px;

    .change-before { color: var(--el-color-danger); text-decoration: line-through; }
    .change-arrow { color: var(--el-text-color-secondary); }
    .change-after { color: var(--el-color-success); }
  }
}
</style>
