<div align="center">

# 🌈 Dromara mayfly-go

**一个高度集成的 Web 版「统一资源管理操作平台」—— 让机器、数据库、缓存、搜索、容器、消息队列与 AI 智能运维，都在同一个入口完成。**

<p>
    <a href="./README.md">简体中文</a> |
    <a href="./README_EN.md">English</a> |
    <a href="https://www.yuque.com/may-fly/mayfly-go">项目文档</a> |
    <a href="https://space.bilibili.com/484091081/channel/collectiondetail?sid=392854">操作视频</a> |
    <a href="http://go.mayfly.run">在线演示</a>
</p>

<p>
  <a href="https://gitee.com/dromara/mayfly-go" target="_blank">
    <img src="https://gitee.com/dromara/mayfly-go/badge/star.svg?theme=white" alt="star"/>
    <img src="https://gitee.com/dromara/mayfly-go/badge/fork.svg" alt="fork"/>
  </a>
  <a href="https://github.com/dromara/mayfly-go" target="_blank">
    <img src="https://img.shields.io/github/stars/dromara/mayfly-go.svg?style=social" alt="github star"/>
    <img src="https://img.shields.io/github/forks/dromara/mayfly-go.svg?style=social" alt="github fork"/>
  </a>
  <a href="https://gitcode.com/dromara/mayfly-go" target="_blank">
    <img src="https://gitcode.com/dromara/mayfly-go/star/badge.svg" alt="gitcode star"/>
  </a>
  <a href="https://hub.docker.com/r/mayflygo/mayfly-go/tags" target="_blank">
    <img src="https://img.shields.io/docker/pulls/mayflygo/mayfly-go.svg?label=docker%20pulls&color=fac858" alt="docker pulls"/>
  </a>
  <a href="https://github.com/golang/go" target="_blank">
    <img src="https://img.shields.io/badge/Golang-1.26%2B-yellow.svg" alt="golang"/>
  </a>
  <a href="https://cn.vuejs.org" target="_blank">
    <img src="https://img.shields.io/badge/Vue-3.x-green.svg" alt="vue"/>
  </a>
  <a href="./LICENSE" target="_blank">
    <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="license"/>
  </a>
</p>

</div>

---

## 📖 前言

**mayfly-go** 把日常运维与开发中散落的工具收敛到一个平台里：以「**资源树**」为统一底座，向下挂载机器、数据库、缓存、搜索引擎、向量库、消息队列与容器；对每一类资源提供 **终端 / 文件 / SQL / 数据 / 容器 / 命令** 的可视化操作能力。

在此基础上，平台原生内嵌 **工单流程审批**、**命令安全拦截**、**告警中心**、**多渠道消息通知（含企业微信 / 钉钉 / 飞书）**，并引入 **AI 智能助手** 与 **AI 智能运维编排（AgentOps）**，形成"操作 → 审批 → 拦截 → 通知 → 智能自愈"的一站式闭环。

> 🎯 适合人群：运维工程师、DBA、后端 / 全栈开发者，以及希望把重复运维动作交给 AI 自动执行的团队。

---

## ✨ 核心特性一览

