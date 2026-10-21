package api

import (
	"reflect"
	"testing"

	"mayfly-go/internal/agentops/api/form"
	"mayfly-go/internal/agentops/api/vo"
)

// jsonTags 取结构体的 json 字段名集合（忽略 "-" 与内嵌基础模型）
func jsonTags(t reflect.Type) map[string]struct{} {
	out := make(map[string]struct{}, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous {
			continue
		}
		name := f.Tag.Get("json")
		if name == "" || name == "-" {
			continue
		}
		// 去掉 ,omitempty 等后缀
		for j := 0; j < len(name); j++ {
			if name[j] == ',' {
				name = name[:j]
				break
			}
		}
		out[name] = struct{}{}
	}
	return out
}

// missingJsonFields 返回 form 里有、VO 里没有的 json 字段名
func missingJsonFields(formType, voType reflect.Type) []string {
	voFields := jsonTags(voType)
	var missing []string
	for name := range jsonTags(formType) {
		if _, ok := voFields[name]; !ok {
			missing = append(missing, name)
		}
	}
	return missing
}

// TestTaskVoCoversTaskForm 任务表单可编辑字段必须全部能在任务 VO 里回传。
//
// 为什么钉这条：任务/触发器保存走「可编辑列整列真实写入」（零值不再被 GORM 跳过），而编辑表单
// 是从列表行/详情回填的。VO 少一个字段（曾少 dryRun/tokenBudget），前端就把该字段回填成
// 零值再提交，一次普通编辑即静默改掉真实配置（影子模式被关掉、token 预算被清零）。
// 这条反射断言把「表单字段 ⊆ VO 字段」变成编译期之外的硬门禁，新增表单字段漏进 VO 会直接红。
func TestTaskVoCoversTaskForm(t *testing.T) {
	missing := missingJsonFields(reflect.TypeOf(form.AgentTaskForm{}), reflect.TypeOf(vo.AgentTaskVO{}))
	if len(missing) > 0 {
		t.Fatalf("AgentTaskVO 缺少表单可编辑字段对应的回传字段：%v（VO 缺字段会导致表单回填丢值并被零值覆盖）", missing)
	}
}

// TestTriggerVoCoversTriggerForm 触发器同理（webhookToken 例外：它不在表单里，由服务端治理）
func TestTriggerVoCoversTriggerForm(t *testing.T) {
	missing := missingJsonFields(reflect.TypeOf(form.TriggerForm{}), reflect.TypeOf(vo.TriggerVO{}))
	for _, name := range missing {
		if name == "rotateWebhookToken" {
			// 命令位而非状态位：轮换走单独动作，不需要回传
			continue
		}
		t.Fatalf("TriggerVO 缺少表单字段 %q，编辑回填会丢值", name)
	}
}
