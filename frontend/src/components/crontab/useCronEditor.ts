import { computed, ref } from 'vue';
import {
    buildExpression,
    defaultSpec,
    nextRunTimes,
    parseCronExpression,
    withRule,
    type CronError,
    type CronFieldKey,
    type CronRule,
    type CronSpec,
} from './cronSpec';

/**
 * cron 面板的草稿状态。
 *
 * 规则集是唯一真源：表达式与运行时间预览都由它派生，控件只回写自己那一条规则，
 * 因此读操作（切页签、翻预览）不会改动表达式，字段之间的牵连也只发生在显式声明的「日/周互斥」上。
 */
export function useCronEditor() {
    const spec = ref<CronSpec>(defaultSpec());
    const error = ref<CronError | null>(null);
    // 描述符（@every 1m 等）无法由规则集表达，面板只读展示原表达式
    const descriptorOnly = ref(false);

    const expression = computed(() => buildExpression(spec.value));
    const preview = computed(() => nextRunTimes(spec.value));

    function load(expr: string) {
        const parsed = parseCronExpression(expr);
        spec.value = parsed.spec;
        error.value = parsed.error;
        descriptorOnly.value = parsed.kind === 'descriptor';
    }

    /** 面板内改动一律产生合法表达式，故同时清掉载入时的解析错误 */
    function setRule(key: CronFieldKey, rule: CronRule) {
        spec.value = withRule(spec.value, key, rule);
        error.value = null;
    }

    /** 草稿重置为默认规则，同样要点「确定」才落回输入框 */
    function reset() {
        load('');
    }

    return { spec, error, descriptorOnly, expression, preview, load, setRule, reset };
}
