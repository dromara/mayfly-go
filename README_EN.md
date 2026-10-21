<div align="center">

# 🌈 Dromara mayfly-go

**A highly integrated, web-based "Unified Resource Management & Operation Platform" — machines, databases, caches, search engines, containers, message queues and AI-driven operations, all in a single entry point.**

<p>
    <a href="./README.md">中文介绍</a> |
    <a href="./README_EN.md">English</a> |
    <a href="https://www.yuque.com/may-fly/mayfly-go">Documentation</a> |
    <a href="https://space.bilibili.com/484091081/channel/collectiondetail?sid=392854">Operate Video</a> |
    <a href="http://go.mayfly.run">Live Demo</a>
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
  <a href="https://vuejs.org" target="_blank">
    <img src="https://img.shields.io/badge/Vue-3.x-green.svg" alt="vue"/>
  </a>
  <a href="./LICENSE" target="_blank">
    <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="license"/>
  </a>
</p>

</div>

---

## 📖 Introduction

**mayfly-go** consolidates the tools scattered across day-to-day operations and development into a single platform. Built on a unified **resource tree**, it mounts machines, databases, caches, search engines, vector stores, message queues and containers, and provides visual **terminal / file / SQL / data / container / command** operations for every resource type.

On top of that, the platform natively embeds **work-order process approval**, **command security interception**, an **alert center** and **multi-channel notifications (incl. WeCom / DingTalk / Feishu)**, and introduces an **AI assistant** and **AI-driven operations orchestration (AgentOps)** — forming an all-in-one loop of "operate → approve → intercept → notify → self-heal".

> 🎯 Who it's for: operations engineers, DBAs, backend / full-stack developers, and teams looking to hand repetitive ops tasks to AI automation.

---

## ✨ Core Features

<table>
  <tr>
    <td width="50%" valign="top">🖥️ <b>Full-stack Machine Ops</b><br/><sub>Web terminal, terminal session recording & replay, command interception, RDP remote desktop, file management, script execution, cron jobs, process monitoring, machine metrics, batch execution.</sub></td>
    <td width="50%" valign="top">🗄️ <b>Multi-source Data Operations</b><br/><sub>Relational / document / cache / search / vector / message queues fully covered, SQL editor, inline data-grid editing, export, masking, session management.</sub></td>
  </tr>
  <tr>
    <td width="50%" valign="top">🔁 <b>Data Sync & Migration</b><br/><sub>Cross-database schema diff, incremental / full data sync, database migration tasks, verification and rollback SQL.</sub></td>
    <td width="50%" valign="top">🛡️ <b>Command Security Interception</b><br/><sub>Blacklist regex rules + trigger policies (forbid / approve / warn-only), with terminal, API and AI Agent sharing one unified pre-check.</sub></td>
  </tr>
  <tr>
    <td width="50%" valign="top">🤖 <b>AI Assistant</b><br/><sub>Streaming chat, tool-call visualization, interrupt & approval; read-only commands auto-run, dangerous actions require human confirmation (fail-closed).</sub></td>
    <td width="50%" valign="top">🧠 <b>AI Ops Orchestration (AgentOps)</b><br/><sub>Ops task orchestration, event triggers, change-proposal approval loop, run history & offline evaluation, governance kill switch.</sub></td>
  </tr>
  <tr>
    <td width="50%" valign="top">📋 <b>Work-order Process Approval</b><br/><sub>Visual process definition, condition groups, my tasks / my processes, approve-before-execute with full audit trail.</sub></td>
    <td width="50%" valign="top">🚨 <b>Alert Center</b><br/><sub>Alert rules, event acknowledgment, silences, inhibitions, escalation policies, notification policies and an overview dashboard.</sub></td>
  </tr>
  <tr>
    <td width="50%" valign="top">💌 <b>Multi-channel Notifications</b><br/><sub>Site messages, WebSocket, email, <b>WeCom</b>, DingTalk and Feishu bots; signing secrets safely hosted, reusable message templates.</sub></td>
    <td width="50%" valign="top">🔐 <b>Resource & Permission Foundation</b><br/><sub>Unified resource tree, ciphertext credential hosting, labels / teams, RBAC menu permissions, operation-log auditing.</sub></td>
  </tr>
</table>

---

## 🧩 Feature Overview

<details open>
<summary><b>🖥️ Machine (Linux) Operations</b></summary>

