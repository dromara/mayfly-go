# Mayfly-Go CLI

Mayfly-Go 命令行工具，用于在终端中连接和管理各种资源（数据库、机器 SSH、Redis 等）。

## 快速开始

### 构建

```bash
# 构建 CLI 工具
make build-cli

# 或手动构建
cd cli && go mod tidy
cd cli && go build -o ../build/mayfly-cli main.go
```

### 配置

创建配置文件 `~/.mayfly-cli.yaml`：

```yaml
server: http://localhost:8888
token: your-access-token-here
default_db: "1"
default_ssh: "1"
```

或者通过命令行参数指定：

```bash
mayfly-cli --server http://your-server:8888 --token your-token
```

## 使用示例

### 数据库操作

```bash
# 列出所有数据库
mayfly-cli db list

# 执行 SQL 查询
mayfly-cli db exec 1 mydb "SELECT * FROM users LIMIT 10"

# 指定服务器和 token
mayfly-cli db list --server http://localhost:8888 --token xxx
```

### SSH 连接

```bash
# 列出所有机器
mayfly-cli ssh list

# 在机器上执行命令
mayfly-cli ssh exec 1 "ls -la /tmp"

# 执行复杂命令
mayfly-cli ssh exec 1 "ps aux | grep nginx"
```

### Redis 操作

```bash
# 列出所有 Redis 实例
mayfly-cli redis list

# 执行 Redis 命令
mayfly-cli redis exec 1 0 "GET mykey"
mayfly-cli redis exec 1 0 "LRANGE mylist 0 -1"
```

## 命令参考

### 全局选项

```
--config string    配置文件路径 (默认: $HOME/.mayfly-cli.yaml)
-s, --server string   mayfly-go 服务端地址 (默认 "http://localhost:8888")
-t, --token string    访问令牌
-v, --verbose         详细输出
-h, --help            帮助信息
```

### db 命令

```bash
mayfly-cli db           # 数据库操作
mayfly-cli db list      # 列出可用数据库
mayfly-cli db exec      # 执行 SQL 语句
```

### ssh 命令

```bash
mayfly-cli ssh          # SSH 连接管理
mayfly-cli ssh list     # 列出可用机器
mayfly-cli ssh exec     # 在机器上执行命令
```

### redis 命令

```bash
mayfly-cli redis        # Redis 操作
mayfly-cli redis list   # 列出可用 Redis 实例
mayfly-cli redis exec   # 执行 Redis 命令
```

## 为 Agent 集成

CLI 工具设计为可被 AI Agent 通过终端调用：

```bash
# Agent 查询数据库
mayfly-cli db exec 1 production "SELECT COUNT(*) FROM orders WHERE status='pending'"

# Agent 检查机器状态
mayfly-cli ssh exec 5 "systemctl status nginx"

# Agent 查看 Redis 缓存
mayfly-cli redis exec 2 0 "GET session:user:12345"
```

### 输出格式

所有命令输出为标准格式，便于 Agent 解析：

- 表格形式展示列表
- JSON 格式展示详细结果
- 错误信息输出到 stderr

### 脚本集成

```bash
#!/bin/bash
# 示例：批量检查机器状态

for machine_id in 1 2 3 4 5; do
  echo "Checking machine $machine_id..."
  mayfly-cli ssh exec $machine_id "uptime"
done
```

## 架构说明

### 远程模式（当前实现）

CLI 通过 HTTP API 与 mayfly-go 服务端交互：

```
CLI (mayfly-cli) → HTTP API → mayfly-go Server → 资源（DB/SSH/Redis）
```

**优势：**
- 独立部署，无需后端配置
- 轻量级二进制文件
- 支持远程服务器

### 内嵌模式（未来扩展）

直接调用后端逻辑（需要完整的 mayfly-go 配置）：

```
CLI → 后端代码 → 资源（DB/SSH/Redis）
```

**优势：**
- 零网络延迟
- 支持离线操作
- 完整的后端功能

## 开发指南

### 目录结构

```
cli/
├── cmd/                    # 命令定义
│   ├── root.go            # 根命令
│   ├── db.go              # 数据库命令
│   ├── ssh.go             # SSH 命令
│   └── redis.go           # Redis 命令
├── client/                 # API 客户端
│   └── api_client.go      # HTTP API 封装
├── config/                 # 配置管理
│   └── config.go          # 配置文件加载/保存
├── main.go                 # 入口文件
└── go.mod                  # 依赖管理
```

### 添加新命令

1. 在 `cmd/` 目录下创建新文件（如 `mongo.go`）
2. 定义命令结构：

```go
var mongoCmd = &cobra.Command{
    Use:   "mongo",
    Short: "MongoDB 操作",
}
```

3. 在 `init()` 函数中注册：

```go
func init() {
    rootCmd.AddCommand(mongoCmd)
}
```

4. 在 `client/api_client.go` 中添加对应的 API 方法

### 依赖管理

```bash
# 添加新依赖
cd cli && go get github.com/spf13/cobra

# 整理依赖
cd cli && go mod tidy
```

## 安全注意事项

1. **Token 安全**：配置文件权限设置为 600
   ```bash
   chmod 600 ~/.mayfly-cli.yaml
   ```

2. **敏感操作**：生产环境建议启用二次确认

3. **网络传输**：建议使用 HTTPS 连接服务端

## 故障排查

### 连接失败

```bash
# 检查服务端是否可达
curl http://localhost:8888/api/dbs

# 检查 token 是否有效
mayfly-cli db list -v  # 详细输出
```

### 权限错误

确保 token 对应的账户有访问相应资源的权限。

### 依赖问题

```bash
cd cli
go mod tidy
go build
```

## 许可证

与 mayfly-go 项目保持一致。
