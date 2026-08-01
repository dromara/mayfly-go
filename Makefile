
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
