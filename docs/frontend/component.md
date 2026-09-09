---
trigger: always_on
---

# 组件开发规范

## 代码组织顺序

```
Imports
Props/Emits
常量定义 (as const)
类型定义
响应式数据
计算属性
监听器
工具函数
事件处理方法 (on 开头)
```

## 命名规范

- **事件方法**: 必须以 `on` 开头（`onSubmit`, `onDelete`, `onEdit`）
- **变量/函数**: camelCase
- **常量**: UPPER_SNAKE_CASE + `as const`
- **组件**: PascalCase
- **文件**: 组件用 PascalCase，其他用小写

## Props & Emits

```vue
<script lang="ts" setup>
interface Props {
    visible?: boolean;
    data?: any;
}

const props = withDefaults(defineProps<Props>(), {
    visible: false,
    data: null,
});

const emit = defineEmits<{
    (e: 'update:visible', value: boolean): void;
    (e: 'success'): void;
}>();
</script>
```

## 双向绑定规范

### 使用 defineModel (Vue 3.4+)

**必须使用 `defineModel` 实现双向绑定**，替代旧的 `computed` + `emit('update:xxx')` 模式。

#### 基本用法

```vue
<script lang="ts" setup>
// 单个 v-model
const modelValue = defineModel<string>('modelValue', {
    default: '',
});

// 命名 v-model
const authCertName = defineModel<string>('authCertName');
const machineId = defineModel<number>('machineId');
</script>
```

#### 内部字段联动更新

当组件内部有多个字段，需要联动更新外部的 `modelValue` 时，使用 `watch` 监听：

```vue
<script lang="ts" setup>
import { watch } from 'vue';

const authCertName = defineModel<string>('authCertName');
const machineName = defineModel<string>('machineName');
const selectNode = defineModel<string>('modelValue', { default: '' });

// 监听内部字段变化，自动更新 selectNode
watch(
    [authCertName, machineName],
    () => {
        selectNode.value = authCertName.value 
            ? `${machineName.value} > ${authCertName.value}` 
            : '';
    },
    { immediate: true }
);
</script>
```

#### 规范要点

- ✅ **Always**: 使用 `defineModel` 替代 `computed` + `emit('update:xxx')`
- ✅ **Always**: 为 `defineModel` 提供合适的 `default` 值
- 🚫 **Never**: 使用旧的 `computed` getter/setter 模式实现双向绑定

## 图标使用规范

### 统一使用 SvgIcon 组件

**所有图标必须使用 `SvgIcon` 组件**，禁止使用 `<el-icon>` 配合导入图标组件。

```vue
<!-- ✅ 正确：使用 SvgIcon -->
<SvgIcon name="Monitor" :size="20" />
<SvgIcon name="check" class="text-success" />

<!-- ❌ 错误：使用 el-icon + 导入 -->
<el-icon><Check /></el-icon>
```

**规范要点**：
- ✅ 使用 `name` 属性指定图标，`size` 属性控制大小
- ✅ 图标名称使用 PascalCase 或 kebab-case
- 🚫 禁止使用 `<el-icon>` 和导入 `@element-plus/icons-vue`
- 🚫 禁止通过 class 设置图标大小

### 自定义 SVG 图标

项目支持在 `assets/icon` 目录下添加自定义 SVG 图标。

#### 目录结构

```
frontend/src/assets/icon/
├── db/              # 数据库图标（mysql.svg, postgres.svg...）
├── machine/         # 机器图标
└── ...
```

#### 使用方法

**格式**: `name="icon {目录}/{文件名}"`（不含 .svg）

```vue
<SvgIcon name="icon db/mysql" :size="20" />
```

#### 添加步骤

1. 将 SVG 文件放到 `frontend/src/assets/icon/` 对应子目录
2. 文件名使用小写 + 连字符（如 `mysql.svg`）
3. 使用 `name="icon db/mysql"` 引用

#### 注意事项

- ✅ SVG 必须有 `viewBox` 属性
- ✅ 使用 `size` 属性控制大小
- ✅ 图标颜色继承当前元素的 `color`
- 🚫 不要在 SVG 中硬编码颜色值
- 🚫 文件名不要使用大写或下划线

## JSON 表单 DSL（v1 Schema）

纯 JSON 可序列化的表单定义，供后端下发（系统配置 `t_sys_config.params`、机器脚本入参 `t_machine_script.params`）或代码内以纯数据方式声明表单。定义在 `frontend/src/components/auto-form/json/`，由 `compileJsonForm()` 编译为 `AutoFormItem[]` 后经 AutoForm 系列组件渲染。

