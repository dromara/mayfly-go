---
trigger: always_on
---

# API 定义与调用规范

## API 定义

```typescript
import Api from '@/common/Api';

export const accountApi = {
    list: Api.newGet('/sys/accounts'),
    save: Api.newPost('/sys/accounts'),
    update: Api.newPut('/sys/accounts/{id}'),
    del: Api.newDelete('/sys/accounts/{id}'),
    changeStatus: Api.newPut('/sys/accounts/change-status/{id}/{status}'),
};
```

## 调用模式

```typescript
// 简单请求
await accountApi.del.request({ id: row.id });

// 响应式（用于 loading 状态）
const { execute, isFetching } = accountApi.list.useApi();

// 表格集成
<data-table :page-api="accountApi.list" />
```

## 边界

- ✅ **Always**: API 定义集中放在 `api.ts`
- 🚫 **Never**: 直接调用 axios，必须通过 API 封装
