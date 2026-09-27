<template>
    <div class="flex h-full min-h-0 flex-col gap-2">
        <!-- 快捷命令：模板来自当前 key 视角的 consoleHints，点击只填入不执行（多数模板还需补成员/字段参数） -->
        <div v-if="props.hints.length" class="flex flex-wrap items-center gap-1.5">
            <span class="hint-label">{{ $t('redis.consoleHintsLabel') }}</span>
            <button v-for="hint in props.hints" :key="hint" type="button" class="hint-chip" @click="onPickHint(hint)">{{ hint }}</button>
        </div>

        <div class="flex items-center gap-2">
            <el-autocomplete
                v-model="cmdLine"
                class="console-input min-w-0 flex-1"
                :placeholder="$t('redis.consolePlaceholder')"
                :fetch-suggestions="fetchSuggestions"
                :trigger-on-focus="false"
                value-key="value"
                @select="onPickSuggestion"
                @keyup.enter="onRun"
            >
                <template #prefix>
                    <span class="prompt">&gt;</span>
                </template>
                <template #default="{ item }">
                    <div class="suggestion">
                        <span class="suggestion-name">{{ item.name }}</span>
                        <span v-if="item.write" class="suggestion-tag suggestion-tag--write">{{ $t('redis.cmdWrite') }}</span>
                        <span v-if="item.confirm" class="suggestion-tag suggestion-tag--danger">{{ $t('redis.cmdDangerous') }}</span>
                    </div>
                </template>
                <template #append>
                    <el-button :loading="running" :disabled="!cmdLine.trim()" @click="onRun">
                        <SvgIcon name="Promotion" :size="14" />
                        <span class="ml-1">{{ $t('redis.runCmd') }}</span>
                    </el-button>
                </template>
            </el-autocomplete>
            <el-button size="small" link type="info" :disabled="!history.length" @click="history = []">{{ $t('redis.clearHistory') }}</el-button>
        </div>

        <el-scrollbar class="console-output min-h-0 flex-1 rounded-md">
            <div v-if="!history.length" class="flex h-full items-center justify-center">
                <el-empty :description="$t('redis.consoleEmpty')" :image-size="70" />
            </div>
            <div v-for="(item, index) in history" :key="index" class="console-item" :class="{ 'console-item--error': item.error }">
                <div class="flex items-center gap-2">
                    <div class="console-cmd min-w-0 flex-1 truncate font-mono text-xs">$ {{ item.cmd }}</div>
                    <el-tooltip :content="$t('common.copy')" placement="top">
                        <el-button class="console-copy" text size="small" icon="DocumentCopy" :aria-label="$t('common.copy')" @click="onCopy(item)" />
                    </el-tooltip>
                </div>
                <pre class="console-result font-mono text-xs">{{ item.error || item.output }}</pre>
            </div>
        </el-scrollbar>
    </div>
</template>

<script lang="ts" setup>
import { onMounted, onBeforeUnmount, ref } from 'vue';
import { copyToClipboard } from '@/common/utils/string';
import { useI18nConfirm } from '@/hooks/useI18n';
import type { RedisCommandSpec } from '../types';
import type { RedisInst } from '../redis';
import { commandNeedsConfirm, editingToken, isKeyArgument, suggestCommands, suggestKeys, splitCommand, type RedisConsoleSuggestion } from './console';
import { cachedCommandCatalog, loadCommandCatalog } from './commandCatalog';
import { formatOpResult } from './descriptor';

const props = defineProps<{
    redis: RedisInst;
    /** 可提示的 key 名：取已加载的 key 列表，避免每敲一个字符就去服务端扫一次库 */
    keys: string[];
    /** 当前 key 的快捷命令（{key} 占位符已替换） */
    hints: string[];
}>();

type ConsoleEntry = { cmd: string; output: string; error: string };

const cmdLine = ref('');
const running = ref(false);
const history = ref<ConsoleEntry[]>([]);
// 目录缓存挂在模块级（commandCatalog.ts）：<script setup> 里的变量是组件实例一份，
// tab 重开就丢，这里只需要在挂载时把共享缓存同步进来
const commands = ref<RedisCommandSpec[]>(cachedCommandCatalog(props.redis.id) ?? []);

const commandSpecOf = (name: string) => commands.value.find((item) => item.name === name.toUpperCase());

onMounted(async () => {
    if (commands.value.length) {
        return;
    }
    try {
        commands.value = await loadCommandCatalog(props.redis.id, props.redis.db);
    } catch {
        // 提示只是增强能力：拿不到目录就没有补全，命令照样能执行，不可逆动作由兜底确认名单挡住
    }
});

/**
 * 输入提示按参数位分派：命令名位置提示命令，键参数位置提示 key 名，
 * 其余位置（分值、成员名等）不提示，避免一屏无关候选挡住视线
 */
const fetchSuggestions = (line: string, cb: (items: RedisConsoleSuggestion[]) => void) => {
    const { index, token } = editingToken(line);
    if (index === 0) {
        cb(suggestCommands(commands.value, token));
        return;
    }
    const spec = commandSpecOf(splitCommand(line.trim())[0] ?? '');
    cb(isKeyArgument(spec, index) ? suggestKeys(props.keys, token) : []);
};

