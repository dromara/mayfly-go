package application

import (
	"context"
	"testing"

	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/domain/repository"
	"mayfly-go/internal/msg/infra/persistence"
	"mayfly-go/internal/msg/msgx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupMsgChannelRepo 打开 sqlite 内存库并装配绑定真实仓储的渠道仓储。
//
// 用真实仓储而不是 mock：本组用例钉的就是「条件到底有没有拼进 SQL」，mock 掉的仓储会恒返回
// 调用方给它的条件，等于把断言写成同义反复
func setupMsgChannelRepo(t *testing.T) repository.MsgChannel {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:               logger.Default.LogMode(logger.Silent),
		NamingStrategy:       gormx.MetaNamingStrategy(),
		DisableAutomaticPing: true,
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	// :memory: 每连接独立库，锁死单连接避免建表与读写落在不同连接
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	require.NoError(t, db.AutoMigrate(&entity.MsgChannel{}))

	prev := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = prev })

	registerMsgRepoOnce.Do(persistence.InitIoc)
	repo := ioc.Get[repository.MsgChannel]()
	require.NotNil(t, repo)

	ctx := context.Background()
	for _, name := range []string{"本地钉钉验证", "test-企微", "运维邮件通知"} {
		require.NoError(t, repo.Insert(ctx, &entity.MsgChannel{
			Name: name, Code: "code-" + name, Type: msgx.ChannelTypeDingBot, Status: entity.ChannelStatusEnable,
		}))
	}
	return repo
}

// TestChannelListFiltersAreApplied 渠道列表声明的两个筛选维度必须真的收窄结果。
//
// 存在理由（本轮真机前读代码抓到）：渠道列表的「名称」搜索框发得出 `?name=xxx`，但 api 层
// 不读任何查询参数、直接拿空实体去查，仓储侧也压根没有 name 这一维——两处各自缺一块，症状却
// 一模一样：接口照返 200、列表返回全量，而界面搜索框还在，用户只会以为筛选不区分或「都长这样」。
// 这类缺陷编译、类型检查、lint 全绿，只能把「搜索维度 ↔ 谓词」钉成用例
func TestChannelListFiltersAreApplied(t *testing.T) {
	repo := setupMsgChannelRepo(t)
	page := model.PageParam{PageNum: 1, PageSize: 10}

	all, err := repo.GetPageList(&entity.MsgChannel{}, page)
	require.NoError(t, err)
	require.Equal(t, int64(3), all.Total, "空条件不加过滤（基座零值跳过语义），返回全量才对")

	byName, err := repo.GetPageList(&entity.MsgChannel{Name: "钉钉"}, page)
	require.NoError(t, err)
	require.Equal(t, int64(1), byName.Total, "按名称搜「钉钉」只该命中一条；返回 3 条说明这个搜索维度是死的")
	require.Equal(t, "本地钉钉验证", byName.List[0].Name)

	byCode, err := repo.GetPageList(&entity.MsgChannel{Code: "code-运维"}, page)
	require.NoError(t, err)
	require.Equal(t, int64(1), byCode.Total, "按编码搜同样必须收窄")
	require.Equal(t, "运维邮件通知", byCode.List[0].Name)

	// 无匹配时得是空集，而不是退回全量：否则「搜不到」和「搜索坏了」在界面上不可区分
	none, err := repo.GetPageList(&entity.MsgChannel{Name: "不存在的渠道名"}, page)
	require.NoError(t, err)
	require.Equal(t, int64(0), none.Total)
}

// TestChannelSaveAssignsCodeOnlyOnCreate 编码只在新建时生成：编辑不得换码。
//
// 模板与告警侧是按 code 引用渠道的，编辑一次就换码等于把既有引用全部指向空气
func TestChannelSaveAssignsCodeOnlyOnCreate(t *testing.T) {
	setupMsgChannelRepo(t)
	app := &msgChannelAppImpl{}
	app.Repo = ioc.Get[repository.MsgChannel]()
	ctx := context.Background()

	created := &entity.MsgChannel{Name: "新建渠道", Type: msgx.ChannelTypeDingBot, Status: entity.ChannelStatusEnable}
	require.NoError(t, app.SaveChannel(ctx, created))
	require.Len(t, created.Code, 8, "新建必须自动生成编码")

	codeBefore := created.Code
	created.Name = "改名后的渠道"
	require.NoError(t, app.SaveChannel(ctx, created))
	after, err := app.GetById(created.Id)
	require.NoError(t, err)
	require.Equal(t, codeBefore, after.Code, "编辑不得改编码：下游按 code 引用该渠道")
	require.Equal(t, "改名后的渠道", after.Name)
}