表单 Schema 编辑器为 `AutoFormSchemaEdit`（表格化编辑字段列表，用于系统配置/脚本入参定义页面）。

### 设计要点

- **编译层模式**：JSON Schema 是 AutoFormItem 函数型配置的可序列化子集。函数型配置（`when` / `disabled(fn)` / `options(fn)` / `onChange`）由结构化声明（`JsonCondition` / `JsonOptionsSource`）编译生成，**不使用 eval**。
- **显式版本号**：`version: 1` 用于格式自识别（裸 JSON 无类型标签）、后端迁移幂等守卫（见 `v1_12.go` 的 `convertLegacyFormParams`）与未来演进升级。前端经 `isJsonFormSchema()` 守卫入口，仅接受 v1。
- **严格可序列化**：Schema 不允许出现函数；需要行为时用条件/数据源声明表达。

### Schema 结构

```json
{
  "version": 1,
  "cols": 1,
  "fields": [
    {
      "prop": "host",
      "label": "machine.host",
      "type": "input",
      "placeholder": "machine.hostPlaceholder",
      "tooltip": "...",
      "defaultValue": "",
      "rules": { "required": true, "maxLength": 64 },
      "span": 12,
      "when": { "field": "mode", "op": "eq", "value": "custom" },
      "disabled": false,
      "props": {}
    }
  ]
}
```

字段说明（`JsonField`）：

| 属性 | 类型 | 说明 |
| --- | --- | --- |
| `prop` | string | 字段名（必填） |
| `label` / `placeholder` / `tooltip` / `description` | string | 文本类属性，支持 i18n key 或原文（`description` 显示在控件下方） |
| `type` | string | 控件类型，默认 `input`；白名单：`input` / `password` / `number` / `textarea` / `select` / `switch` / `date` / `datetime` / `time` / `monaco` / `divider` / `group`；白名单外类型（如 `custom` / `enum`，无法纯 JSON 表达）编译时跳过并 console.warn |
| `groupDescription` | string | 分组描述（i18n key 或原文，`type='group'` 时显示在分组标题下方） |
| `rules` | JsonRules | 校验规则，见下表 |
| `defaultValue` | any | 字段默认值 |
| `disabled` | boolean \| JsonCondition | 布尔或条件表达式（条件满足时禁用） |
| `readonly` | boolean \| JsonCondition | 只读（语义等同禁用；AutoForm 另支持表单级 `readonly` prop 一键全局只读） |
| `hidden` | boolean | 隐藏字段：不渲染控件但保留在表单数据中（如透传 tenant_id） |
| `when` | JsonCondition | 条件显隐，不满足时隐藏字段（隐藏时不参与校验，字段值保留） |
| `span` | number | el-col 栅格跨度，缺省时由 `cols` 均分 24 |
| `options` | Option[] | select 静态选项（`{ value, label }`），与 `optionsSource` 二选一，优先 `options` |
| `optionsSource` | JsonOptionsSource | select 声明式选项数据源，见下文 |
| `multiple` | boolean | select 多选 |
| `rows` | number | textarea 行数（默认 3） |
| `prefix` / `suffix` | string | input / number 输入框前后缀文本（i18n key 或原文） |
| `min` / `max` | number | number 控件范围 |
| `props` | object | 透传底层控件的额外属性（如 switch 的 `active-value`） |

`JsonRules`：`required`（必填）、`minLength` / `maxLength`（字符串长度）、`pattern`（正则字符串，编译时 `new RegExp`）、`min` / `max`（数值范围）、`message`（校验提示，i18n key 或原文）。

### 条件表达式（JsonCondition）

简单条件 `{ field, op, value }`，或组合条件 `{ all: [...] }`（与）、`{ any: [...] }`（或）、`{ not: {...} }`（非），可任意嵌套：

| 操作符 | 含义 |
| --- | --- |
| `eq` / `ne` | 等于 / 不等于（宽松比较：`'1'` 与 `1` 相等，`null` 与 `undefined` 相等） |
| `in` / `notIn` | 值在 / 不在指定数组内 |
| `empty` / `notEmpty` | 值为空（undefined / null / 空串 / 空数组）/ 非空 |
| `gt` / `gte` / `lt` / `lte` | 数值比较，无法转为数字时恒为 false |

