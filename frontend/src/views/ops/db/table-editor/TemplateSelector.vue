<template>
  <div class="template-selector">
    <el-dialog v-model="visible" :title="$t('db.teTemplateSelector')" width="700px" :close-on-click-modal="false" append-to-body>
      <div class="template-selector__search">
        <el-input v-model="searchKeyword" :placeholder="$t('db.teTemplateSearch')" clearable prefix-icon="Search" />
        <el-select v-model="selectedCategory" :placeholder="$t('db.teTemplateCategory')" clearable style="width: 140px">
          <el-option v-for="cat in categories" :key="cat" :label="cat" :value="cat" />
        </el-select>
      </div>

      <div class="template-selector__list">
        <div
          v-for="tpl in filteredTemplates"
          :key="tpl.id"
          class="template-selector__item"
          :class="{ 'is-active': selectedTemplate?.id === tpl.id }"
          @click="selectedTemplate = tpl"
        >
          <div class="template-selector__item-header">
            <span class="template-selector__item-name">{{ tpl.name }}</span>
            <el-tag size="small" type="info">{{ tpl.category }}</el-tag>
          </div>
          <div class="template-selector__item-desc">{{ tpl.description }}</div>
          <div class="template-selector__item-meta">
            <span>{{ $t('db.teTemplateColumns', { count: tpl.columns.length }) }}</span>
            <span>{{ $t('db.teTemplateIndexes', { count: tpl.indexes?.length || 0 }) }}</span>
            <span v-if="tpl.tags.length">{{ tpl.tags.join(', ') }}</span>
          </div>
        </div>
      </div>

      <div v-if="selectedTemplate" class="template-selector__preview">
        <h4>{{ $t('db.teTemplatePreview', { name: selectedTemplate.name }) }}</h4>
        <el-table :data="selectedTemplate.columns" size="small" max-height="200" border>
          <el-table-column prop="name" :label="$t('db.teColumnName')" width="120" />
          <el-table-column prop="type" :label="$t('db.teColumnType')" width="100" />
          <el-table-column prop="length" :label="$t('db.teColumnLength')" width="80" />
          <el-table-column :label="$t('db.teColumnNotNull')" width="150">
            <template #default="{ row }">
              <el-tag v-if="row.isPrimaryKey" size="small" type="danger" style="margin-right:4px">PK</el-tag>
              <el-tag v-if="row.autoIncrement" size="small" type="warning" style="margin-right:4px">AI</el-tag>
              <el-tag v-if="row.notNull" size="small">NN</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="comment" :label="$t('db.teColumnComment')" />
        </el-table>
      </div>

      <template #footer>
        <el-button @click="visible = false">{{ $t('db.teTemplateCancel') }}</el-button>
        <el-button type="primary" @click="handleApply" :disabled="!selectedTemplate">{{ $t('db.teTemplateApply') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { Search } from '@element-plus/icons-vue';
import { templateService } from './services';
import type { TableTemplate, TableDefinition } from '../types/schema';

interface Props {
  tableName: string;
  dialect?: string;
}

const props = withDefaults(defineProps<Props>(), {
  dialect: 'mysql',
});

const emit = defineEmits<{
  (e: 'apply', table: TableDefinition): void;
}>();

const visible = ref(false);
const searchKeyword = ref('');
const selectedCategory = ref('');
const selectedTemplate = ref<TableTemplate | null>(null);

const allTemplates = templateService.getAllTemplates();
const categories = templateService.getCategories();

const filteredTemplates = computed(() => {
  let result = allTemplates;
  if (searchKeyword.value) {
    result = templateService.searchTemplates(searchKeyword.value);
  }
  if (selectedCategory.value) {
    result = result.filter(t => t.category === selectedCategory.value);
  }
  return result;
});

function open() {
  visible.value = true;
  selectedTemplate.value = null;
  searchKeyword.value = '';
  selectedCategory.value = '';
}

function handleApply() {
  if (!selectedTemplate.value) return;
  const table = templateService.applyTemplate(selectedTemplate.value, {
    tableName: props.tableName,
    dialect: props.dialect,
  });
  emit('apply', table);
  visible.value = false;
}

defineExpose({ open });
</script>

<style scoped lang="scss">
.template-selector {
  &__search {
    display: flex;
    gap: 12px;
    margin-bottom: 16px;
  }

  &__list {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 12px;
    max-height: 300px;
    overflow-y: auto;
    margin-bottom: 16px;
  }

  &__item {
    padding: 12px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s;

    &:hover {
      border-color: var(--el-color-primary);
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    }

    &.is-active {
      border-color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }

    &-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 8px;
    }

    &-name {
      font-weight: 600;
      font-size: 14px;
    }

    &-desc {
      font-size: 12px;
      color: var(--el-text-color-secondary);
      margin-bottom: 8px;
    }

    &-meta {
      display: flex;
      gap: 12px;
      font-size: 12px;
      color: var(--el-text-color-placeholder);
    }
  }

  &__preview {
    h4 {
      margin: 0 0 12px;
      font-size: 14px;
    }
  }
}
</style>
