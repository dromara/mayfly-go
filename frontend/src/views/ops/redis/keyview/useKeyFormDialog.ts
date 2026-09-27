import { reactive } from 'vue';
import { useI18n } from 'vue-i18n';
import type { RedisFormSchema, RedisKeyMember } from '../types';
import { formatOpResult, prefillForm, toArgs } from './descriptor';
import type { KeyViewStore } from './useKeyView';

/** 弹层承载的动作种类：成员写入与视角操作共用一套表单渲染与提交链路 */
type DialogKind = 'member' | 'op';

/**
 * 成员表单弹层与视角操作弹层的共用逻辑。
 *
 * 表单结构来自后端视角描述符（AutoForm v1 Schema），因此新增类型不需要新增弹层组件；
 * 提交动作只做「组装 args + 调接口」，成功提示与关闭由 AutoFormDialog 宿主负责
 */
export function useKeyFormDialog(store: KeyViewStore) {
    const { t } = useI18n();

    /** 弹层状态：schema 决定渲染哪些字段，data 是编辑态的回填值 */
    const dialog = reactive<{ visible: boolean; title: string; schema?: RedisFormSchema; data: Record<string, unknown> | null }>({
        visible: false,
        title: '',
        schema: undefined,
        data: null,
    });

    /** 只读操作的回包（如位图统计、GEO 距离），与表单弹层分开显示 */
    const opResult = reactive({ visible: false, title: '', content: '' });

    const state = reactive<{ kind: DialogKind; op: string; editing: RedisKeyMember | null; ttl?: number }>({
        kind: 'member',
        op: '',
        editing: null,
        ttl: undefined,
    });

    /** key 还不存在时，首个成员写入需要顺带把 TTL 落下去 */
    function openMember(opName: 'create' | 'update', row?: RedisKeyMember, ttl?: number) {
        const descriptor = store.descriptor.value;
        const schema = opName === 'update' ? (descriptor?.updateForm ?? descriptor?.form) : descriptor?.form;
        if (!descriptor || !schema) {
            return;
        }

        state.kind = 'member';
        state.op = opName;
        state.editing = opName === 'update' ? (row ?? null) : null;
        state.ttl = ttl;
        dialog.title = t(opName === 'update' ? 'redis.editMember' : 'redis.addMember');
        dialog.schema = schema;
        // 新增态传 null：宿主据此按 schema 的 defaultValue 重建表单（传 {} 会被当作回填数据）
        dialog.data = opName === 'update' ? prefillForm(schema, row) : null;
        dialog.visible = true;
    }

    /** 无入参的操作直接执行，避免弹出一个空表单 */
    async function openOp(opName: string) {
        const spec = store.descriptor.value?.ops.find((item) => item.name === opName);
        if (!spec) {
            return;
        }
        if (spec.form?.fields.length) {
            state.kind = 'op';
            state.op = opName;
            dialog.title = t(spec.label);
            dialog.schema = spec.form;
            dialog.data = null;
            dialog.visible = true;
            return;
        }
        await submitOp(opName, {});
    }

    async function submitOp(opName: string, args: Record<string, string>) {
        const spec = store.descriptor.value?.ops.find((item) => item.name === opName);
        const content = formatOpResult(await store.runOp(opName, args));
        opResult.title = t(spec?.label ?? opName);
        // 操作跑完就必须给出结果：nil 回包也要占位，否则用户无法区分「没执行」与「执行了但无返回」
        opResult.content = content || t('redis.opResultEmpty');
        opResult.visible = true;
    }

    /**
     * 成员写入走宿主的 confirmApi：校验 → 请求 → 成功提示 → 关弹层全部内置。
     * 视角操作（多为只读统计）走 @confirm：由本函数负责关弹层，避免给一次「查询」弹出「保存成功」
     */
    async function submit(form: Record<string, unknown>) {
        const args = toArgs(form);
        if (state.kind === 'op') {
            await submitOp(state.op, args);
            dialog.visible = false;
            return;
        }
        await store.saveMember(state.op, args, state.editing, state.ttl);
    }

    return { dialog, dialogKind: () => state.kind, opResult, openMember, openOp, submit };
}