```json
{
  "when": {
    "all": [
      { "field": "type", "op": "eq", "value": "db" },
      { "any": [
          { "field": "engine", "op": "eq", "value": "mysql" },
          { "field": "engine", "op": "eq", "value": "postgres" }
      ] }
    ]
  }
}
```

### 声明式选项数据源（JsonOptionsSource）

select 字段可通过 `optionsSource` 从后端接口异步加载选项，编译为 options 函数：

```json
{
  "prop": "tagId",
  "type": "select",
  "optionsSource": {
    "url": "/api/v1/tag/list",
    "method": "get",
    "params": { "type": 1 },
    "dataField": "list",
    "valueField": "id",
    "labelField": "name",
    "deps": ["tagType"]
  }
}
```

- `url`（必填）：**仅允许站内相对路径**（如 `/api/...`），走统一 request 封装自动附带认证信息，防止被配置成开放代理
- `method`：默认 `get`，可选 `post`
- `params`：附加请求参数
- `dataField`：从响应中提取选项数组的字段名，不传则响应本身需为数组
- `valueField` / `labelField`：每项取 value / label 的字段名（默认 `value` / `label`）
- `deps`：依赖的字段名列表，这些表单字段值变化时重新加载选项，且依赖值会合并进请求参数

### 使用示例

```vue
<script setup lang="ts">
import { AutoFormDialog, type AutoFormJsonSchema } from '@/components/auto-form';

const schema: AutoFormJsonSchema = {
    version: 1,
    fields: [
        { prop: 'name', label: 'common.name', rules: { required: true } },
        { prop: 'type', label: 'common.type', type: 'select', options: [{ value: 1, label: 'A' }] },
        { prop: 'ext', label: 'common.remark', when: { field: 'type', op: 'eq', value: 1 } },
    ],
};
</script>

<template>
    <AutoFormDialog v-model="visible" :schema="schema" v-model:form-data="form" title="demo" @confirm="onConfirm" />
</template>
```

> 说明：AutoForm 系列组件（AutoForm / AutoFormDialog / AutoFormDrawer）均支持 `schema` prop 直接接收 v1 Schema（与 `items` 二选一，优先 schema），内部经编译层渲染；存量旧格式 params 由后端幂等迁移（`v1.12.0-form-params-json-schema-v1`）统一升级为 v1，前端不再兼容旧数组格式。历史上的 dynamic-form 组件已移除，表单能力统一收敛到 auto-form 系列。

### 布局能力（group 分组 / tabs 页签）

AutoForm 的企业级布局能力，函数式与 JSON DSL 双层支持：

**group 分组容器**（`type: 'group'`）：非字段项，渲染标题 + 可选 `groupDescription` + 带边框容器包裹后续字段，直到下一个 group。字段全部隐藏（`when`/`hidden`）时空分组不渲染：

```ts
const items: AutoFormItem[] = [
    { type: 'group', label: 'common.basic' },
    { prop: 'name', label: 'common.name', required: true },
    { prop: 'host', label: 'Host', required: true },
    { type: 'group', label: 'common.other' },
    { prop: 'sshTunnelMachineId', label: 'machine.sshTunnel' },
];
```

**tabs 页签**（TabConfig）：AutoForm / AutoFormDialog / AutoFormDrawer 均支持 `tabs` prop（`AutoFormTab[]`：`{ name, label, icon?, items }`）。所有 Tab 共享同一表单数据与校验（el-tab-pane 非懒渲染，未激活 Tab 字段同样挂载，`validate` 全量生效）：

```ts
const tabs: AutoFormTab[] = [
    { name: 'basic', label: 'common.basic', items: [{ prop: 'name', label: 'common.name', required: true }] },
    { name: 'other', label: 'common.other', items: [{ prop: 'remark', label: 'common.remark', type: 'textarea' }] },
];
```

JSON Schema 内置 `tabs` 字段（`JsonTab[]`：`{ name, label, icon?, fields }`）时自动渲染为 Tab 布局，经 `compileJsonTabs()` 编译。

**其它字段能力**：`prefix` / `suffix`（input/number 前后缀）、`validate`（自定义校验函数，仅函数式 AutoFormItem：`(value, form) => boolean | string | Promise<boolean | string>`，JSON 下发不可用；返回 Promise 支持异步校验，如远程重名/连通性检查，reject 视为不通过）。

**企业级提交与校验契约**：

