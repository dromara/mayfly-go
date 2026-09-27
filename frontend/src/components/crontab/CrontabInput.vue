<template>
    <div class="w-full">
        <el-input v-model="value" :placeholder="$t('components.crontab.inputPlaceholder')" clearable>
            <template #prepend>
                <el-button icon="Setting" :title="$t('components.crontab.configure')" @click="onOpen" />
            </template>
        </el-input>
        <p v-if="hint.text" class="cron-hint mt-0.5 truncate text-xs" :class="hint.isError ? 'cron-hint-error' : 'text-muted-foreground'" :title="hint.text">
            {{ hint.text }}
        </p>

        <el-dialog
            v-model="visible"
            :title="$t('components.crontab.title')"
            width="min(880px, 94%)"
            append-to-body
            align-center
            draggable
            :close-on-click-modal="false"
        >
            <CrontabPanel
                :spec="spec"
                :expression="expression"
                :source-expression="value"
                :error="error"
                :descriptor-only="descriptorOnly"
                :preview="preview"
                @change-rule="setRule"
                @apply-preset="onApplyPreset"
            />
            <template #footer>
                <div class="flex items-center justify-between gap-2">
                    <span class="text-xs text-muted-foreground">{{ $t('components.crontab.draftHint') }}</span>
                    <div class="flex gap-2">
                        <el-button @click="visible = false">{{ $t('common.cancel') }}</el-button>
                        <el-button @click="reset">{{ $t('common.reset') }}</el-button>
                        <el-button type="primary" :disabled="descriptorOnly" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
                    </div>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import CrontabPanel from './CrontabPanel.vue';
import { inspectCronExpression, type CronPreset } from './cronSpec';
import { useCronEditor } from './useCronEditor';
import { formatDate } from '@/common/utils/format';

const value = defineModel<string>({ default: '' });

const { t } = useI18n();

const visible = ref(false);

const { spec, error, descriptorOnly, expression, preview, load, setRule, reset } = useCronEditor();

// 输入框下一行实时给出下次运行时间，不合法时换成短红字；截断保证不撑开表单项
const hint = computed<{ text: string; isError: boolean }>(() => {
    const state = inspectCronExpression(value.value);
    if (state.incomplete) return { text: '', isError: false };
    if (!state.valid) return { text: t('components.crontab.invalidExpression'), isError: true };
    if (!state.nextRun) return { text: '', isError: false };
    return { text: t('components.crontab.nextRunHint', { time: formatDate(state.nextRun) }), isError: false };
});

/** 打开面板即以输入框当前值载入草稿，未点确定前不回写 */
function onOpen() {
    load(value.value);
    visible.value = true;
}

/** 预设给出一条完整表达式，等同载入一份新草稿 */
function onApplyPreset(preset: CronPreset) {
    load(preset.expr);
}

function onConfirm() {
    value.value = expression.value;
    visible.value = false;
}
</script>

<style scoped>
.cron-hint-error {
    color: var(--el-color-danger);
}
</style>
