package dbtool

import (
	"fmt"
)

// connLevelFailureHint 连接层失败时补一条**可执行**的提示；其它情形返回 nil 保持原错。
//
// 存在动机（真机实测）：把实例显示名当 schema 传入时，MySQL 的 1044 Access denied 在**首次建连**
// 会被驱动吞成 `invalid connection`，于是第一次失败完全不携带任何提示，模型只能换个实例再撞一次
// （上一轮就是这样白跑 4 次、最后回答「抱歉，无法获取具体数字」）。
//
// 提示内容是一条已被验证能救回模型的动作：同一次任务里模型把 dbName 留空重调，正好命中
// ensureDbConn 里「多库资产必须指定库」那道守卫拿到可连库清单，最终答对了表数量。
// 所以这里不复述库名（连接都没建立，拿不到可靠清单），只把「留空再试」这条路指给它。
func connLevelFailureHint(dbName string, err error) error {
	if dbName == "" || !isConnectionLevelError(err) {
		return nil
	}
	return fmt.Errorf("%w；提示：本次失败发生在建立连接阶段（不是库名写错）。"+
		"可先把 dbName 留空再调一次，系统会列出该实例可连接的真实库名；若仍失败则是该凭证或网络不可用", err)
}