<table>
  <tr>
    <td width="50%" valign="top">🖥️ <b>机器全栈运维</b><br/><sub>Web 终端、终端录像回放、命令拦截、RDP 远程桌面、文件管理、脚本执行、计划任务、进程监控、机器指标、批量执行。</sub></td>
    <td width="50%" valign="top">🗄️ <b>多数据源操作</b><br/><sub>关系型 / 文档 / 缓存 / 搜索 / 向量 / 消息队列全覆盖，SQL 编辑器、数据网格在线编辑、导出、脱敏、会话管理。</sub></td>
  </tr>
  <tr>
    <td width="50%" valign="top">🔁 <b>数据同步与迁移</b><br/><sub>跨库结构比对（Schema Diff）、增量/全量数据同步、数据库迁移任务、校验与回滚 SQL。</sub></td>
    <td width="50%" valign="top">🛡️ <b>命令安全拦截</b><br/><sub>黑名单正则规则 + 触发策略（禁止 / 审批 / 仅提醒），终端、API、AI Agent 三入口统一校验。</sub></td>
  </tr>
  <tr>
    <td width="50%" valign="top">🤖 <b>AI 智能助手</b><br/><sub>流式对话、工具调用可视化、中断审批；只读命令自动放行、危险动作强制人工确认（fail-closed）。</sub></td>
    <td width="50%" valign="top">🧠 <b>AI 智能运维 AgentOps</b><br/><sub>运维任务编排、事件触发、变更提案审批闭环、执行历史与离线评测、治理急停开关。</sub></td>
  </tr>
  <tr>
    <td width="50%" valign="top">📋 <b>工单流程审批</b><br/><sub>可视化流程定义、条件组、我的任务 / 我的流程，操作可审批后执行，全链路留痕。</sub></td>
    <td width="50%" valign="top">🚨 <b>告警中心</b><br/><sub>告警规则、事件确认、静默、抑制、升级策略、通知策略与概览看板。</sub></td>
  </tr>
  <tr>
    <td width="50%" valign="top">💌 <b>多渠道消息通知</b><br/><sub>站内信、WebSocket、邮件、<b>企业微信</b>、钉钉、飞书机器人；签名密钥安全托管，消息模板可复用。</sub></td>
    <td width="50%" valign="top">🔐 <b>资源与权限底座</b><br/><sub>统一资源树、授权凭证密文托管、标签 / 团队、RBAC 菜单权限、操作日志审计。</sub></td>
  </tr>
</table>

---

## 🧩 功能全景

<details open>
<summary><b>🖥️ 机器（Linux）运维</b></summary>

| 功能 | 说明 |
| :--- | :--- |
| 机器管理 | 机器列表、连接测试、启用停用、SSH 隧道、按资源树授权访问范围 |
| Web 终端 | 浏览器交互式 SSH，支持 Vim 等 TUI、命令逐条解析 |
| 终端录像回放 | 会话全程录制（asciinema），支持回放与执行命令记录 |
| **命令拦截 / 安全配置** | 黑名单正则规则 + 触发策略处置级别（禁止 / 需审批 / 仅提醒），终端、API、AI 三入口共用同一前置校验 |
| RDP 远程桌面 | 基于 Apache Guacamole 的 Windows 远程桌面接入 |
| 文件管理 | 目录浏览、创建、上传、下载、在线编辑与内容查看、文件配置 |
| 脚本管理 | 脚本保存与一键执行、执行记录 |
| 计划任务 | Cron 表达式配置、任务启停、执行历史 |
| 进程监控 | 实时进程列表、终止进程 |
| 机器指标 | CPU / 内存 / 磁盘 / 网卡等运行指标采集与统计、磁盘分析 |
| 主机密钥 | known_hosts 主机密钥管理，防中间人 |
| 批量命令 | 多机批量命令执行（`batch-run-cmd`） |

</details>

<details open>
<summary><b>🗄️ 数据库（DBMS）</b></summary>

| 功能 | 说明 |
| :--- | :--- |
| 多方言支持 | MySQL、PostgreSQL、Oracle、SQL Server、达梦、高斯、SQLite、ClickHouse |
| SQL 编辑器 | 多方言 SQL 补全、多结果页签、执行计划、SQL 分类与安全守卫 |
| 数据操作 | 选表查数据、数据网格在线增删改、结果导出 |
| 表结构编辑 | DDL 建表 / 改表、索引管理 |
| 结构比对 | 跨库 / 跨实例 Schema Diff |
| 数据同步 | 增量 / 全量同步任务，启停控制、同步日志 |
| 数据库迁移 | 迁移任务、校验、迁移文件执行 / 下载 / 删除、运行日志 |
| 数据脱敏 | 脱敏规则配置，敏感列展示与导出脱敏 |
| 会话管理 | 活动会话查看、取消会话 |
| 执行审计 | SQL 执行日志、回滚 SQL 生成 |

</details>

<details open>
<summary><b>🧱 缓存 / 搜索 / 向量 / 文档库</b></summary>

