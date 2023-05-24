package application

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils"

	"gorm.io/gorm"
)

type Account interface {
	GetAccount(condition *entity.Account, cols ...string) error

	GetPageList(condition *entity.Account, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	Create(account *entity.Account)

	Update(account *entity.Account)

	Delete(id uint64)
}

func newAccountApp(accountRepo repository.Account) Account {
	return &accountAppImpl{
		accountRepo: accountRepo,
	}
}

type accountAppImpl struct {
	accountRepo repository.Account
}

// 根据条件获取账号信息
func (a *accountAppImpl) GetAccount(condition *entity.Account, cols ...string) error {
	return a.accountRepo.GetAccount(condition, cols...)
}

func (a *accountAppImpl) GetPageList(condition *entity.Account, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return a.accountRepo.GetPageList(condition, pageParam, toEntity)
}

func (a *accountAppImpl) Create(account *entity.Account) {
	biz.IsTrue(a.GetAccount(&entity.Account{Username: account.Username}) != nil, "该账号用户名已存在")
	// 默认密码为账号用户名
	account.Password = utils.PwdHash(account.Username)
	account.Status = entity.AccountEnableStatus
	a.accountRepo.Insert(account)
}

func (a *accountAppImpl) Update(account *entity.Account) {
	// 禁止更新用户名，防止误传被更新
	account.Username = ""
	a.accountRepo.Update(account)
}

func (a *accountAppImpl) Delete(id uint64) {
	err := model.Tx(
		func(db *gorm.DB) error {
			// 删除account表信息
			return db.Delete(new(entity.Account), "id = ?", id).Error
		},
		func(db *gorm.DB) error {
			// 删除账号关联的角色信息
			accountRole := &entity.AccountRole{AccountId: id}
			return db.Where(accountRole).Delete(accountRole).Error
		},
	)
	biz.ErrIsNilAppendErr(err, "删除失败：%s")
}