| Feature | Description |
| :--- | :--- |
| Machine Management | Machine list, connection test, enable/disable, SSH tunnel, resource-tree-based access scoping |
| Web Terminal | Interactive SSH in the browser, TUI apps like Vim supported, per-command parsing |
| Terminal Recording & Replay | Full session recording (asciinema), with replay and executed-command log |
| **Command Interception / Security Config** | Blacklist regex rules + trigger-policy handling levels (forbid / needs-approval / warn-only); terminal, API and AI share one pre-check |
| RDP Remote Desktop | Windows remote desktop via Apache Guacamole |
| File Management | Directory browsing, create, upload, download, inline edit & content view, file configs |
| Script Management | Save scripts and run with one click, execution records |
| Cron Jobs | Cron expression config, task start/stop, execution history |
| Process Monitoring | Real-time process list, kill process |
| Machine Metrics | CPU / memory / disk / network metric collection & stats, disk analysis |
| Host Keys | known_hosts management, MITM protection |
| Batch Commands | Multi-machine batch command execution (`batch-run-cmd`) |

</details>

<details open>
<summary><b>🗄️ Databases (DBMS)</b></summary>

| Feature | Description |
| :--- | :--- |
| Multi-dialect Support | MySQL, PostgreSQL, Oracle, SQL Server, Dameng, Gauss, SQLite, ClickHouse |
| SQL Editor | Multi-dialect completion, multiple result tabs, execution plans, SQL classification & security guards |
| Data Operations | Select-table-to-query, inline data-grid add/edit/delete, result export |
| Table Structure Editing | DDL create/alter table, index management |
| Schema Diff | Cross-database / cross-instance schema comparison |
| Data Sync | Incremental / full sync tasks, start/stop control, sync logs |
| Database Migration | Migration tasks, verification, migration file run / download / delete, run logs |
| Data Masking | Masking rule config, sensitive-column display & export masking |
| Session Management | View active sessions, cancel sessions |
| Execution Audit | SQL execution logs, rollback SQL generation |

</details>

<details open>
<summary><b>🧱 Cache / Search / Vector / Document Stores</b></summary>

| Module | Features |
| :--- | :--- |
| **Redis** | Instance management, key browsing, JSON / text viewers, data add/edit/delete, console commands, server INFO |
| **MongoDB** | Command execution, aggregate queries, batch operations, document view, collection / index DDL, data import/export, account & role management |
| **Elasticsearch** | Instance management, DSL queries, index data operations |
| **Milvus** | Vector DB connection, collection & vector data operations |

</details>

<details open>
<summary><b>📨 Message Queue Kafka & 🐳 Container Docker</b></summary>

| Module | Features |
| :--- | :--- |
| **Kafka** | Instance management, topic create/delete/config, partition scale-up, broker & config viewing, consumer groups (members / delete), produce & consume messages |
| **Docker** | Container list & stats, start / stop / restart / remove / create, in-container terminal, live logs, image management (remove / export / import) |

</details>

<details open>
<summary><b>🤖 AI Assistant & 🧠 AI Ops Orchestration (AgentOps)</b></summary>

**AI Assistant (embedded in the platform sidebar)**

- Streaming chat, tool-call process visualization, execution interrupts and manual approval.
- Built-in tools: `machine command exec`, `SQL exec`, `query data`, `query tables / table DDL`, `monitor metrics`, `resource list`, `skill read`, `propose remediation`.
- Security baseline: whitelisted read-only commands auto-run, chained commands validated segment-by-segment, and command substitution / redirection / shell-escape commands always require approval (fail-closed).
- AI settings, integrations, plugin management and session memory.

**AI Ops Orchestration (AgentOps, automated orchestration)**

- **Ops Tasks**: task orchestration & manual triggering; **Event Triggers**: alert source / webhook source auto-invocation.
- **Run History**: run detail with step trace, cancel run, start orchestration, offline evaluation, run feedback.
- **Change Proposals**: AI-generated remediation proposal → approval → execution closed loop.
- **Governance**: governance dashboard, stalled detection, **kill switch**, run-as-another-identity.

</details>

<details open>
<summary><b>🚨 Alert Center & 💌 Notifications</b></summary>

| Feature | Description |
| :--- | :--- |
| Alert Overview | Alert posture dashboard and distribution stats |
| Alert Rules | Metric evaluation rule config (extensible evaluators) |
| Alert Events | Event list, acknowledgment, status transitions |
| Silence / Inhibition | Alert silence rules, alert inhibition rules |
| Escalation / Notification Policy | Alert escalation policies, notification policy orchestration |
| **Message Channels** | Site messages, WebSocket, email, **WeCom bot**, DingTalk bot, Feishu bot (signing secrets safely hosted, never leaked via list endpoints) |
| Templates / Send | Reusable message templates, manual message sending |

</details>

<details open>
<summary><b>📋 Process Approval & 🔐 System Management</b></summary>

