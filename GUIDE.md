# gitf 操作指南

> 一句话：gitf 是一个"自己家的小型 GitHub"——单文件部署的私有 Git 托管与协作平台。
> 本指南面向实际部署和日常使用，所有步骤均基于当前已实现并测试通过的功能。
> 架构与开发信息见 [ARCHITECTURE.md](ARCHITECTURE.md)，测试情况见 [TESTING.md](TESTING.md)。

---

## 目录

1. [它能做什么 / 不能做什么](#1-它能做什么--不能做什么)
2. [安装部署](#2-安装部署)
3. [首次启动配置](#3-首次启动配置)
4. [账号与登录](#4-账号与登录)
5. [仓库：创建与代码推送](#5-仓库创建与代码推送)
6. [协作流程：Issue → 分支 → PR → 合并](#6-协作流程issue--分支--pr--合并)
7. [组织与团队](#7-组织与团队)
8. [发布版本（Release）](#8-发布版本release)
9. [Webhook](#9-webhook)
10. [管理员操作](#10-管理员操作)
11. [日常运维：备份 / 升级 / 监控](#11-日常运维)
12. [常见问题 FAQ](#12-常见问题-faq)

---

## 1. 它能做什么 / 不能做什么

**能做（实测可用）：**

- 私有 Git 托管：建仓、`clone / push / pull`、公开/私有可见性、协作者权限、分支保护
- 代码浏览：文件树、提交历史、分支对比、Raw 下载、归档打包下载
- 协作：Issue（评论/标签/里程碑/看板）、Pull Request（diff、评审、三种方式合并）
- 组织与团队、Release 发布（带附件）、push 事件 Webhook
- 管理后台：用户管理、站点统计

**不能做（请勿依赖）：**

- ❌ CI/CD 流水线 —— Actions 只是空壳，跑不了构建测试（需要请外挂 Jenkins 等）
- ❌ 私有包仓库 —— npm/PyPI 等端点只是协议占位
- ❌ 企业级账号体系 —— 没有 LDAP/SSO、两步验证、邮箱验证
- ❌ SSH 协议 —— Git 操作只支持 HTTP(S)

## 2. 安装部署

### 前置条件

- 一台 Linux 或 Windows 机器（1 核 1G 起步完全够用）
- 服务器上必须装有 `git` 命令（push/pull 靠它干活）
- 只有一种情况需要 Go 1.25+ 和 Node 22+：从源码自己编译

### 方式 A：直接使用编译好的二进制（推荐）

```bash
# 1. 上传 bin/gitf（Linux）或 bin/gitf.exe（Windows）到服务器，例如 /opt/gitf
# 2. 建数据目录和配置文件
mkdir -p /opt/gitf/data
cd /opt/gitf
```

创建 `.env`（和二进制放同一目录）：

```ini
ADDR=:8080
DATA_DIR=/opt/gitf/data
DB_DRIVER=sqlite
DB_DSN=/opt/gitf/data/app.db

# ↓↓↓ 三个生产必填项 ↓↓↓
# 固定密钥（随便一串长随机字符；不设的话每次重启所有人都要重新登录）
JWT_SECRET=换成你的长随机字符串
# 首次启动（空库）必填：引导管理员账号
DEFAULT_ADMIN_USERNAME=admin
DEFAULT_ADMIN_PASSWORD=管理员初始密码
# 启用分支保护强制执行（不设也能用，只是保护规则不生效）
HOOK_SECRET=再换一串随机字符串
```

```bash
# 3. 启动
./gitf
# => 浏览器访问 http://服务器IP:8080
```

### 方式 B：源码构建

```bash
git clone <repo-url> && cd gitf
make build        # 前端构建 + 嵌入 + Go 编译，产出 bin/gitf
make run
```

### 方式 C：systemd 常驻（生产推荐）

```bash
cp deploy/gitf.service /etc/systemd/system/gitf.service
systemctl daemon-reload
systemctl enable --now gitf
```

### 方式 D：HTTPS 对外（可选）

gitf 自带 TLS 配置（`TLS_ENABLED` / `TLS_CERT_FILE` / `TLS_KEY_FILE`），
更常见的做法是前置 nginx 做 HTTPS 和 gzip 卸载，参考 `deploy/nginx-gitf.conf`。

## 3. 首次启动配置

按上面配置启动后：

1. 用 `DEFAULT_ADMIN_USERNAME` / `DEFAULT_ADMIN_PASSWORD` 直接登录 —— 这就是站点管理员。
2. **立即改掉初始密码**（管理员后台或个人设置）。
3. 检查注册策略：默认任何人可自助注册。内网随意；公网建议在 `.env` 加
   `DISABLE_REGISTRATION=true`，由管理员手动建账号。

## 4. 账号与登录

| 场景 | 用什么 |
|------|--------|
| 网页登录 | 用户名 + 密码（浏览器会保存会话） |
| 网页/API 调用 | `Authorization: Bearer <JWT或PAT>` |
| `git clone/push` | 用户名 + **账号密码**（HTTP Basic） |

**创建个人访问令牌（PAT）**：网页右上角用户菜单 → PAT 令牌 → 新建。
令牌只在创建时显示一次，适合脚本调用 API（`gitf_` 开头），可随时吊销。

## 5. 仓库：创建与代码推送

网页上：**仓库 → 新建仓库**，填名字、选 Public/Private、可勾选"添加 README"。

然后像用 GitHub 一样推代码：

```bash
git clone http://服务器IP:8080/<用户名>/<仓库名>.git
cd <仓库名>
git add . && git commit -m "init" && git push origin main
```

实用功能速查：

| 想做什么 | 在哪做 |
|----------|--------|
| 加协作者 / 改权限 | 仓库 → 设置 → Collaborators |
| 保护分支 | 暂无页面，用 API（见下） |
| 对比两个分支 | `仓库页面 → Compare`，或访问 `/api/v1/repos/.../compare/base...head` |
| 下载源码包 | 仓库页面 → Tags/Release 旁的 archive，或 `/api/v1/repos/.../archive/main` |
| Fork 到自己名下 | 仓库页面 → Fork |
| 改名 / 改描述 / 删库 | 仓库 → 设置 |
| 管理访问令牌 | 右上角用户菜单 → PAT 令牌 |

> 注意：删除仓库是**物理删除**（数据库记录 + 磁盘上的 bare 仓库一起没），
> 不可恢复，删前想清楚。

**保护分支（API 操作，暂无界面）** —— 前提：服务器 `.env` 已配置 `HOOK_SECRET`：

```bash
# 保护 main：禁止删除、禁止强推、PR 合并需 1 个批准
curl -X POST http://服务器IP:8080/api/v1/repos/<所有者>/<仓库>/protection \
  -H "Authorization: Bearer <你的令牌>" -H "Content-Type: application/json" \
  -d '{"pattern":"main","allow_delete":false,"allow_force_push":false,"require_approvals":1}'

# 查看已有规则 / 删除规则
curl http://服务器IP:8080/api/v1/repos/<所有者>/<仓库>/protection -H "Authorization: Bearer <令牌>"
curl -X DELETE http://服务器IP:8080/api/v1/repos/<所有者>/<仓库>/protection/<规则ID> -H "Authorization: Bearer <令牌>"
```

生效后：直接 push 删被保护分支、强制覆盖历史都会被拒绝，
`require_approvals > 0` 时 PR 合并前必须先拿到对应数量的 approving 评审。

## 6. 协作流程：Issue → 分支 → PR → 合并

标准工作流和 GitHub 一致：

```bash
# 1. 记个 Issue（网页上：仓库 → Issues → 新建）
# 2. 开分支干活
git checkout -b feat/xxx && git push -u origin feat/xxx
# 3. 网页上：仓库 → PR → 新建，选择 base(main) ← head(feat/xxx)
```

**PR 页面能做的事：**

- 看 diff（改动内容逐文件展示）
- 评审：`approved`（批准）/ `changes_requested`（打回）/ 普通评论
- 合并，三种方式任选：
  - `merge` —— 产生一个合并提交（默认，保留完整历史）
  - `squash` —— 压成一个提交合入（历史干净）
  - `rebase` —— 把分支提交逐个搬到 main 上（线性历史）
- 合并后可关闭分支；保护分支若设置了"需要 N 个批准"，评审不够数会拒绝合并

配合分支保护可以实现"代码必须过评审才能进 main"的规矩。

## 7. 组织与团队

网页上：**组织 → 新建组织**。

- **成员**：邀请用户加入，角色分 `owner / admin / member`
- **团队**：组织内分组（如"后端组"），可批量管理成员
- 权限边界：只有组织 owner/admin 能管理成员和团队；普通成员不行
  （这一条经过安全测试验证，不再存在越权）

> 当前限制：REST API / 网页**不能直接创建"组织名下"的仓库**，
> 团队绑定仓库功能只对已存在的组织仓库有效。个人仓库归个人管，够用。

## 8. 发布版本（Release）

```bash
# 1. 打 tag 并推送
git tag v1.0.0 && git push origin v1.0.0
# 2. 网页上：仓库 → 发布 → 基于该 tag 创建 Release，写更新说明
# 3. 上传编译产物（如二进制、安装包）作为附件，别人可直接下载
```

tag 名在同一仓库内唯一，不同仓库互不影响。

## 9. Webhook

仓库 → 设置 → Hooks：填一个接收地址、密钥、事件类型。

**当前只支持 `push` 事件**：有人推代码 → gitf 会向你的地址 POST 一条 JSON
（包含 `ref`、`branch`、`before`、`after`、`repository`）。用途举例：
触发外部的部署脚本、通知群机器人、同步镜像。

- 每次投递都有记录（投递历史可查看、可手动重发）
- 投递失败不影响 push 本身
- `pull_request` 等其他事件类型可以配置但不会触发

## 10. 管理员操作

站点管理员登录后：

- **站点统计**：`GET /api/v1/admin/stats`（用户数、仓库数等）
- **用户管理**：`/api/v1/admin/users` —— 查询/创建/修改/删除任意用户、
  授予或收回站点管理员权限（`is_admin` 字段）

把某个已有用户提升为管理员：管理员通过 admin API 修改，或直接改数据库：

```sql
UPDATE users SET is_site_admin = 1 WHERE username = '某人';
```

## 11. 日常运维

**备份**（整个 gitf 的家当就是 data 目录）：

```bash
# 建议停服或低峰期执行；SQLite 支持在线拷贝但有短暂一致性风险
tar czf gitf-backup-$(date +%F).tar.gz /opt/gitf/data
# 恢复 = 解压回原位置 + 重启
```

**升级**：换新二进制 → 重启。数据库结构变更自动迁移，无需手动操作。
（升级前备份 data 目录，永远是对的。）

**健康检查**（接负载均衡/监控用）：

| 端点 | 含义 |
|------|------|
| `/health` | 进程活着 |
| `/health/ready` | 数据库可连（就绪） |
| `/metrics` | JSON 格式运行指标（请求计数/耗时），建议用 nginx 限制为内网访问 |

**日志**：JSON 格式输出到 stdout，每行带 `request_id`，交给 systemd/journald 即可。

**限流**（防滥用，均为每 IP 每分钟）：注册 5 次、登录 10 次、API 1000 次、
Git 操作 120 次。放反代后记得让 nginx 传真实 IP（`X-Forwarded-For` 相关配置），
否则所有人共享 nginx 的 IP 限额。

## 12. 常见问题 FAQ

**Q：重启服务后所有人都被登出了？**
没设 `JWT_SECRET`（或每次设了不同值），重启后旧令牌全部作废。设一个固定的即可。

**Q：git push 被拒绝 "branch is protected"？**
分支保护规则在起作用——这正是配了 `HOOK_SECRET` 后的预期行为。
要么走 PR 评审合并，要么让管理员调整保护规则。

**Q：管理员密码忘了？**
```sql
-- 最直接的办法（密码哈希用 bcrypt，无法反查，只能改库重置或重建账号）
-- 1. 让其他管理员通过 admin API 重置；或
-- 2. 停服后用任意 bcrypt 哈希生成工具生成新哈希：
UPDATE users SET password_hash='<新生成的bcrypt哈希>' WHERE username='admin';
```

**Q：能在公网直接裸奔吗？**
技术上可以（有 HTTPS、限流、严格 CSP），但没有邮箱验证和防垃圾注册，
公网部署强烈建议：`DISABLE_REGISTRATION=true` + 前置 nginx 上 HTTPS +
把 `/metrics` 限制为内网。

**Q：数据都在哪？**
全部在 `DATA_DIR` 一个目录里：`app.db`（用户/Issue/PR 等结构化数据）+
`repositories/`（各仓库的 bare 仓库文件）。换机器 = 拷走这个目录。

**Q：和 Gitea 怎么选？**
要完整 DevOps（CI/CD、容器仓库、组织级安全策略）→ Gitea/GitLab；
要"小团队私有代码托管 + 轻协作"、追求极简单文件部署和低维护 → gitf。
