package application

import (
	"context"
	"sync"
	"testing"

	"mayfly-go/internal/ai/application/dto"
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/domain/repository"
	aipersist "mayfly-go/internal/ai/infra/persistence"
	"mayfly-go/internal/ai/pkg/headless"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var aiRepoOnce sync.Once

// setupConversationSvc 内存 sqlite + 真实仓储：会话授权规则必须落在真实读写路径上验证
func setupConversationSvc(t *testing.T) *conversationAppImpl {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:               logger.Default.LogMode(logger.Silent),
		NamingStrategy:       gormx.MetaNamingStrategy(),
		DisableAutomaticPing: true,
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	require.NoError(t, db.AutoMigrate(&entity.Conversation{}, &entity.TurnItem{}))

	prev := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = prev })

	aiRepoOnce.Do(aipersist.InitIoc)

	svc := &conversationAppImpl{}
	svc.Repo = ioc.Get[repository.Conversation]()
	svc.turnItemRepo = ioc.Get[repository.TurnItem]()
	return svc
}

// asUser 以指定账号身份构造上下文
func asUser(id uint64) context.Context {
	return contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: id, Username: "u"})
}

// 会话消息读取与改名/删除的归属规则：列表按 creatorId 过滤，但消息详情接口按路径 id 直取，
// 不校验时会话内容可被任何登录账号按递增 id 枚举
func TestConversationAccessRules(t *testing.T) {
	svc := setupConversationSvc(t)

	userConv, err := svc.CreateConversation(asUser(7), &dto.CreateConversationReq{Title: "我的对话", Source: entity.ConversationSourceUser})
	require.NoError(t, err)
	systemConv, err := svc.CreateConversation(asUser(1), &dto.CreateConversationReq{Title: "运维运行", Source: entity.ConversationSourceSystem})
	require.NoError(t, err)

	// 本人可读自己的会话；他人不可
	require.NoError(t, svc.AssertReadable(asUser(7), userConv.Id))
	require.Error(t, svc.AssertReadable(asUser(8), userConv.Id), "别人的用户会话不得可读")

	// 系统任务会话对所有登录账号可读（执行历史回放入口）
	require.NoError(t, svc.AssertReadable(asUser(8), systemConv.Id))

	// 写操作更严：系统会话也不得被任意账号改名/删除
	require.NoError(t, svc.AssertOwned(asUser(7), userConv.Id))
	require.Error(t, svc.AssertOwned(asUser(8), userConv.Id))
	require.Error(t, svc.AssertOwned(asUser(8), systemConv.Id), "系统会话不得被非归属账号改名/删除")

	// 无登录态一律拒绝（不因「读接口不校验」而默认放行）
	require.Error(t, svc.AssertReadable(context.Background(), userConv.Id))

	// 写侧（发起新一轮对话）：本人用户会话可写，他人不可写
	require.NoError(t, svc.AssertWritable(asUser(7), userConv.Id))
	require.Error(t, svc.AssertWritable(asUser(8), userConv.Id), "不得往别人的会话里追加消息")

	// 系统运行会话对所有账号不可写，连 owner 也不行：它是审计产物，
	// 写进去会混入事后对话并改写该运行的 token 成本（与可读规则故意不对称）
	require.Error(t, svc.AssertWritable(asUser(1), systemConv.Id), "系统会话只读")
	require.Error(t, svc.AssertWritable(context.Background(), userConv.Id))
}

// 会话标题兜底契约：未提供标题时落默认哨兵，不随运行类型生成文案。
//
// 无头运行曾按 RunKind 把「定时任务运行」这类中文写进标题：标题是一次性落库的展示文本，
// 英文界面会永远看到中文（切语言也不变），新增 RunKind 还必须回来改 case
func TestCreateConversationTitleFallback(t *testing.T) {
	svc := setupConversationSvc(t)
	ctx := asUser(1)

	blank, err := svc.CreateConversation(ctx, &dto.CreateConversationReq{})
	require.NoError(t, err)
	require.Equal(t, headless.DefaultConversationTitle, blank.Title, "未提供标题必须落默认哨兵（前端靠它识别「尚未命名」并用首条消息自动改名）")

	given, err := svc.CreateConversation(ctx, &dto.CreateConversationReq{Title: "每日运维日报"})
	require.NoError(t, err)
	require.Equal(t, "每日运维日报", given.Title, "调用方给的标题优先，不得被兜底覆盖")
}