| 模块 | 功能 |
| :--- | :--- |
| **Redis** | 实例管理、键浏览、JSON / 文本查看器、数据增删改、控制台命令、服务器 INFO |
| **MongoDB** | 命令执行、聚合查询、批量操作、文档视图、集合 / 索引 DDL、数据导入导出、账号与角色管理 |
| **Elasticsearch** | 实例管理、DSL 查询、索引数据操作 |
| **Milvus** | 向量库连接、集合与向量数据操作 |

</details>

<details open>
<summary><b>📨 消息队列 Kafka & 🐳 容器 Docker</b></summary>

| 模块 | 功能 |
| :--- | :--- |
| **Kafka** | 实例管理、Topic 创建/删除/配置、分区扩容、Broker 与配置查看、消费者组（成员 / 删除）、消息生产与消费 |
| **Docker** | 容器列表与状态统计、启动 / 停止 / 重启 / 删除 / 创建、进入容器终端、实时日志、镜像管理（删除 / 导出 / 导入） |

</details>

<details open>
<summary><b>🤖 AI 智能助手 & 🧠 智能运维 AgentOps</b></summary>

**AI 智能助手（嵌入平台侧栏）**

- 流式对话、工具调用过程可视化、执行中断与人工审批。
- 内置工具：`机器命令执行`、`SQL 执行`、`查询数据`、`查询表 / 表结构`、`监控指标`、`资源列表`、`技能读取`、`修复提案`。
- 安全基线：白名单只读命令自动执行，链式命令逐段校验，命令替换 / 重定向 / Shell 逃逸命令一律需审批（fail-closed）。
- AI 设置、集成、插件管理与会话记忆。

**AI 智能运维 AgentOps（自动化编排）**

- **运维任务**：任务编排与手动触发；**事件触发器**：告警源 / Webhook 源自动拉起。
- **执行历史**：运行详情与步骤 Trace、取消运行、发起编排、离线评测、运行反馈。
- **变更提案**：AI 生成修复提案 → 审批放行 → 执行的闭环链路。
- **运维治理**：治理看板、失速（stalled）检测、**急停开关**、以他人身份运行。

</details>

<details open>
<summary><b>🚨 告警中心 & 💌 消息通知</b></summary>

| 功能 | 说明 |
| :--- | :--- |
| 告警概览 | 告警态势看板与分布统计 |
| 告警规则 | 指标评估规则配置（可扩展评估器） |
| 告警事件 | 事件列表、确认（Ack）、状态流转 |
| 静默 / 抑制 | 告警静默规则、告警抑制规则 |
| 升级 / 通知策略 | 告警升级策略、通知策略编排 |
| **消息渠道** | 站内信、WebSocket、邮件、**企业微信机器人**、钉钉机器人、飞书机器人（加签密钥安全托管，不随列表外溢） |
| 消息模板 / 发送 | 可复用消息模板、手动发送消息 |

</details>

<details open>
<summary><b>📋 工单流程审批 & 🔐 系统管理</b></summary>

| 功能 | 说明 |
| :--- | :--- |
| 流程定义 | 可视化编排审批流程、启用停用 |
| 流程条件组 | 规则条件组（Rule Segment）配置 |
| 我的任务 / 我的流程 | 待办审批、发起与跟踪流程实例 |
| 资源树 / 授权凭证 | 统一资源层级树、SSH / DB 凭证密文托管（密文查看权限分离） |
| 标签 / 团队 | 标签管理与团队、成员管理 |
| 账号 / 角色 / 菜单权限 | RBAC 权限体系、角色分配、菜单权限分配 |
| 系统配置 / 操作日志 | 全局配置、操作审计日志 |
| 个人中心 | 个人信息、我的资源与基本权限 |

</details>

---

## 🗄️ 支持的数据库与中间件

#### 关系型数据库

| 数据库类型 | 支持状态 | 数据库类型 | 支持状态 | 数据库类型 | 支持状态 |
| :--------: | :------: | :--------: | :------: | :--------: | :------: |
|   MySQL    |    ✅    | PostgreSQL |    ✅    |   Oracle   |    ✅    |
| SQL Server |    ✅    |    达梦    |    ✅    |    高斯    |    ✅    |
|   SQLite   |    ✅    | ClickHouse |    ✅    |            |          |

