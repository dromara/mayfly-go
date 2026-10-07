import { ref, type Ref } from 'vue';

import { type PolicyScenario, type PolicySchema } from './policyModel';

/**
 * policy-schema 的共享加载器。
 *
 * 字段字典、检查项与可复用条件组在一次会话内不会变，而策略编辑器、流程图条件编辑器、
 * 条件组管理页都需要它，因此按模块缓存一份并复用进行中的请求：
 * 逐个组件各自 request 会让同一次操作打三次同一个接口，且各自显示各自的加载态
 */
const schema = ref<PolicySchema | null>(null);
const loading = ref(false);
let pending: Promise<PolicySchema> | null = null;

type SchemaLoader = () => Promise<PolicySchema>;

export function usePolicySchema(loader: SchemaLoader): {
    policySchema: Ref<PolicySchema | null>;
    schemaLoading: Ref<boolean>;
    load: () => Promise<void>;
    /** 让缓存失效并立即重取：用于条件组增删后同页就要看到新字典 */
    reload: () => Promise<void>;
} {
    const load = async () => {
        if (schema.value) return;
        if (!pending) {
            loading.value = true;
            pending = loader()
                .then((res) => {
                    schema.value = res;
                    return res;
                })
                .finally(() => {
                    loading.value = false;
                    pending = null;
                });
        }
        await pending;
    };

    // 只清缓存不重取是不够的：宿主组件已经挂载，onMounted 不会再跑一次，
    // 结果就是列表摘要降级成「未配置条件」、编辑抽屉永远停在加载中
    const reload = async () => {
        schema.value = null;
        pending = null;
        await load();
    };

    return { policySchema: schema, schemaLoading: loading, load, reload };
}

/** 触发策略可配置的场景（不含只服务于流程内部条件的字段字典） */
export function triggerScenariosOf(policySchema: PolicySchema | null): PolicyScenario[] {
    return policySchema?.scenarios ?? [];
}

/** 流程内部条件（连线跳转、节点完成）用的字段字典 */
export function conditionScenariosOf(policySchema: PolicySchema | null): PolicyScenario[] {
    return policySchema?.conditionScenarios ?? [];
}

/**
 * 条件组管理页可选的字段字典：条件组只能被同场景的规则引用，
 * 因此能选的场景必须与两类条件编辑器看到的字典一致
 */
export function allScenariosOf(policySchema: PolicySchema | null): PolicyScenario[] {
    return [...triggerScenariosOf(policySchema), ...conditionScenariosOf(policySchema)];
}

/** 按 bizType 取流程内部条件用的字段字典 */
export function conditionScenarioOf(policySchema: PolicySchema | null, bizType: string): PolicyScenario | null {
    return conditionScenariosOf(policySchema).find((item) => item.bizType === bizType) ?? null;
}
