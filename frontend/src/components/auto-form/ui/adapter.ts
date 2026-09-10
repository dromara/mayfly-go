/**
 * AutoForm UI 适配层（element-plus 实现）
 *
 * 将 auto-form 模块与 element-plus 的耦合收敛到单一文件：
 * - 字段家族组件 / 布局壳 / 表单宿主 均从此处导入 UI 组件，禁止直接 import 'element-plus'
 * - 切换 UI 框架时仅需替换本文件实现（如 ant-design/adapter.ts），
 *   字段组件模板的框架特定属性（如 el-input 的 show-password）需在新适配文件中对齐
 *
 * 架构守护：__tests__/architecture.test.ts 确保 auto-form 的 .ts 文件
 * （types.ts / shared.ts / useFieldControl.ts / json/）不出现 element-plus 导入。
 * 本文件是适配层，不在守护范围内，是 element-plus 的唯一入口。
 */

// ── 组件适配（pass-through，保持 props / slots / events 透传） ──

export {
    // 表单控件
    ElInput as AInput,
    ElInputNumber as AInputNumber,
    ElSelect as ASelect,
    ElOption as AOption,
    ElRadioGroup as ARadioGroup,
    ElRadio as ARadio,
    ElSwitch as ASwitch,
    ElDatePicker as ADatePicker,
    ElTimePicker as ATimePicker,
    ElInputTag as AInputTag,
    // 布局与表单壳
    ElForm as AForm,
    ElFormItem as AFormItem,
    ElRow as ARow,
    ElCol as ACol,
    ElTooltip as ATooltip,
    ElDivider as ADivider,
    // Tab 布局
    ElTabs as ATabs,
    ElTabPane as ATabPane,
    // 弹层宿主
    ElDialog as ADialog,
    ElDrawer as ADrawer,
    // Schema 编辑器
    ElTable as ATable,
    ElTableColumn as ATableColumn,
    ElButton as AButton,
    ElCheckbox as ACheckbox,
} from 'element-plus';

// ── 类型适配（UI 框架类型再导出，切换框架时仅本文件变动） ──

export type { FormInstance, FormItemRule } from 'element-plus';
