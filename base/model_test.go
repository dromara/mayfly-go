package base

import (
	"fmt"
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
	"mayfly-go/base/utils"
	"mayfly-go/controllers/vo"
	"mayfly-go/models"
	"strings"
	"testing"
)

type AccountDetailVO struct {
	Id       int64
	Username string
}

func init() {
	orm.RegisterDriver("mysql", orm.DRMySQL)

	orm.RegisterDataBase("default", "mysql", "root:111049@tcp(localhost:3306)/mayfly-go?charset=utf8")
	orm.Debug = true
}

func TestGetList(t *testing.T) {
	query := QuerySetter(new(models.Account)).OrderBy("-Id")
	list := new([]AccountDetailVO)
	GetList(query, new([]models.Account), list)
	fmt.Println(list)
}

func TestGetOne(t *testing.T) {
	model := new(models.Account)
	query := QuerySetter(model).Filter("Id", 2)
	adv := new(AccountDetailVO)
	GetOne(query, model, adv)
	fmt.Println(adv)
}

func TestMap(t *testing.T) {
	//o := getOrm()
	//
	////v := new([]Account)
	//var maps []orm.Params
	//_, err := o.Raw("SELECT a.Id, a.Username, r.Id AS 'Role.Id', r.Name AS 'Role.Name' FROM " +
	//	"t_account a JOIN t_role r ON a.id = r.account_id").Values(&maps)
	//fmt.Println(err)
	//////res := new([]Account)
	////model := &Account{}
	////o.QueryTable("t_account").Filter("id", 1).RelatedSel().One(model)
	////o.LoadRelated(model, "Role")
	res := new([]vo.AccountVO)
	sql := "SELECT a.Id, a.Username, r.Id AS 'Role.Id', r.Name AS 'Role.Name' FROM t_account a JOIN t_role r ON a.id = r.account_id"
	//limitSql := sql + " LIMIT 1, 3"
	//selectIndex := strings.Index(sql, "SELECT ") + 7
	//fromIndex := strings.Index(sql, " FROM")
	//selectCol := sql[selectIndex:fromIndex]
	//countSql := strings.Replace(sql, selectCol, "COUNT(*)", 1)
	//fmt.Println(limitSql)
	//fmt.Println(selectCol)
	//fmt.Println(countSql)
	page := GetPageBySql(sql, res, &PageParam{PageNum: 1, PageSize: 1})
	fmt.Println(page)
	//return res
}

func TestCase2Camel(t *testing.T) {
	fmt.Println(utils.Case2Camel("create_time"))
	fmt.Println(strings.Title("username"))
}