#### 非关系型数据库 / 中间件

| 数据库类型    | 支持状态 | 数据库类型 | 支持状态 |
| :-----------: | :------: | :--------: | :------: |
|    MongoDB    |    ✅    |   Redis    |    ✅    |
| Elasticsearch |    ✅    |   Milvus   |    ✅    |
|    Kafka      |    ✅    |   Docker   |    ✅    |

---

## 🛠️ 技术栈

| 层次 | 技术选型 |
| :--- | :--- |
| 前端 | TypeScript、Vue 3 (Composition API)、Vite、Element Plus、Tailwind CSS、Pinia |
| 后端 | Go 1.26+、Gin、GORM、Clean Architecture 分层 |
| 远程桌面 | Apache Guacamole（RDP） |
| 终端录制 | Asciinema 回放 |
| 部署 | 单二进制内置前端静态资源、Docker / Docker Compose |

---

## 🚀 快速开始

### 方式一：Docker（推荐）

```bash
# 拉取并运行（默认端口 18888，浏览器访问 http://ip:18888）
docker run -d -p 18888:18888 --name mayfly-go docker.io/mayflygo/mayfly-go:latest
```

### 方式二：Docker Compose

```bash
git clone https://github.com/dromara/mayfly-go.git
cd mayfly-go
docker-compose up -d
```

### 方式三：源码启动

```bash
# 后端（默认读取 server/config.yml，端口 18888）
cd server && go run main.go

# 前端（开发调试）
cd frontend && pnpm install && pnpm dev
```