| Feature | Description |
| :--- | :--- |
| Process Definition | Visual approval-flow orchestration, enable/disable |
| Rule Segment | Rule condition-group (Rule Segment) configuration |
| My Tasks / My Processes | Pending approvals, start & track process instances |
| Resource Tree / Auth Certs | Unified resource hierarchy tree, SSH / DB credential ciphertext hosting (ciphertext view permission separated) |
| Labels / Teams | Label management, teams and member management |
| Account / Role / Menu Permissions | RBAC permission model, role allocation, menu-permission allocation |
| System Config / Operation Log | Global configuration, operation audit logs |
| Personal Center | Personal info, my resources & base permissions |

</details>

---

## 🗄️ Supported Databases & Middleware

#### Relational Databases

|  Database  | Status |  Database  | Status | Database | Status |
| :--------: | :----: | :--------: | :----: | :------: | :----: |
|   MySQL    |   ✅   | PostgreSQL |   ✅   |  Oracle  |   ✅   |
| SQL Server |   ✅   |   Dameng   |   ✅   |  Gauss   |   ✅   |
|   SQLite   |   ✅   | ClickHouse |   ✅   |          |        |

#### NoSQL Databases / Middleware

|   Database    | Status | Database | Status |
| :-----------: | :----: | :------: | :----: |
|    MongoDB    |   ✅   |  Redis   |   ✅   |
| Elasticsearch |   ✅   |  Milvus  |   ✅   |
|    Kafka      |   ✅   |  Docker  |   ✅   |

---

## 🛠️ Tech Stack

| Layer | Technology |
| :--- | :--- |
| Frontend | TypeScript, Vue 3 (Composition API), Vite, Element Plus, Tailwind CSS, Pinia |
| Backend | Go 1.26+, Gin, GORM, Clean Architecture layering |
| Remote Desktop | Apache Guacamole (RDP) |
| Terminal Recording | Asciinema replay |
| Deployment | Single binary with embedded frontend static assets, Docker / Docker Compose |

---

## 🚀 Quick Start

### Option 1: Docker (recommended)

```bash
# Pull and run (default port 18888, open http://ip:18888 in a browser)
docker run -d -p 18888:18888 --name mayfly-go docker.io/mayflygo/mayfly-go:latest
```

### Option 2: Docker Compose

```bash
git clone https://github.com/dromara/mayfly-go.git
cd mayfly-go
docker-compose up -d
```

### Option 3: Run from source

```bash
# Backend (reads server/config.yml by default, port 18888)
cd server && go run main.go

# Frontend (development)
cd frontend && pnpm install && pnpm dev
```