- `confirm-api`（推荐，Dialog/Drawer 均支持）：传入统一提交 API 后走内置默认提交逻辑——`校验 → 调 API → 成功提示（Msg.saveSuccess）→ 触发 @submitted(form) → 关闭弹层`；失败时保留弹层供修改重提。简单保存场景不再需要手写 confirm 处理器，只需 `@submitted="refreshList"`。
- `@confirm(form)`：保留的完整自定义提交事件（未传 confirmApi 时生效）。组件会捕获处理器的返回 Promise：请求期间确认按钮 loading 且忽略重复点击，请求结束（成功关弹层/失败保留）自动恢复。处理器声明为 async 即可获得请求期防重。
- `#footer` 自定义确认按钮场景：改为调用宿主暴露的 `drawerRef.submit()` 触发同一套 confirmApi 流程（校验 → confirmApi → 提示 → submitted → 关闭），按钮 loading 绑定 `drawerRef.submitting`（覆盖校验期与请求期）。全站保存类表单禁止再自写「校验 → API → Msg → 关弹层」样板。
- 确认按钮内置两层防重：①组件内部 `confirming` 守卫（从点击含校验到提交 Promise settle 全程）；②外部 `confirm-loading`（兼容旧用法的补充驱动）。
- 提交动作函数（confirmApi / @confirm 处理器）职责边界：只做「前置业务校验（失败抛错中止）+ 参数组装 + 调 API + 成功后事件通知」；成功提示与关闭弹层一律由组件内置逻辑负责，不得重复书写。
- `AutoFormInstance.validateField(props?)`：单字段校验（三宿主均暴露），向导式分步场景可配合 `v-model:active-tab` 只校验当前步字段。
- 表单壳 `el-form` 原生属性（如 `scroll-to-error` 校验失败滚动到首个错误项）经 `$attrs` 直接透传。

## 分层架构与 UI 框架无关性

auto-form 采用「core 逻辑层 + UI 适配层」分层，保证换 UI 框架（如 shadcn-vue）时调用点零改动：

| 层 | 文件 | 职责与约束 |
|---|---|---|
| **core（框架无关）** | `types.ts`（契约 + CONTROL_REGISTRY）、`shared.ts`（字段解析/回填/校验编排）、`json/`（Schema 编译与条件求值） | 纯 `.ts`，禁止 import 任何 UI 框架与控件；规则用框架无关的 `AutoFormItemRule`（async-validator 结构化子集）表达。由 `__tests__/architecture.test.ts` 架构守护测试强制 |
| **适配层** | `AutoFormControl.vue`（控件渲染）、`AutoForm.vue`（表单壳/校验桥接）、`AutoFormFieldCol.vue`（栅格） | 当前为 element-plus 实现；换框架时仅重写本层模板与控件分支，`items` / `schema` / `AutoFormItemRule` / `AutoFormInstance` 契约不变 |
| **host（弹层宿主）** | `AutoFormDialog.vue`、`AutoFormDrawer.vue`、`AutoFormSchemaEdit.vue`（Schema 编辑工具）；提交/回填/防重/expose 对称逻辑统一收敛在 `@/hooks/useAutoFormHost.ts` | 弹层组合层，依赖适配层；换框架时随适配层一并替换。Dialog/Drawer 仅保留模板与形态差异（宽高/方向/头部结构），宿主能力新增只改 useAutoFormHost 一处，避免两宿主契约漂移 |

换框架迁移路径：①重写适配层 `.vue` 组件（把 `el-*` 换为新框架控件，`AutoFormItemRule` 转换为新框架规则格式）；②core 层与全站调用点（`items` 声明、`Rules.*`、`AutoFormInstance` ref、事件契约）零改动。注意 `item.props` 是「底层控件原生属性透传」逃生口，其中 UI 专有属性（如 switch 的 `active-value`）需随框架调整。

扩展点（开闭原则）：新增控件类型 = `AutoFormControl` 渲染分支 + `CONTROL_REGISTRY` 登记能力位（`selectLike` / `jsonCompilable`），其余（placeholder 语义、JSON 编译白名单、excludeValues 语义）自动派生；复杂控件一律 `type: 'custom'` 具名插槽逃生口，不经注册表。

## 表单统一规范（el-form → auto-form）

全站表单已统一收敛到 auto-form 系列，新建/修改表单**必须使用** AutoForm / AutoFormDialog / AutoFormDrawer 声明式实现，禁止手写 `el-form + el-form-item` 逐字段模板。

### 基本模式