> 📚 数据库初始化、配置项、反向代理等详细部署说明请参见 [项目文档](https://www.yuque.com/may-fly/mayfly-go)。

---

## 🎮 演示环境

| 地址 | 账号 / 密码 |
| :--- | :--- |
| http://go.mayfly.run | `test` / `test123.` |

---

## 🖼️ 系统核心功能截图

> 截图按功能模块归档于 `docs/images/<模块>/`，均为亮色模式实拍。

### 🏠 首页工作台 · `home/`

<div align="center"><img src="./docs/images/home/overview.jpg" alt="首页工作台" width="820"/></div>

### 🌳 资源与权限 · `resource/`

<table>
  <tr>
    <td align="center" width="50%"><b>统一资源树</b><br/><img src="./docs/images/resource/resource-tree.jpg" alt="资源树" width="100%"/></td>
    <td align="center" width="50%"><b>我的资源</b><br/><img src="./docs/images/resource/my-resource.jpg" alt="我的资源" width="100%"/></td>
  </tr>
</table>

### 🖥️ 机器运维 · `machine/`

<table>
  <tr>
    <td align="center" width="33%"><b>Web 终端（实时 SSH）</b><br/><img src="./docs/images/machine/terminal.jpg" alt="终端" width="100%"/></td>
    <td align="center" width="33%"><b>文件管理器</b><br/><img src="./docs/images/machine/file-manager.jpg" alt="文件管理" width="100%"/></td>
    <td align="center" width="33%"><b>命令安全拦截</b><br/><img src="./docs/images/machine/cmd-security.jpg" alt="命令拦截" width="100%"/></td>
  </tr>
</table>

### 🗄️ 数据库 / 缓存 / 文档 / 容器 · `db/` `redis/` `mongo/` `docker/`

<table>
  <tr>
    <td align="center" width="50%"><b>SQL 编辑器（Monaco）</b><br/><img src="./docs/images/db/sql-editor.jpg" alt="SQL编辑器" width="100%"/></td>
    <td align="center" width="50%"><b>数据表 · 可视化编辑</b><br/><img src="./docs/images/db/table-data.jpg" alt="数据表" width="100%"/></td>
  </tr>
  <tr>
    <td align="center"><b>数据同步</b><br/><img src="./docs/images/db/sync.jpg" alt="数据同步" width="100%"/></td>
    <td align="center"><b>数据库迁移</b><br/><img src="./docs/images/db/transfer.jpg" alt="数据库迁移" width="100%"/></td>
  </tr>
  <tr>
    <td align="center"><b>数据脱敏</b><br/><img src="./docs/images/db/mask.jpg" alt="数据脱敏" width="100%"/></td>
    <td align="center"><b>Redis 键值操作</b><br/><img src="./docs/images/redis/key-value.jpg" alt="Redis" width="100%"/></td>
  </tr>
  <tr>
    <td align="center" colspan="2"><b>MongoDB 集合文档数据</b><br/><img src="./docs/images/mongo/data.jpg" alt="MongoDB" width="60%"/></td>
  </tr>
  <tr>
    <td align="center" colspan="2"><b>容器管理（运行状态 / 资源 / 端口 / 日志）</b><br/><img src="./docs/images/docker/containers.jpg" alt="容器" width="60%"/></td>
  </tr>
</table>

### 🚨 告警中心 · `alert/`　📋 工单流程 · `flow/`

<table>
  <tr>
    <td align="center" width="50%"><b>告警概览看板</b><br/><img src="./docs/images/alert/overview.jpg" alt="告警概览" width="100%"/></td>
    <td align="center" width="50%"><b>告警规则</b><br/><img src="./docs/images/alert/rules.jpg" alt="告警规则" width="100%"/></td>
  </tr>
  <tr>
    <td align="center"><b>工单 · 我的待办任务</b><br/><img src="./docs/images/flow/tasks.jpg" alt="工单任务" width="100%"/></td>
    <td align="center"><b>流程定义</b><br/><img src="./docs/images/flow/procdef.jpg" alt="流程定义" width="100%"/></td>
  </tr>
</table>

### 🤖 AI 智能助手 & 🧠 AgentOps · `ai/` `agentops/`　📬 消息通知 · `msg/`

<table>
  <tr>
    <td align="center" width="50%"><b>AI 助手（工具调用 / 执行报告）</b><br/><img src="./docs/images/ai/assistant.jpg" alt="AI助手" width="100%"/></td>
    <td align="center" width="50%"><b>运维治理（急停 / KPI / 运行分布）</b><br/><img src="./docs/images/agentops/governance.jpg" alt="运维治理" width="100%"/></td>
  </tr>
  <tr>
    <td align="center" colspan="2"><b>消息渠道（企业微信 / 钉钉 / 飞书 / 邮件）</b><br/><img src="./docs/images/msg/channels.jpg" alt="消息渠道" width="60%"/></td>
  </tr>
</table>

### ⚙️ 系统管理 · `system/`

<table>
  <tr>
    <td align="center" width="33%"><b>账号管理</b><br/><img src="./docs/images/system/account.jpg" alt="账号管理" width="100%"/></td>
    <td align="center" width="33%"><b>角色管理</b><br/><img src="./docs/images/system/role.jpg" alt="角色管理" width="100%"/></td>
    <td align="center" width="33%"><b>菜单权限</b><br/><img src="./docs/images/system/menu.jpg" alt="菜单权限" width="100%"/></td>
  </tr>
</table>

> **Elasticsearch / 容器 / Kafka 等操作视图同样从统一资源树入口进入；更多功能与操作指南可查看项目文档。**

---

## 💬 交流及问题反馈

加入 QQ 群交流：[119699946](https://qm.qq.com/cgi-bin/qm/qr?k=IdJSHW0jTMhmWFHBUS9a83wxtrxDDhFj&jump_from=webapi)

---

## 🌟 Star History

如果项目对你有帮助，欢迎在 <a href="https://github.com/dromara/mayfly-go" target="_blank">Github</a>、<a href="https://gitee.com/dromara/mayfly-go" target="_blank">Gitee</a> 或 <a href="https://gitcode.com/dromara/mayfly-go" target="_blank">Gitcode</a> 点亮 ⭐ Star，这是对项目最大的鼓励与支持！

<div align="center">

> 喝杯咖啡 ☕️ 或者来杯奶茶 🧋，让作者更有精神，写出更棒的代码！

<img class="no-margin" src="https://foruda.gitee.com/images/1744113367791412282/36a3c23b_1240250.png" alt="微信打赏" width="200" height="200">

**特别感谢：** 赞助金额达 199 元以上，加微信（wx-error），可受邀进入付费交流群，享受更快、更优先的技术支持与交流服务！

</div>
