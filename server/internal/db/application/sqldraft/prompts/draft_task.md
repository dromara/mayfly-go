## 需求
{{.Prompt}}
{{if .Selection}}
## 编辑器当前选区（仅作上下文参考，不要求在原样基础上改）
```sql
{{.Selection}}
```
{{end}}
{{if .Refine}}
{{.Refine}}
{{end}}
## 可用上下文
{{.Context}}
