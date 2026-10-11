<template>
    <el-dialog v-model="visible" :title="dialogTitle" width="720px">
        <div class="mb-2 flex flex-wrap items-center gap-2">
            <el-tag size="small" type="info">{{ $t('db.binaryBytes', { bytes: String(byteCount) }) }}</el-tag>
            <el-radio-group v-model="form" size="small" :disabled="!decodable">
                <el-radio-button value="hex">HEX</el-radio-button>
                <el-radio-button value="base64">Base64</el-radio-button>
                <el-radio-button value="utf8">{{ $t('db.binaryFormUtf8') }}</el-radio-button>
            </el-radio-group>
            <el-button link type="primary" :disabled="!shown" @click="onCopy">{{ $t('common.copy') }}</el-button>
        </div>

        <!-- 值不是 hex 形态时不能装作能预览：硬解出来的字节与库里的内容无关 -->
        <el-alert v-if="!decodable" type="warning" :title="$t('db.binaryNotHex')" :closable="false" show-icon class="mb-2" />
        <template v-else>
            <el-alert
                v-if="form === 'utf8'"
                :type="lowPrintable ? 'warning' : 'info'"
                :title="lowPrintable ? $t('db.binaryUtf8LowPrintable', { ratio: printablePercent }) : $t('db.binaryUtf8Hint')"
                :closable="false"
                show-icon
                class="mb-2"
            />
            <el-alert v-if="preview?.truncated" type="info" :title="$t('db.binaryUtf8Truncated')" :closable="false" show-icon class="mb-2" />
            <pre class="binary-preview__content">{{ shown }}</pre>
        </template>

        <template #footer>
            <el-button type="primary" @click="visible = false">{{ $t('common.close') }}</el-button>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
/**
 * 二进制列值的只读预览。
 *
 * 单元格里的值是后端回读出来的 hex 串，这里只换算形态（hex / base64 / 按 UTF-8 猜读），
 * 不提供写回：写路径仍是就地编辑那个 hex 文本框，与本弹窗互不干扰。
 * UTF-8 那一档必须带着「猜测性解读」的说明——二进制列没有编码声明，把替换字符画出来当原文是骗人。
 */
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { copyToClipboard } from '@/common/utils/string';
import { bytesToBase64, hexToBytes, utf8Preview } from './binaryValue';

const props = defineProps<{ hex?: string; columnName?: string }>();

const visible = defineModel<boolean>('visible', { default: false });

const { t } = useI18n();

const form = ref<'hex' | 'base64' | 'utf8'>('hex');

// 列名拼在标题里而不是写进文案：vue-i18n 的消息体不支持在占位符里写表达式，
// 而「无列名时不要留一对空方括号」属于呈现组合，归组件自己
const dialogTitle = computed(() => (props.columnName ? `${t('db.binaryPreviewTitle')} [${props.columnName}]` : t('db.binaryPreviewTitle')));

const bytes = computed(() => hexToBytes(props.hex ?? ''));
const decodable = computed(() => bytes.value !== null);
const byteCount = computed(() => bytes.value?.length ?? 0);
const preview = computed(() => (bytes.value ? utf8Preview(bytes.value) : null));

// utf8Preview 算出的可打印占比必须落地到界面上：一屏替换字符摆在用户面前时，他需要的是
// 「这本来就不是文本」这个结论与量到的比例，而不是自己去数
const LOW_PRINTABLE_RATIO = 0.6;
const lowPrintable = computed(() => (preview.value ? preview.value.printableRatio < LOW_PRINTABLE_RATIO : false));
const printablePercent = computed(() => `${Math.round((preview.value?.printableRatio ?? 1) * 100)}%`);

const shown = computed(() => {
    const value = bytes.value;
    if (!value) {
        return props.hex ?? '';
    }
    if (form.value === 'base64') {
        return bytesToBase64(value);
    }
    if (form.value === 'utf8') {
        return preview.value?.text ?? '';
    }
    return (props.hex ?? '')
        .toUpperCase()
        .replace(/(.{2})/g, '$1 ')
        .trim();
});

// 复制成功提示由 copyToClipboard 统一发出，这里再叠一条会弹两个一模一样的 toast
function onCopy() {
    void copyToClipboard(shown.value);
}
</script>

<style lang="scss" scoped>
.binary-preview__content {
    max-height: 380px;
    overflow: auto;
    margin: 0;
    padding: 8px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 4px;
    background-color: var(--el-fill-color-lighter);
    font-family: var(--el-font-family-mono, ui-monospace, monospace);
    font-size: 12px;
    line-height: 18px;
    white-space: pre-wrap;
    word-break: break-all;
}
</style>
