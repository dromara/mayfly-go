<template>
  <div class="ddl-preview">
    <div class="ddl-preview__header">
      <span class="ddl-preview__title">{{ $t('db.teDdlPreview') }}</span>
      <div class="ddl-preview__actions">
        <el-button size="small" @click="handleCopy" :disabled="!ddl">
          <el-icon><DocumentCopy /></el-icon> {{ $t('db.teDdlCopy') }}
        </el-button>
        <el-button size="small" @click="handleFormat" :disabled="!ddl">
          <el-icon><MagicStick /></el-icon> {{ $t('db.teDdlFormat') }}
        </el-button>
      </div>
    </div>
    <div class="ddl-preview__content">
      <div v-if="!ddl" class="ddl-preview__empty">
        <el-empty :description="$t('db.teDdlEmpty')" :image-size="80" />
      </div>
      <div v-else class="ddl-preview__code">
        <pre><code>{{ displayDdl }}</code></pre>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { DocumentCopy, MagicStick } from '@element-plus/icons-vue';
import { Msg } from '@/hooks/useI18n';

const { t } = useI18n();

interface Props {
  ddl: string;
}

const props = defineProps<Props>();
const emit = defineEmits<{ (e: 'update:ddl', value: string): void }>();

const formattedDdl = ref('');
const displayDdl = computed(() => formattedDdl.value || props.ddl);

async function handleCopy() {
  try {
    await navigator.clipboard.writeText(displayDdl.value);
    Msg.success(t('db.teDdlCopySuccess'));
  } catch {
    Msg.error(t('db.teDdlCopyFail'));
  }
}

function handleFormat() {
  const keywords = ['SELECT', 'FROM', 'WHERE', 'CREATE', 'TABLE', 'ALTER', 'DROP', 'INDEX', 'PRIMARY KEY', 'FOREIGN KEY', 'NOT NULL', 'DEFAULT', 'AUTO_INCREMENT', 'COMMENT', 'ENGINE', 'CHARSET', 'ORDER BY'];
  let formatted = props.ddl;
  keywords.forEach(kw => {
    formatted = formatted.replace(new RegExp(`\\b${kw}\\b`, 'gi'), `\n${kw}`);
  });
  formatted = formatted.replace(/^\n/, '').trim();
  formattedDdl.value = formatted;
  emit('update:ddl', formatted);
  Msg.success(t('db.teDdlFormatSuccess'));
}
</script>

<style scoped lang="scss">
.ddl-preview {
  display: flex;
  flex-direction: column;
  height: 100%;
  border: 1px solid var(--el-border-color-light);
  border-radius: 4px;

  &__header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 12px;
    background: var(--el-fill-color-light);
    border-bottom: 1px solid var(--el-border-color-light);
  }

  &__title {
    font-weight: 600;
    font-size: 14px;
  }

  &__actions {
    display: flex;
    gap: 8px;
  }

  &__content {
    flex: 1;
    overflow: auto;
    padding: 12px;
  }

  &__empty {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 200px;
  }

  &__code {
    pre {
      margin: 0;
      padding: 12px;
      background: var(--el-fill-color-darker);
      border-radius: 4px;
      overflow: auto;
      font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
      font-size: 13px;
      line-height: 1.6;

      code {
        color: var(--el-text-color-primary);
        white-space: pre-wrap;
        word-break: break-all;
      }
    }
  }
}
</style>