const onPickHint = (hint: string) => {
    cmdLine.value = hint;
};

/**
 * 用回车采纳建议时，同一次按键会紧接着触发 keyup.enter，不拦一下就会把
 * 「get 」这种刚补到一半的命令当真执行，历史里多一条无意义的参数个数报错。
 * 鼠标点选不会有随后的 keyup，所以标志位必须定时复位，否则会吞掉下一次回车
 */
const PICK_GUARD_MS = 400;
let pickedBySuggestion = false;
let pickTimer = 0;

const onPickSuggestion = () => {
    pickedBySuggestion = true;
    window.clearTimeout(pickTimer);
    pickTimer = window.setTimeout(() => (pickedBySuggestion = false), PICK_GUARD_MS);
};

onBeforeUnmount(() => window.clearTimeout(pickTimer));

const onRun = async () => {
    if (pickedBySuggestion) {
        pickedBySuggestion = false;
        return;
    }
    const line = cmdLine.value.trim();
    const args = splitCommand(line);
    if (!args.length) {
        return;
    }
    if (commandNeedsConfirm(commandSpecOf(args[0]), args[0])) {
        try {
            await useI18nConfirm('redis.consoleDangerConfirm', { cmd: line });
        } catch {
            // 取消即放弃执行：确认框被 reject 是正常交互路径，不能冒到控制台变成一条 error
            return;
        }
    }

    running.value = true;
    try {
        const result = await props.redis.runCmd<unknown>(args);
        history.value.unshift({ cmd: line, output: formatOpResult(result), error: '' });
        cmdLine.value = '';
    } catch (error) {
        // 命令失败也要留在历史里，否则用户看不到服务端返回的错误原因
        history.value.unshift({ cmd: line, output: '', error: (error as Error).message });
    } finally {
        running.value = false;
    }
};

const onCopy = async (item: ConsoleEntry) => {
    // 复制结果提示由 copyToClipboard 统一发出，这里不再叠一条
    await copyToClipboard(item.error || item.output);
};
</script>

<style lang="scss" scoped>
@use './toolbar.scss' as *;

.hint-label {
    color: var(--el-text-color-secondary);
    font-size: 11px;
}

// 快捷命令用胶囊：内容是命令文本，等宽字体比按钮底色更能读清参数
.hint-chip {
    padding: 1px 7px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 999px;
    background-color: var(--el-fill-color-lighter);
    color: var(--el-text-color-regular);
    font-family: var(--el-font-family-mono, ui-monospace, monospace);
    font-size: 11.5px;
    cursor: pointer;
    transition:
        border-color $redis-duration $redis-ease,
        color $redis-duration $redis-ease;
}

.hint-chip:hover {
    border-color: var(--el-color-primary);
    color: var(--el-color-primary);
}

// 建议项：命令名靠左，写/高危标记靠右，一眼能分辨这条命令会不会改数据
.suggestion {
    display: flex;
    align-items: center;
    gap: 6px;
    width: 100%;
}

.suggestion-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-family: var(--el-font-family-mono, ui-monospace, monospace);
    font-size: 12.5px;
}

.suggestion-tag {
    margin-left: auto;
    padding: 0 5px;
    border-radius: 4px;
    font-size: 10.5px;
}

.suggestion-tag--write {
    background-color: var(--el-color-primary-light-9);
    color: var(--el-color-primary-dark-2);
}

.suggestion-tag--danger {
    background-color: var(--el-color-danger-light-9);
    color: var(--el-color-danger-dark-2);
}

.console-output {
    padding: 4px;
    background-color: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: $redis-radius;
}

// 每次执行一张卡片：失败用整圈描边 + 同色浅底强调，不用侧边色条
.console-item {
    padding: 6px 10px;
    border: 1px solid transparent;
    border-radius: 6px;
    transition:
        background-color $redis-duration $redis-ease,
        border-color $redis-duration $redis-ease;
}

.console-item:hover {
    background-color: var(--el-bg-color);
}

.console-item:hover .console-copy {
    opacity: 1;
}

.console-item--error {
    border-color: var(--el-color-danger-light-7);
    background-color: var(--el-color-danger-light-9);
}

.console-item--error:hover {
    background-color: var(--el-color-danger-light-9);
}

.console-copy {
    flex-shrink: 0;
    opacity: 0;
    transition: opacity $redis-duration $redis-ease;
}

.console-cmd {
    color: var(--el-color-primary);
    font-weight: 600;
}

.console-item--error .console-cmd {
    color: var(--el-color-danger);
}

.console-result {
    margin-top: 4px;
    white-space: pre-wrap;
    word-break: break-all;
    color: var(--el-text-color-regular);
}

.console-item--error .console-result {
    color: var(--el-color-danger);
}

@media (prefers-reduced-motion: reduce) {
    .console-item,
    .console-copy,
    .hint-chip {
        transition: none;
    }
}
</style>
