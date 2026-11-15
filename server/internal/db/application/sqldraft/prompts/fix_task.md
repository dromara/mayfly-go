## 需要改写的 SQL
```sql
{{.Materials}}
```

## 改写要求
{{if .Prompt}}
{{.Prompt}}
{{else}}
用户没有指定具体目标：按本意图的通用目标处理（见系统提示里的约束），不要把空目标当成无需处理而原样返回。
{{end}}

{{if .Refine}}
{{.Refine}}
{{end}}
## 可用上下文
{{.Context}}
