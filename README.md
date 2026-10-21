# gitf

A lightweight self-hosted Git service built with **Go** and **Vue 3** — inspired by Gitea, but much simpler.

[English](#) | [中文](#gitf-中文)

## Who it is for

**A private code host + lightweight collaboration platform for small teams.** One
24 MB binary, no external database server, backups are one directory.

Good fit:
- Teams of a few to a few dozen people who want their source code on **their own
  server** (intranet, compliance, cost)
- Individual developers who want a private Git home on a cheap VPS or NAS
- Chinese-speaking small teams (Chinese-first UI and docs)
- Learning / secondary development: the codebase is small enough to read in full

Not a good fit:
- Teams that need **CI/CD built in** — the Actions subsystem is a shell; wire up
  Jenkins or similar externally
- Large organizations needing SSO/LDAP, 2FA, email verification or audit features
- Public community hosting (weak anti-spam for open registration)
- Private package registries (those endpoints are stubs)

## Features

- User registration / login (JWT) and personal access tokens (PAT)
- Create, fork and delete Git repositories (public / private)
- `git clone / push / pull` over HTTP, with collaborator roles and branch protection
- Browse repositories: branches, tags, commits, file tree, blob view, raw download, compare, archive
- Issues with comments, labels, milestones and a kanban board
- Pull requests: diff, reviews, merge (merge / squash / rebase), branch protection enforcement
- Organizations: members, teams, team repositories
- Releases with binary asset attachments
- Webhooks with push-event dispatch and delivery history
- Admin: site statistics and user management
- CI (Actions) runner API and package-registry endpoints (experimental)
- Single-binary deployment with embedded frontend assets, gzip compression built in

## Tech Stack

| Layer          | Technology            |
|----------------|-----------------------|
| Backend        | Go 1.25 + chi + GORM + go-git |
| Frontend       | Vue 3 + Element Plus + Vite |
| Database       | SQLite (default) / PostgreSQL |
| Deployment     | Single binary (systemd) |

## Quick Start

```bash
# One-command build (requires Go 1.25+ and Node 22+).
make build

# Run.
make run
# => http://localhost:8080
```

> **Important:** Set `JWT_SECRET` in your environment or `.env` file. If left
> empty, the server auto-generates a random secret on every start, which
> invalidates all user sessions after each restart.

## Development

```bash
# Backend
go run ./cmd/server

# Frontend (hot reload, proxies /api to :8080)
cd web && npm run dev
# => http://localhost:4200

# Unit tests
make test

# API end-to-end tests (against a running server)
node scripts/api_test.mjs
```

## Environment Variables

| Variable     | Default  | Description                       |
|-------------|----------|-----------------------------------|
| `ADDR`      | `:8080`  | Listen address                     |
| `DATA_DIR`  | `./data` | Data directory for repos and DB    |
| `DB_DRIVER` | `sqlite` | Database driver (`sqlite` or `postgres`) |
| `DB_DSN`    | `./data/app.db` | Database connection string |
| `JWT_SECRET`| *(auto-generated)* | JWT signing secret — set a fixed value for production |
| `HEALTH_LEAK_DETAILS` | `false` | Include DB error details in `/health/ready` responses |
| `ENABLE_PPROF` | `false` | Expose `/debug/pprof/` endpoints (do NOT enable in production) |

## Project Layout

```
gitf/
├── cmd/server/        # Go entry point
├── internal/          # Go internal packages
├── web/               # Vue 3 frontend source
├── Dockerfile
├── Makefile
├── ARCHITECTURE.md    # How the pieces fit together
├── CONTRIBUTING.md    # Contributor guide
└── README.md
```

## Documentation

- [`GUIDE.md`](GUIDE.md) — **operations guide**: deploy, configure, daily use, admin, backup, FAQ (中文).
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — design decisions, data flow, conventions.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — getting started, testing, code style, PR process.
- [`TESTING.md`](TESTING.md) — test strategy, end-to-end suite, quality audit record.
- [`REVIEW_CHECKLIST.md`](REVIEW_CHECKLIST.md) — pre-merge review checklist.

## License

MIT. See [LICENSE](LICENSE).

---

# gitf (中文)

基于 **Go** + **Vue 3** 构建的轻量级自托管 Git 服务，灵感来自 Gitea，但更加精简。

## 定位：适合谁

**一个面向小团队的"私有代码仓库 + 轻量协作平台"。** 24MB 单文件部署，不依赖外部数据库，备份就是拷一个目录。

**适合：**

- 几人到几十人的团队，想把源码放在**自己的服务器**上（内网、合规、成本考虑）
- 个人开发者，想在一台便宜 VPS 或 NAS 上有个私有代码归置地
- 中文小团队（界面与文档对中文友好）
- 学习与二次开发：代码量不大、结构清晰，可以完整读一遍

**不适合：**

- 指望内置 **CI/CD** 的团队 —— Actions 子系统是空壳，需要外挂 Jenkins 等工具
- 需要 SSO/LDAP、两步验证、邮箱验证等企业级账号体系的大组织
- 公网开源社区托管 —— 开放注册缺乏防垃圾能力
- 需要私有包仓库的团队 —— 相关端点只是占位

一句话：要完整 DevOps 平台请上 Gitea/GitLab；要"极简部署、低维护的私有代码托管 + 轻协作"，gitf 就是干这个的。

## 功能

- 用户注册 / 登录 (JWT) 与个人访问令牌 (PAT)
- 创建、Fork、删除 Git 仓库（公开 / 私有）
- 通过 HTTP 进行 `git clone / push / pull`，支持协作者角色与分支保护
- 浏览仓库：分支、标签、提交历史、文件树、Blob 查看、Raw 下载、分支对比、归档下载
- Issue（评论、标签、里程碑）与看板
- Pull Request：diff、代码评审、合并（merge / squash / rebase）、分支保护强制执行
- 组织：成员、团队、团队仓库
- Release 发布（支持二进制附件）
- Webhook（push 事件自动投递，含投递历史）
- 管理后台：站点统计与用户管理
- CI (Actions) Runner API 与包仓库端点（实验性）
- 单二进制部署，前端资源嵌入，内置 gzip 压缩

## 技术栈

| 层级     | 技术                    |
|----------|------------------------|
| 后端     | Go 1.25 + chi + GORM + go-git |
| 前端     | Vue 3 + Element Plus + Vite |
| 数据库   | SQLite（默认）/ PostgreSQL |
| 部署     | 单二进制（systemd）      |

## 快速启动

```bash
# 一键构建（需要 Go 1.25+ 和 Node 22+）
make build

# 运行
make run
# => http://localhost:8080
```

生产部署用单二进制 + systemd（见下方[生产部署](#生产部署)或
[`GUIDE.md`](GUIDE.md)），完整操作说明见 [`GUIDE.md`](GUIDE.md)。

> **重要：** 请在环境变量或 `.env` 文件中设置 `JWT_SECRET`。如果留空，
> 服务器每次启动都会自动生成随机密钥，这将导致每次重启后所有用户会话失效。

## 开发

```bash
# 后端
go run ./cmd/server

# 前端（热更新，代理 /api 到 8080）
cd web && npm run dev
# => http://localhost:4200

# 单元测试
make test

# API 端到端测试（需先启动服务）
node scripts/api_test.mjs
```

## 生产部署

```bash
# 1. 在本地或服务器编译单二进制
make build

# 2. 上传到服务器，例如 /opt/gitf
# 3. 创建数据目录和 .env
mkdir -p /opt/gitf/data
cat > /opt/gitf/.env <<'EOF'
ADDR=:8080
DATA_DIR=/opt/gitf/data
DB_DRIVER=sqlite
DB_DSN=/opt/gitf/data/app.db
JWT_SECRET=<your-fixed-secret>
LOG_LEVEL=info
LOG_FORMAT=json
EOF

# 4. 安装 systemd 服务
cp deploy/gitf.service /etc/systemd/system/gitf.service
systemctl daemon-reload
systemctl enable gitf
systemctl start gitf
```

## 环境变量

| 变量          | 默认值    | 说明                  |
|--------------|-----------|----------------------|
| `ADDR`       | `:8080`   | 监听地址               |
| `DATA_DIR`   | `./data`  | 仓库及数据库数据目录      |
| `DB_DRIVER`  | `sqlite`  | 数据库驱动 (`sqlite` 或 `postgres`) |
| `DB_DSN`     | `./data/app.db` | 数据库连接串     |
| `JWT_SECRET` | *(自动生成)* | JWT 签名密钥 — 生产环境请设置固定值 |
| `HEALTH_LEAK_DETAILS` | `false` | `/health/ready` 是否返回 DB 错误详情 |
| `ENABLE_PPROF` | `false` | 是否暴露 `/debug/pprof/`（生产环境勿启用） |

## 项目结构

```
gitf/
├── cmd/server/        # Go 入口
├── internal/          # Go 内部包
├── web/               # Vue 3 前端源码
├── Dockerfile
├── Makefile
├── ARCHITECTURE.md    # 架构说明
├── CONTRIBUTING.md    # 贡献者指南
└── README.md
```

## 文档

- [`GUIDE.md`](GUIDE.md) — **操作指南**：部署、配置、日常使用、管理员操作、备份、FAQ。
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — 架构说明。
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — 贡献者指南。
- [`TESTING.md`](TESTING.md) — 测试策略、端到端测试与质量审计记录。
- [`REVIEW_CHECKLIST.md`](REVIEW_CHECKLIST.md) — 合并前审查清单。

## 协议

MIT。详见 [LICENSE](LICENSE)。