```vue
<template>
    <auto-form ref="formRef" v-model="form" :items="items" label-width="auto">
        <template #customField>
            <!-- 复杂控件（monaco / 远程选择 / 表格编辑器 / el-tabs 块）放 custom 插槽 -->
        </template>
    </auto-form>
</template>
<script lang="ts" setup>
import { type AutoFormInstance } from '@/components/auto-form';

// ref 契约类型统一从 auto-form 导入，禁止各调用点手写结构体类型
const formRef = useTemplateRef<AutoFormInstance>('formRef');

const items: AutoFormItem[] = [
    { prop: 'name', label: 'common.name', required: true, rules: [Rules.requiredInput('common.name')] },
    { prop: 'type', label: 'common.type', type: 'select', options: [...] },
    { prop: 'customField', type: 'custom' },
];

// 校验统一 Promise 风格
await formRef.value?.validate();
formRef.value?.resetFields?.();
</script>
```

### 关键约定

- **label / placeholder / description / tooltip 传 i18n key**，由 auto-form 统一 `$t`；`description` 渲染在控件下方（替代手写 field-tip div）
- **props 透传文本不走 $t**：经 `props` 透传给底层控件的文本（如 el-switch 的 `active-text`、第三方控件 placeholder）不会被翻译，必须用 `t('key')` 计算值（此时 items 用 computed）
- **保守校验语义**：原表单无校验的字段转换时不加校验（仅星号展示时保持“不参与 validate”行为）；回调风格 `validate(cb)` 统一改 Promise 风格
- **primitive 代理**：独立 ref 绑定 auto-form 时用 `computed({ get, set })` 包装，或改用 reactive 对象
- **条件显隐**用 `when: (form) => boolean`（隐藏字段不参与校验）；`enums` 可直接传枚举对象
- **options label 统一经 $t**：select / enum / radio 的选项 label 支持 i18n key 或原文（missing key 时原文透传）
- **弹层宿主防误关**：AutoFormDialog / AutoFormDrawer 默认 `close-on-click-modal=false`（防误点遮罩丢失已填数据），可用 `close-on-click-modal` prop 显式开启；两者均 expose `validate / resetFields / clearValidate`，回填 `data` 为深拷贝（嵌套对象编辑不污染外部行数据）；支持 `v-model:active-tab` 在弹层内向导式切换 Tab；回填仅在打开瞬间执行，打开期间外部 data 引用变化不重置表单
- **新增控件类型**：在 `AutoFormControl` 增加渲染分支后，同步在 `types.ts` 的 `CONTROL_REGISTRY` 登记能力位（`selectLike` 必填提示语、`jsonCompilable` 是否允许 JSON Schema 下发），选择类提示与 JSON 编译白名单即自动生效，无需散点修改多处判断
- **schema 与 tabs 互斥语义**：`schema.tabs` 存在时以 tabs 为准（顶层 fields 被忽略），默认值回填同样取 tabs 合集；`group` 分组内字段全部被 when 隐藏时整组不渲染（不残留空壳容器）

### 保留原生 el-form 的边界（开闭原则）

以下场景**允许保留/使用原生 el-form**，不视为违规：

1. **动态列/动态字段驱动的表单**：字段名无法静态映射到 items 或插槽（如 `DbTableDataForm` 动态列、`GenericParamInput` 动态参数）
2. **依赖父 el-form 上下文的动态组件面板**：子组件契约依赖 el-form 注入（如流程节点 `PropSettingDrawer` 系列）
3. **复杂交互面板**：深层状态绑定 + 分组下拉/表格勾选等重度交互（如 `RolesGrantPrivilege`、`FieldConfigPanel`、`EsSearch` 内联条件行、`EsDashboard` 只读展示、`AiSettings` 响应式断点栅格配置面板）
4. **无 model/rules 的布局壳**：仅用 el-form 做垂直间距、内部是单个自绘控件或无逻辑占位页（如 `MobileLogin`、`DbTablesOp` 散用 form-item）
5. **强样式图标式登录表单**：无 label、prefix-icon 风格与 auto-form 的 label+控件语义不匹配（如 `AccountLogin` 主表单）
6. **auto-form 自身 / crontab 等基础组件内部**

## 边界

- ✅ **Always**: 使用 Composition API + `<script setup>`
- ✅ **Always**: 事件方法以 `on` 开头
- ✅ **Always**: 移除无用的导入（import）和无用的字段、变量、函数
- 🚫 **Never**: 保留未使用的代码或注释掉的代码
- 🚫 **Never**: 使用固定高度计算，优先用 Flexbox
