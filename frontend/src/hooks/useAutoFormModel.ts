/**
 * 页面侧接管 AutoForm 宿主的内部表单
 *
 * AutoFormDrawer / AutoFormDialog 在回填完成后通过 @opened 抛出内部表单引用，
 * 页面需要在提交、连通性测试等环节读取它。直接声明 `ref<AutoFormData>` 会让业务表单类型
 * 在整条链上丢失，页面只能在使用点反复 `rawForm as XxxForm` 断言回来。
 *
 * 本 composable 把「宿主表单袋 → 业务表单类型」这一处类型收敛集中到唯一入口
 * （宿主内部以 Record<string, any> 持有表单，TS 无法从 @opened 的实参推断页面表单形状），
 * 页面侧全部读取点因此获得真实类型。
 */
import { shallowRef } from 'vue';
import type { ShallowRef } from 'vue';
import type { AutoFormData } from '@/components/auto-form/types';

export interface AutoFormModel<TForm> {
    /**
     * 宿主内部表单引用（回填完成前为 undefined）
     *
     * 渲染期派生状态（按钮 disabled、Tab 可用性、徽标文案等）只能读它并按未接管兜底：
     * 弹层关闭时宿主也会渲染 #footer / #body-extra 内容，那一次渲染早于 @opened。
     */
    form: ShallowRef<TForm | undefined>;
    /** 绑定宿主 @opened：宿主抛出的即它内部持有的同一表单对象，此处只做一次类型收敛 */
    onOpened: (raw: AutoFormData) => void;
    /** 组合 @opened：先接管表单，再跑页面自己的回填后置逻辑（形参已是业务表单类型） */
    openedWith: (after: (form: TForm) => void) => (raw: AutoFormData) => void;
    /**
     * 读取当前业务表单；未回填即读取属调用时序错误（宿主保证 confirm 发生在 opened 之后）
     *
     * 仅供交互期（提交、按钮点击、字段联动等）调用，不得包成 computed 供模板读取，
     * 否则弹层关闭态的那次渲染就会抛错（见 form 的说明）。
     */
    requireForm: () => TForm;
}

/**
 * @typeParam TForm 页面业务表单类型（如 MachineForm、DbInstanceForm）
 *
 * @example
 * ```ts
 * const { form, onOpened, requireForm } = useAutoFormModel<MachineForm>();
 * // 模板：<auto-form-drawer @opened="onOpened" :confirm-api="onConfirm" />
 * const onConfirm = async () => saveMachineExec(toSubmitForm(requireForm()));
 *
 * // 渲染期派生状态按未接管兜底（关闭态也会渲染）：
 * // <el-button :disabled="!form?.name">
 *
 * // 需要在回填后做额外初始化时（页面回调形参仍直接是业务表单）：
 * const onOpened = openedWith((form) => loadDbList(form.db));
 * ```
 */
export function useAutoFormModel<TForm extends AutoFormData>(): AutoFormModel<TForm> {
    // shallowRef：只整体替换引用、不做深层解包（宿主抛出的对象本身已是响应式表单）
    const form = shallowRef<TForm | undefined>();

    const onOpened = (raw: AutoFormData) => {
        form.value = raw as TForm;
    };

    const requireForm = (): TForm => {
        if (!form.value) {
            throw new Error('[AutoForm] form is read before @opened, check the drawer lifecycle');
        }
        return form.value;
    };

    const openedWith = (after: (form: TForm) => void) => (raw: AutoFormData) => {
        onOpened(raw);
        after(requireForm());
    };

    return { form, onOpened, openedWith, requireForm };
}