> 📚 For detailed deployment steps — database initialization, config items, reverse proxy, etc. — see the [Documentation](https://www.yuque.com/may-fly/mayfly-go).

---

## 🎮 Live Demo

| URL | Account / Password |
| :--- | :--- |
| http://go.mayfly.run | `test` / `test123.` |

---

## 🖼️ Screenshots

> Screenshots are archived by feature module under `docs/images/<module>/`, all captured in light mode.

### 🏠 Home Dashboard · `home/`

<div align="center"><img src="./docs/images/home/overview.jpg" alt="Home Dashboard" width="820"/></div>

### 🌳 Resource & Permission · `resource/`

<table>
  <tr>
    <td align="center" width="50%"><b>Unified Resource Tree</b><br/><img src="./docs/images/resource/resource-tree.jpg" alt="Resource Tree" width="100%"/></td>
    <td align="center" width="50%"><b>My Resources</b><br/><img src="./docs/images/resource/my-resource.jpg" alt="My Resources" width="100%"/></td>
  </tr>
</table>

### 🖥️ Machine Ops · `machine/`

<table>
  <tr>
    <td align="center" width="33%"><b>Web Terminal (live SSH)</b><br/><img src="./docs/images/machine/terminal.jpg" alt="Terminal" width="100%"/></td>
    <td align="center" width="33%"><b>File Manager</b><br/><img src="./docs/images/machine/file-manager.jpg" alt="File Manager" width="100%"/></td>
    <td align="center" width="33%"><b>Command Security Interception</b><br/><img src="./docs/images/machine/cmd-security.jpg" alt="Command Interception" width="100%"/></td>
  </tr>
</table>

### 🗄️ Database / Cache / Document / Container · `db/` `redis/` `mongo/` `docker/`

<table>
  <tr>
    <td align="center" width="50%"><b>SQL Editor (Monaco)</b><br/><img src="./docs/images/db/sql-editor.jpg" alt="SQL Editor" width="100%"/></td>
    <td align="center" width="50%"><b>Table Data · Visual Edit</b><br/><img src="./docs/images/db/table-data.jpg" alt="Table Data" width="100%"/></td>
  </tr>
  <tr>
    <td align="center"><b>Data Sync</b><br/><img src="./docs/images/db/sync.jpg" alt="Data Sync" width="100%"/></td>
    <td align="center"><b>Data Migration</b><br/><img src="./docs/images/db/transfer.jpg" alt="Data Migration" width="100%"/></td>
  </tr>
  <tr>
    <td align="center"><b>Data Masking</b><br/><img src="./docs/images/db/mask.jpg" alt="Data Masking" width="100%"/></td>
    <td align="center"><b>Redis Key Ops</b><br/><img src="./docs/images/redis/key-value.jpg" alt="Redis" width="100%"/></td>
  </tr>
  <tr>
    <td align="center" colspan="2"><b>MongoDB Collection Documents</b><br/><img src="./docs/images/mongo/data.jpg" alt="MongoDB" width="60%"/></td>
  </tr>
  <tr>
    <td align="center" colspan="2"><b>Container Management (status / resources / ports / logs)</b><br/><img src="./docs/images/docker/containers.jpg" alt="Containers" width="60%"/></td>
  </tr>
</table>

### 🚨 Alert Center · `alert/`　📋 Work-order Flow · `flow/`

<table>
  <tr>
    <td align="center" width="50%"><b>Alert Overview</b><br/><img src="./docs/images/alert/overview.jpg" alt="Alert Overview" width="100%"/></td>
    <td align="center" width="50%"><b>Alert Rules</b><br/><img src="./docs/images/alert/rules.jpg" alt="Alert Rules" width="100%"/></td>
  </tr>
  <tr>
    <td align="center"><b>My Pending Tasks</b><br/><img src="./docs/images/flow/tasks.jpg" alt="Flow Tasks" width="100%"/></td>
    <td align="center"><b>Process Definition</b><br/><img src="./docs/images/flow/procdef.jpg" alt="Process Definition" width="100%"/></td>
  </tr>
</table>

### 🤖 AI Assistant & 🧠 AgentOps · `ai/` `agentops/`　📬 Notifications · `msg/`

<table>
  <tr>
    <td align="center" width="50%"><b>AI Assistant (tool calls / report)</b><br/><img src="./docs/images/ai/assistant.jpg" alt="AI Assistant" width="100%"/></td>
    <td align="center" width="50%"><b>Governance (kill switch / KPI / runs)</b><br/><img src="./docs/images/agentops/governance.jpg" alt="Governance" width="100%"/></td>
  </tr>
  <tr>
    <td align="center" colspan="2"><b>Channels (WeCom / DingTalk / Feishu / Email)</b><br/><img src="./docs/images/msg/channels.jpg" alt="Channels" width="60%"/></td>
  </tr>
</table>

### ⚙️ System Management · `system/`

<table>
  <tr>
    <td align="center" width="33%"><b>Accounts</b><br/><img src="./docs/images/system/account.jpg" alt="Accounts" width="100%"/></td>
    <td align="center" width="33%"><b>Roles</b><br/><img src="./docs/images/system/role.jpg" alt="Roles" width="100%"/></td>
    <td align="center" width="33%"><b>Menu Permissions</b><br/><img src="./docs/images/system/menu.jpg" alt="Menu Permissions" width="100%"/></td>
  </tr>
</table>

> **MongoDB / Elasticsearch / container / Kafka operation views are also entered from the unified resource tree; more features & guides are available in the project documentation.**

---

## 💬 Community & Feedback

Join the QQ group: [119699946](https://qm.qq.com/cgi-bin/qm/qr?k=IdJSHW0jTMhmWFHBUS9a83wxtrxDDhFj&jump_from=webapi)

---

## 🌟 Star History

If the project has been helpful to you, please consider giving it a ⭐ Star on <a href="https://github.com/dromara/mayfly-go" target="_blank">Github</a>, <a href="https://gitee.com/dromara/mayfly-go" target="_blank">Gitee</a> or <a href="https://gitcode.com/dromara/mayfly-go" target="_blank">Gitcode</a> — it's the greatest encouragement and support!

<div align="center">

> Buy the author a coffee ☕️ or a bubble tea 🧋 to keep the code flowing!

<img class="no-margin" src="https://foruda.gitee.com/images/1744113367791412282/36a3c23b_1240250.png" alt="WeChat tip" width="200" height="200">

**Special thanks:** Sponsorships of ¥199 or more can add the author's WeChat (wx-error) to join the paid community for faster, priority technical support.

</div>
