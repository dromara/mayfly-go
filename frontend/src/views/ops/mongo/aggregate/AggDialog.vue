<template>
    <el-dialog v-model="visible" width="72%" top="6vh" :title="$t('mongo.aggTitle', { ns })" destroy-on-close :close-on-click-modal="false" @closed="reset">
        <div class="flex flex-col gap-2">
            <span class="text-[13px] text-gray-500">{{ $t('mongo.aggTip') }}</span>
            <monaco-editor v-model="aggState.pipelineText" language="json" height="240px" />

            <div class="flex flex-wrap items-center gap-3">
                <!-- explain 只解释不执行：含 $out 的管道在解释模式下也不会写数据，服务端据此按只读放行 -->
                <el-switch v-model="aggState.explain" :active-text="$t('mongo.aggExplain')" />
                <el-checkbox v-model="aggState.allowDiskUse">{{ $t('mongo.aggAllowDisk') }}</el-checkbox>
                <el-button type="primary" icon="video-play" :loading="aggState.loading" :disabled="!target" @click="onRun">
                    {{ $t('mongo.run') }}
                </el-button>
                <span v-if="result" class="text-[13px] text-gray-500">{{ resultStatus }}</span>
            </div>

            <el-alert v-if="aggState.error" :title="aggState.error" type="error" show-icon :closable="false" />

            <div v-if="result" class="agg-result">
                <doc-table :docs="result.docs ?? []" :columns="columns" readonly :loading="aggState.loading" />
            </div>
        </div>

        <template #footer>
            <span class="text-[12px] text-gray-400">{{ $t('mongo.aggLimitTip') }}</span>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
import { computed, defineAsyncComponent } from 'vue';

import { useI18n } from 'vue-i18n';
import DocTable from '../docview/DocTable.vue';
import { inferColumns } from '../docview/schema';
import { useAggregation } from '../resource/composables/useAggregation';
import type { CollectionParam } from '../types';

const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

const props = defineProps<{ target: CollectionParam | null }>();

const visible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits<{
    /** 管道含 $out/$merge 且真的执行了：目标集合已被改动，父级需要重查 */
    written: [];
}>();

const { t } = useI18n();

const { state: aggState, run, reset } = useAggregation();

const ns = computed(() => (props.target ? `${props.target.database}.${props.target.collection}` : ''));

const result = computed(() => aggState.page);

/** 聚合输出的形状由 stage 决定，列只能从结果里推断；不做列选择是因为这里只用于看一眼结果 */
const columns = computed(() => inferColumns(result.value?.docs ?? []).columns);

const resultStatus = computed(() => {
    const page = result.value;
    if (!page) {
        return '';
    }
    const loaded = page.docs?.length ?? 0;
    return page.truncated ? t('mongo.aggTruncated', { loaded, limit: page.limit }) : t('mongo.aggLoaded', { loaded });
});

/** 只按文本判断是否写数据：真正的分级由服务端 Classify 决定，这里只用来决定要不要让父级重查 */
const writesData = computed(() => /\$out|\$merge/.test(aggState.pipelineText));

async function onRun() {
    if (!props.target) {
        return;
    }
    const ok = await run(props.target);
    if (ok && !aggState.explain && writesData.value) {
        emit('written');
    }
}
</script>

<style lang="scss" scoped>
.agg-result {
    max-height: 320px;
    overflow: auto;
}
</style>
