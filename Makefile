
docker:
	docker build . -t mayfly-go

# 构建 CLI 工具
.PHONY: build-cli
build-cli:
	cd cli && go mod tidy
	cd cli && go build -o ../build/mayfly-cli main.go

# 构建所有
.PHONY: build-all
build-all: build-cli
	@echo "All builds completed"

# 竞态检测门禁：覆盖后端基座、连接池与其主要业务模块。
#
# internal/agentops 必须在列：它是本仓并发面最大的模块（有界工作池、在途取消跟踪 map、
# 异步受控执行体、多 goroutine 认领），只跑普通 go test 无法证明取消/认领/清扫路径无竞态。
#
# 范围刻意用 ./internal/db/application（不带 /...）：其下的 transfer/itest 直连真实
# MySQL/PG/MSSQL/Oracle 实例，环境不齐时会整批 TestIT* 失败，属 itest target 的职责，
# 不是竞态门禁该承载的噪声。
#
# -p 1 串行化包级测试二进制：并发跑时多个包会互抢同一套共享方言库，表现为偶发主键冲突。
.PHONY: racetest
racetest:
	cd server && go test -race -count=1 -timeout 40m -p 1 ./pkg/... ./internal/flow/... ./internal/label/... ./internal/tag/... ./internal/alert/... ./internal/ai/... ./internal/agentops/... ./internal/sys/... ./internal/auth/... ./internal/db/application ./internal/machine/... ./internal/pkg/...

# 集成测试（需真实方言库）：两种跑法——
#   1) 复用宿主机已有实例（mysql:3306 / pg:5432 / 可选 mssql:11433 / oracle:1521）；
#   2) `make itest-env-up` 用 docker-compose.it.yaml 起齐四方言，用完 `make itest-env-down`。
# MySQL/PG/MSSQL 用测试进程独占随机库，TestMain 无条件删自建库，清理失败返回非零；
# 普通 go test 同样自动清理；SQLite 与导出文件用 t.TempDir；-p 1 控资源峰值。
# 强制杀进程/数据库离线可能中断清理（不自动删共享库或其他运行的临时库）。
# 缺席方言默认跳过；设 MAYFLY_IT_REQUIRED_DIALECTS=mssql,oracle 声明「必须真实执行」，连不上即失败，杜绝「永远跳过=误以为通过」。
.PHONY: itest
itest:
	cd server && go test -tags it -count=1 -timeout 40m -p 1 ./internal/db/...

# 起集成测试方言环境（oracle 首次就绪后需按 docker-compose.it.yaml 末尾说明建 mayfly 用户）
.PHONY: itest-env-up
itest-env-up:
	docker compose -f docker-compose.it.yaml up -d

.PHONY: itest-env-down
itest-env-down:
	docker compose -f docker-compose.it.yaml down -v

# 起环境 + 以「mssql,oracle 必须真实执行」跑全量集成测试
.PHONY: itest-all
itest-all:
	MAYFLY_IT_REQUIRED_DIALECTS=mssql,oracle $(MAKE) itest

