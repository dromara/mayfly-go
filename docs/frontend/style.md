---
trigger: always_on
---

# 样式与 UI 规范

## Tailwind CSS

优先使用 Tailwind 工具类，支持 `dark:` 前缀：

```vue
<template>
    <div class="flex items-center justify-between p-4 bg-white dark:bg-gray-800">
        <span class="text-sm text-gray-600 dark:text-gray-300">Label</span>
    </div>
</template>
```

## 权限控制

按钮权限使用 `v-auth` 指令：

```vue
<el-button v-auth="'account:add'" type="primary" @click="onAdd">新增</el-button>
```

## 类型安全

- 避免 `any`，使用可选链 `?.`
- 使用 TypeScript 严格模式
- 类型断言（`as T`）只用于「编译器无法自行判定、但运行时确实成立」的场景
- 🚫 禁止用 `as unknown as T` 消除 `TS2352`（类型互不重叠）：该报错说明断言两侧本就不一致，
  应改类型声明或改取值，而不是把双重断言当语法糖。确属结构性无法表达时（如 Vue `reactive` 对 `Ref` 字段的
  代理层解包、动态组件实例的 `expose` 成员），保留双重断言并**在同一行写明绕过了什么**，
  且收敛到唯一的边界函数 / 边界入口处，禁止在各消费点重复出现
- 默认值、局部变量应声明为真实的宽松形（如 `Partial<TForm>`），不要靠断言冒充完整类型
- 表单相关断言另见 `docs/frontend/component.md`「表单类型契约」：类型只在 `defineFormItems<TForm>` 与
  `useAutoFormModel<TForm>` 两个边界入口收敛一次，页面侧不得再写 `rawForm as XxxForm` / `ref<AutoFormData>`；
  本地也不要定 `type FormData` 这类与 DOM 全局同名的表单类型（由 `auto-form/__tests__/form-contract.test.ts` 守护）

> `eslint` 已启用 `@typescript-eslint/no-unnecessary-type-assertion`（类型感知，覆盖 `.ts`）：
> 不改变表达式类型的冗余断言会被报为 warning，`--fix` 可直接删除。

## 边界

- ✅ **Always**: 优先使用 Tailwind CSS
- 🚫 **Never**: 使用固定高度计算，优先用 Flexbox
