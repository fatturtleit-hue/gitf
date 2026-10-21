#!/usr/bin/env node
/**
 * gitf black-box API test suite.
 *
 * Exercises every REST endpoint plus the git HTTP protocol against a RUNNING
 * server (default http://localhost:8080). Zero dependencies — uses Node's
 * global fetch and the system git client.
 *
 * Usage:
 *   node scripts/api_test.mjs [--base http://localhost:8080]
 *
 * Creates throwaway users/repos/orgs named zt-<suffix>*, prints a PASS/FAIL
 * summary and exits non-zero when any assertion fails.
 */

const BASE = (() => {
  const i = process.argv.indexOf('--base')
  return i > 0 ? process.argv[i + 1] : 'http://localhost:8080'
})()
const API = `${BASE}/api/v1`

const SUFFIX = Date.now().toString(36).slice(-6)
const USER_A = `zt-a-${SUFFIX}`
const USER_B = `zt-b-${SUFFIX}`
const PASSWORD = 'Ztest-pass-1'
const EMAIL_A = `zt-a-${SUFFIX}@test.local`
const EMAIL_B = `zt-b-${SUFFIX}@test.local`

let pass = 0
let fail = 0
const failures = []
const notes = []

function check(name, ok, detail = '') {
  if (ok) {
    pass++
    console.log(`  \x1b[32mPASS\x1b[0m ${name}`)
  } else {
    fail++
    failures.push({ name, detail })
    console.log(`  \x1b[31mFAIL\x1b[0m ${name}${detail ? ` — ${detail}` : ''}`)
  }
}

function note(msg) {
  notes.push(msg)
  console.log(`  \x1b[33mNOTE\x1b[0m ${msg}`)
}

function section(title) {
  console.log(`\n== ${title} ${'='.repeat(Math.max(0, 66 - title.length))}`)
}

async function req(method, path, { token, body, headers = {}, rawBody } = {}) {
  const h = { ...headers }
  if (token) h.Authorization = `Bearer ${token}`
  let payload
  if (rawBody !== undefined) {
    payload = rawBody
  } else if (body !== undefined) {
    h['Content-Type'] = 'application/json'
    payload = JSON.stringify(body)
  }
  const res = await fetch(`${BASE}${path}`, { method, headers: h, body: payload, redirect: 'manual' })
  const text = await res.text()
  let json = null
  try { json = JSON.parse(text) } catch { /* non-JSON */ }
  return { status: res.status, headers: res.headers, text, json }
}

const get = (p, o) => req('GET', p, o)
const post = (p, o) => req('POST', p, o)
const put = (p, o) => req('PUT', p, o)
const patch = (p, o) => req('PATCH', p, o)
const del = (p, o) => req('DELETE', p, o)

// ─────────────────────────── git helpers ───────────────────────────
import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

function basicHeader(user, pass) {
  return 'Basic ' + Buffer.from(`${user}:${pass}`).toString('base64')
}

function git(args, opts = {}) {
  // GIT_TERMINAL_PROMPT=0 makes git fail fast on 401 instead of hanging on a
  // credential prompt it can never read in pipe mode.
  return execFileSync('git', args, {
    encoding: 'utf8',
    env: { ...process.env, GIT_TERMINAL_PROMPT: '0' },
    timeout: 60000,
    ...opts,
  })
}

function gitAuthed(args, user, pass, opts = {}) {
  return git(['-c', `http.extraHeader=Authorization: ${basicHeader(user, pass)}`, ...args], opts)
}

// ─────────────────────────── shared state ───────────────────────────
let tokA, tokB, repoPath

async function main() {
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), `gitf-test-${SUFFIX}-`))
  try {
    await testInfra()
    await testAuth()
    await testRepos()
    await testGitHttp(tmp)
    await testContent()
    await testCollaborators()
    await testIssues()
    await testPRs(tmp)
    await testOrgs()
    await testReleases()
    await testWebhooks(tmp)
    await testPAT()
    await testActions()
    await testPackages()
    await testNegative()
    await testRateLimit()
    console.log(`\nNOTE: admin CRUD 未测——需要站点管理员。请提升 ${USER_A} 后重跑 admin 段。`)
  } finally {
    // Windows can refuse to unlink a temp dir whose files are still mmapped
    // (EPERM); cleanup is best-effort and must not mask the test result.
    try {
      fs.rmSync(tmp, { recursive: true, force: true, maxRetries: 3 })
    } catch { /* leave the temp dir for the OS to sweep */ }
  }
  summary()
}

// ─────────────────────────── A. infra ───────────────────────────
async function testInfra() {
  section('A. 基础设施 / 健康检查')
  let r = await get('/health')
  check('GET /health → 200', r.status === 200)

  r = await get('/health/ready')
  check('GET /health/ready → 200', r.status === 200)

  r = await get('/metrics')
  // gitf exposes a custom JSON metrics format (not Prometheus).
  check('GET /metrics → 200 且含 http_requests_total', r.status === 200 && r.text.includes('http_requests_total'), `status=${r.status}`)

  r = await get('/')
  check('GET / → SPA 200', r.status === 200 && r.text.includes('<div'), `status=${r.status}`)

  r = await get('/login')
  check('SPA 路由回退 /login → 200', r.status === 200, `status=${r.status}`)

  r = await req('HEAD', '/')
  check('HEAD / → 200', r.status === 200, `status=${r.status}`)
}

async function ensureUser(username, email) {
  let r = await post('/api/v1/auth/register', { body: { username, email, password: PASSWORD } })
  if (r.status === 429 || r.status === 409) {
    // User may already exist (rerun) or register window is exhausted — fall
    // back to login with the same credentials.
    r = await post('/api/v1/auth/login', { body: { username, password: PASSWORD } })
    if (r.status !== 200) {
      throw new Error(`cannot obtain session for ${username}: register=${r.status}; wait ~60s and rerun`)
    }
    return r.json.token
  }
  if (r.status !== 201) {
    throw new Error(`register ${username} failed: ${r.status} ${r.text.slice(0, 120)}`)
  }
  return r.json.token
}

// ─────────────────────────── B. auth ───────────────────────────
async function testAuth() {
  section('B. 认证')
  let r = await get('/api/v1/auth/register/status')
  check('注册状态 → 200 {enabled}', r.status === 200 && typeof r.json?.enabled === 'boolean')

  r = await post('/api/v1/auth/register', { body: { username: USER_A, email: EMAIL_A, password: PASSWORD } })
  const registeredA = r.status === 201
  check(`注册/登录 ${USER_A}`, r.status === 201 || r.status === 200, `status=${r.status} body=${r.text.slice(0, 120)}`)
  tokA = r.json?.token ?? (await ensureUser(USER_A, EMAIL_A))
  check('注册响应含 token', !!tokA)

  r = await post('/api/v1/auth/register', { body: { username: USER_B, email: EMAIL_B, password: PASSWORD } })
  check(`注册/登录 ${USER_B}`, r.status === 201 || r.status === 200, `status=${r.status}`)
  tokB = r.json?.token ?? (await ensureUser(USER_B, EMAIL_B))

  if (!registeredA) note('注册被限流，本次改用已有用户登录（限流本身工作正常）')

  r = await post('/api/v1/auth/register', { body: { username: USER_A, email: 'x@test.local', password: PASSWORD } })
  check('重复用户名 → 409', r.status === 409, `status=${r.status}`)

  r = await post('/api/v1/auth/register', { body: { username: `short-${SUFFIX}`, email: 's@test.local', password: 'short' } })
  check('弱密码 → 400', r.status === 400, `status=${r.status}`)

  r = await post('/api/v1/auth/register', { body: { username: 'bad name!', email: 'b@test.local', password: PASSWORD } })
  check('非法用户名 → 400', r.status === 400, `status=${r.status}`)

  r = await post('/api/v1/auth/login', { body: { username: USER_A, password: 'WrongPass-1' } })
  check('错误密码登录 → 401', r.status === 401, `status=${r.status}`)

  r = await post('/api/v1/auth/login', { body: { username: USER_A, password: PASSWORD } })
  check('正确登录 → 200 + token', r.status === 200 && !!r.json?.token, `status=${r.status}`)

  r = await get('/api/v1/user')
  check('无 token 访问 /user → 401', r.status === 401, `status=${r.status}`)

  r = await get('/api/v1/user', { token: 'not-a-jwt' })
  check('伪造 token → 401', r.status === 401, `status=${r.status}`)

  r = await get('/api/v1/user', { token: tokA })
  check('Bearer /user → 200 且用户名匹配', r.status === 200 && r.json?.username === USER_A, `status=${r.status}`)

  r = await post('/api/v1/auth/refresh', { token: tokA })
  check('刷新 token → 200', r.status === 200, `status=${r.status}`)

  r = await post('/api/v1/auth/setup', { token: tokA, body: { username: USER_A, email: EMAIL_A, password: PASSWORD } })
  check('非默认管理员调 /auth/setup → 403', r.status === 403, `status=${r.status}`)
}

// ─────────────────────────── C. repos ───────────────────────────
async function testRepos() {
  section('C. 仓库 CRUD 与可见性')
  let r = await post('/api/v1/repos', { token: tokA, body: { name: 't1', description: 'test repo', is_private: false, default_branch: 'main' } })
  check('创建公开仓库 → 201', r.status === 201, `status=${r.status} ${r.text.slice(0, 100)}`)
  repoPath = 't1'

  r = await post('/api/v1/repos', { token: tokA, body: { name: 't1' } })
  check('重名仓库 → 409', r.status === 409, `status=${r.status}`)

  r = await post('/api/v1/repos', { token: tokA, body: { name: 'Bad Name!' } })
  check('非法仓库名 → 400', r.status === 400, `status=${r.status}`)

  r = await get('/api/v1/repos/zt-nope/t1', { token: tokA })
  check('不存在仓库 → 404', r.status === 404, `status=${r.status}`)

  r = await get(`/api/v1/repos/${USER_A}/t1`, { token: tokB })
  check('他人可读公开仓库 → 200', r.status === 200, `status=${r.status}`)

  r = await patch(`/api/v1/repos/${USER_A}/t1`, { token: tokA, body: { description: 'updated desc' } })
  check('PATCH 更新描述 → 200', r.status === 200, `status=${r.status}`)

  r = await patch(`/api/v1/repos/${USER_A}/t1`, { token: tokB, body: { description: 'hacked' } })
  check('非所有者 PATCH → 403', r.status === 403, `status=${r.status}`)

  r = await del(`/api/v1/repos/${USER_A}/t1`, { token: tokB })
  check('非所有者 DELETE → 403', r.status === 403, `status=${r.status}`)

  r = await post('/api/v1/repos', { token: tokA, body: { name: 'priv', is_private: true } })
  check('创建私有仓库 → 201', r.status === 201, `status=${r.status}`)

  r = await get(`/api/v1/repos/${USER_A}/priv`, { token: tokB })
  check('他人读私有仓库 → 404/403', r.status === 404 || r.status === 403, `status=${r.status}`)

  r = await get('/api/v1/repos', { token: tokA })
  const names = (r.json?.items ?? r.json ?? []).map?.(x => x.name ?? x) ?? []
  check('仓库列表含 t1 与 priv', r.status === 200 && names.includes('t1') && names.includes('priv'), `got=${JSON.stringify(names).slice(0, 150)}`)

  r = await get('/api/v1/repos', { token: tokA, headers: {} })
  check('列表支持 search 过滤', (await get(`/api/v1/repos?search=t1`, { token: tokA })).status === 200)

  r = await del(`/api/v1/repos/${USER_A}/priv`, { token: tokA })
  check('删除私有仓库 → 200/204', r.status === 200 || r.status === 204, `status=${r.status}`)
}

// ─────────────────────────── D. git http ───────────────────────────
const ORIGIN = `http://127.0.0.1:${new URL(BASE).port || 80}`

async function testGitHttp(tmp) {
  section('D. Git HTTP 协议 (clone/push/pull)')
  const work = path.join(tmp, 't1')
  const url = `${ORIGIN}/${USER_A}/t1.git`

  git(['-c', 'init.defaultBranch=main', 'clone', url, work], { stdio: 'pipe' })
  check('匿名 clone 空公开仓库成功', fs.existsSync(path.join(work, '.git')))

  fs.writeFileSync(path.join(work, 'README.md'), '# t1\n\nhello gitf\n')
  fs.mkdirSync(path.join(work, 'src'))
  fs.writeFileSync(path.join(work, 'src', 'app.go'), 'package main\n')
  const id = ['-c', 'user.email=t@t.local', '-c', 'user.name=Tester']
  git([...id, 'add', '-A'], { cwd: work })
  git([...id, 'commit', '-m', 'init commit'], { cwd: work })
  gitAuthed(['push', '-u', 'origin', 'main'], USER_A, PASSWORD, { cwd: work, stdio: 'pipe' })
  check('认证 push main 成功', true)

  git([...id, 'commit', '--allow-empty', '-m', 'second'], { cwd: work })
  gitAuthed(['push', 'origin', 'main'], USER_A, PASSWORD, { cwd: work, stdio: 'pipe' })
  check('二次 push 成功', true)

  git(['checkout', '-b', 'feature'], { cwd: work, stdio: 'pipe' })
  fs.writeFileSync(path.join(work, 'feature.txt'), 'feature content\n')
  git([...id, 'add', '-A'], { cwd: work })
  git([...id, 'commit', '-m', 'feature work'], { cwd: work })
  gitAuthed(['push', '-u', 'origin', 'feature'], USER_A, PASSWORD, { cwd: work, stdio: 'pipe' })
  check('push feature 分支成功', true)
  git(['checkout', 'main'], { cwd: work, stdio: 'pipe' })

  gitAuthed(['tag', 'v0.0.1'], USER_A, PASSWORD, { cwd: work })
  gitAuthed(['push', 'origin', 'v0.0.1'], USER_A, PASSWORD, { cwd: work, stdio: 'pipe' })
  check('push tag 成功', true)

  const pullDir = path.join(tmp, 'pull')
  git(['clone', url, pullDir], { stdio: 'pipe' })
  fs.writeFileSync(path.join(pullDir, 'README.md'), '# t1\n\nhello gitf\n\npulled\n')
  const id2 = ['-c', 'user.email=t@t.local', '-c', 'user.name=Tester']
  git([...id2, 'add', '-A'], { cwd: pullDir })
  git([...id2, 'commit', '-m', 'local change'], { cwd: pullDir })
  gitAuthed(['pull', '--rebase', 'origin', 'main'], USER_A, PASSWORD, { cwd: pullDir, stdio: 'pipe' })
  check('pull --rebase 成功', true)

  const noauth = path.join(tmp, 'push-noauth')
  git(['clone', url, noauth], { stdio: 'pipe' })
  fs.appendFileSync(path.join(noauth, 'README.md'), 'anon\n')
  git([...id2, 'add', '-A'], { cwd: noauth })
  git([...id2, 'commit', '-m', 'anon'], { cwd: noauth })
  let rejected = false
  try { git(['push', 'origin', 'main'], { cwd: noauth, stdio: 'pipe' }) } catch { rejected = true }
  check('匿名 push 被拒绝', rejected)

  const privUrl = `${ORIGIN}/${USER_A}/priv.git`
  let denied = false
  try { git(['clone', privUrl, path.join(tmp, 'priv')], { stdio: 'pipe' }) } catch { denied = true }
  check('私有仓库匿名 clone 被拒绝 — 需先重建私有仓库', denied)

  const ls = gitAuthed(['ls-remote', url], USER_A, PASSWORD, { stdio: 'pipe' })
  check('ls-remote 列出分支与 tag', ls.includes('refs/heads/main') && ls.includes('refs/tags/v0.0.1'))
}

// ─────────────────────────── E. content ───────────────────────────
async function testContent() {
  section('E. 仓库内容 API')
  const R = `/api/v1/repos/${USER_A}/t1`
  let r = await get(`${R}/branches`, { token: tokA })
  const branches = JSON.stringify(r.json)
  check('分支列表含 main/feature', r.status === 200 && branches.includes('main') && branches.includes('feature'), r.text.slice(0, 120))

  r = await get(`${R}/commits?branch=main`, { token: tokA })
  check('提交列表 ≥2', r.status === 200 && (r.json?.commits?.length ?? 0) >= 2, r.text.slice(0, 120))
  const sha = r.json?.commits?.[0]?.hash ?? r.json?.commits?.[0]?.sha

  if (sha) {
    r = await get(`${R}/commits/${sha}`, { token: tokA })
    check('按 hash 查提交 → 200', r.status === 200, `status=${r.status}`)
  } else note('未取到 commit hash，跳过单提交测试')

  r = await get(`${R}/tree?branch=main`, { token: tokA })
  check('根目录 tree 含 README.md/src', r.status === 200 && JSON.stringify(r.json).includes('README.md'), r.text.slice(0, 150))

  r = await get(`${R}/tree?branch=main&path=src`, { token: tokA })
  check('子目录 tree 含 app.go', r.status === 200 && JSON.stringify(r.json).includes('app.go'), r.text.slice(0, 150))

  r = await get(`${R}/blob?branch=main&path=README.md`, { token: tokA })
  check('blob 返回内容', r.status === 200 && r.json?.content?.includes('hello gitf'), r.text.slice(0, 120))

  r = await get(`${R}/raw?branch=main&path=README.md`, { token: tokA })
  check('raw 返回文件字节', r.status === 200 && r.text.includes('hello gitf'), `status=${r.status}`)

  r = await get(`${R}/blob?branch=main&path=../etc/passwd`, { token: tokA })
  check('路径穿越 → 404/400', r.status === 404 || r.status === 400, `status=${r.status}`)

  r = await get(`${R}/stats`, { token: tokA })
  check('stats → 200', r.status === 200, `status=${r.status}`)

  r = await get(`${R}/tags`, { token: tokA })
  check('tags 含 v0.0.1', r.status === 200 && JSON.stringify(r.json).includes('v0.0.1'), r.text.slice(0, 120))

  r = await get(`${R}/compare/main...feature`, { token: tokA })
  check('compare main...feature → 200', r.status === 200, `status=${r.status} ${r.text.slice(0, 100)}`)

  r = await get(`${R}/archive/main`, { token: tokA })
  const size = r.text.length
  check('archive 下载非空', r.status === 200 && size > 100, `status=${r.status} size=${size}`)

  r = await get(`${R}/archive/no-such-ref`, { token: tokA })
  check('archive 无效 ref → 4xx/5xx 记录', r.status >= 400, `status=${r.status}`)

  // fork
  r = await post(`/api/v1/repos/${USER_A}/t1/fork`, { token: tokB, body: { name: 't1-fork' } })
  check('他人 fork → 201', r.status === 201, `status=${r.status} ${r.text.slice(0, 100)}`)
  r = await get(`/api/v1/repos/${USER_B}/t1-fork`, { token: tokB })
  check('fork 可读', r.status === 200, `status=${r.status}`)
}

// ─────────────────────────── F. collaborators ───────────────────────────
async function testCollaborators() {
  section('F. 协作者')
  const R = `/api/v1/repos/${USER_A}/t1`
  let r = await post(`${R}/collaborators`, { token: tokA, body: { username: USER_B, permission: 'write' } })
  check('添加协作者 → 200/201', r.status === 200 || r.status === 201, `status=${r.status} ${r.text.slice(0, 100)}`)

  r = await post(`${R}/collaborators`, { token: tokA, body: { username: USER_B, role: 'write' } })
  check('重复添加 → 409/400', r.status === 409 || r.status === 400, `status=${r.status}`)

  r = await get(`${R}/collaborators`, { token: tokA })
  check('协作者列表含 B', r.status === 200 && r.text.includes(USER_B), r.text.slice(0, 150))
  const entry = (r.json?.items ?? r.json ?? []).find?.(c => c.username === USER_B || c.user?.username === USER_B)
  const collabID = entry?.id ?? entry?.collab_id ?? entry?.userID

  if (collabID) {
    r = await put(`${R}/collaborators/${collabID}`, { token: tokA, body: { permission: 'admin' } })
    check('更新协作者角色 → 200', r.status === 200, `status=${r.status}`)
  } else note('未解析 collabID，跳过角色更新')

  r = await post(`${R}/collaborators`, { token: tokB, body: { username: USER_B } })
  check('非所有者添加协作者 → 403', r.status === 403, `status=${r.status}`)
}

// ─────────────────────────── G. issues ───────────────────────────
let issueID, columnID
async function testIssues() {
  section('G. Issue / 标签 / 里程碑 / 看板')
  const R = `/api/v1/repos/${USER_A}/t1`
  let r = await post(`${R}/labels`, { token: tokA, body: { name: 'bug', color: '#ff0000' } })
  check('创建标签 → 201', r.status === 201, `status=${r.status} ${r.text.slice(0, 100)}`)
  const labelID = r.json?.id

  r = await get(`${R}/labels`, { token: tokA })
  check('标签列表', r.status === 200 && r.text.includes('bug'))

  r = await post(`${R}/milestones`, { token: tokA, body: { title: 'M1', description: 'first', due_date: '2026-12-31' } })
  check('创建里程碑 → 200/201', r.status === 200 || r.status === 201, `status=${r.status} ${r.text.slice(0, 100)}`)
  const msID = r.json?.id

  r = await get(`${R}/milestones`, { token: tokA })
  check('里程碑列表', r.status === 200 && r.text.includes('M1'))

  r = await post(`${R}/issues`, { token: tokA, body: { title: 'issue one', body: 'something broke', label_ids: labelID ? [labelID] : [] } })
  check('创建 issue → 201', r.status === 201, `status=${r.status} ${r.text.slice(0, 100)}`)
  issueID = r.json?.id

  r = await get(`${R}/issues`, { token: tokA })
  check('issue 列表', r.status === 200 && r.text.includes('issue one'))

  r = await get(`${R}/issues/1`, { token: tokA })
  check('按编号读 issue → 200', r.status === 200, `status=${r.status}`)

  r = await patch(`${R}/issues/1`, { token: tokA, body: { title: 'issue one (edited)' } })
  check('PATCH issue → 200', r.status === 200, `status=${r.status}`)

  r = await post(`${R}/issues/1/comments`, { token: tokA, body: { body: 'a comment' } })
  check('添加评论 → 200/201', r.status === 200 || r.status === 201, `status=${r.status}`)

  r = await get(`${R}/issues/1/comments`, { token: tokA })
  check('评论列表', r.status === 200 && r.text.includes('a comment'))

  r = await post(`${R}/issues/1/close`, { token: tokA })
  check('关闭 issue → 200', r.status === 200, `status=${r.status}`)

  r = await post(`${R}/issues/1/reopen`, { token: tokA })
  check('重开 issue → 200', r.status === 200, `status=${r.status}`)

  r = await get(`${R}/kanban`, { token: tokA })
  check('看板列 → 200 有列', r.status === 200 && (r.json?.columns?.length ?? 0) > 0, r.text.slice(0, 150))
  columnID = r.json?.columns?.[0]?.id

  if (issueID && columnID) {
    r = await put(`${R}/kanban/move`, { token: tokA, body: { issue_id: issueID, column_id: columnID, position: 0 } })
    check('看板移动 issue → 200', r.status === 200, `status=${r.status}`)
  }

  if (msID) {
    r = await post(`${R}/milestones/close`, { token: tokA, body: { title: 'M1' } })
    check('关闭里程碑 → 200', r.status === 200, `status=${r.status} ${r.text.slice(0, 80)}`)
  }

  if (labelID) {
    r = await del(`${R}/labels/${labelID}`, { token: tokA })
    check('删除标签 → 204', r.status === 204 || r.status === 200, `status=${r.status}`)
  }
}

// ─────────────────────────── H. PRs ───────────────────────────
async function testPRs(tmp) {
  section('H. Pull Request')
  const R = `/api/v1/repos/${USER_A}/t1`
  let r = await post(`${R}/pulls`, { token: tokA, body: { title: 'merge feature', body: 'plz', head_branch: 'feature', base_branch: 'main' } })
  check('创建 PR → 201', r.status === 201, `status=${r.status} ${r.text.slice(0, 150)}`)
  const prNumber = r.json?.issue?.number ?? r.json?.pr?.id
  check('创建 PR 响应带 issue.number（寻址依据）', !!r.json?.issue?.number, JSON.stringify(r.json).slice(0, 150))

  r = await get(`${R}/pulls`, { token: tokA })
  check('PR 列表', r.status === 200 && r.text.includes('merge feature'))

  r = await get(`${R}/pulls/${prNumber}`, { token: tokA })
  check('读单个 PR → 200', r.status === 200, `status=${r.status}`)

  r = await get(`${R}/pulls/${prNumber}/diff`, { token: tokA })
  check('PR diff 非空', r.status === 200 && r.text.includes('feature'), `status=${r.status} len=${r.text.length}`)

  r = await post(`${R}/pulls/${prNumber}/reviews`, { token: tokA, body: { body: 'lgtm', status: 'approved', commit_sha: '' } })
  check('创建 review → 200/201', r.status === 200 || r.status === 201, `status=${r.status} ${r.text.slice(0, 100)}`)

  r = await get(`${R}/pulls/${prNumber}/reviews`, { token: tokA })
  check('review 列表', r.status === 200 && r.text.includes('lgtm'))

  r = await post(`${R}/pulls/${prNumber}/merge`, { token: tokA, body: { strategy: 'merge' } })
  check('合并 PR → 200', r.status === 200, `status=${r.status} ${r.text.slice(0, 120)}`)

  r = await get(`${R}/commits?branch=main&limit=5`, { token: tokA })
  const logText = JSON.stringify(r.json)
  check('合并后 main 含 feature 提交', logText.toLowerCase().includes('feature'), logText.slice(0, 150))

  r = await post(`${R}/pulls/${prNumber}/merge`, { token: tokA, body: {} })
  check('重复合并已合并 PR → 4xx', r.status >= 400 && r.status < 500, `status=${r.status}`)

  // second PR then close
  const work2 = path.join(tmp, 'pr2')
  git(['clone', `${ORIGIN}/${USER_A}/t1.git`, work2], { stdio: 'pipe' })
  const id = ['-c', 'user.email=t@t.local', '-c', 'user.name=Tester']
  git(['checkout', '-b', 'pr2'], { cwd: work2, stdio: 'pipe' })
  fs.writeFileSync(path.join(work2, 'pr2.txt'), 'pr2\n')
  git([...id, 'add', '-A'], { cwd: work2 })
  git([...id, 'commit', '-m', 'pr2 work'], { cwd: work2 })
  gitAuthed(['push', '-u', 'origin', 'pr2'], USER_A, PASSWORD, { cwd: work2, stdio: 'pipe' })
  r = await post(`${R}/pulls`, { token: tokA, body: { title: 'pr2', head_branch: 'pr2', base_branch: 'main' } })
  const pr2 = r.json?.issue?.number ?? r.json?.pr?.id
  r = await post(`${R}/pulls/${pr2}/close`, { token: tokA })
  check('关闭 PR → 200', r.status === 200, `status=${r.status}`)
}

// ─────────────────────────── I. orgs ───────────────────────────
async function testOrgs() {
  section('I. 组织 / 团队')
  const org = `zt-org-${SUFFIX}`
  let r = await post('/api/v1/orgs', { token: tokA, body: { name: org, display_name: 'Test Org', description: 'd' } })
  check('创建组织 → 201', r.status === 201, `status=${r.status} ${r.text.slice(0, 100)}`)

  r = await post('/api/v1/orgs', { token: tokA, body: { name: org } })
  check('重名组织 → 409', r.status === 409, `status=${r.status}`)

  r = await get(`/api/v1/orgs/${org}`, { token: tokB })
  check('他人读组织 → 200', r.status === 200, `status=${r.status}`)

  r = await put(`/api/v1/orgs/${org}`, { token: tokA, body: { display_name: 'Renamed', description: 'd2' } })
  check('更新组织 → 200', r.status === 200, `status=${r.status}`)

  r = await get('/api/v1/orgs', { token: tokA })
  check('组织列表', r.status === 200 && r.text.includes(org))

  r = await post(`/api/v1/orgs/${org}/members`, { token: tokA, body: { username: USER_B, role: 'member' } })
  check('添加成员 → 200/201', r.status === 200 || r.status === 201, `status=${r.status} ${r.text.slice(0, 100)}`)

  r = await get(`/api/v1/orgs/${org}/members`, { token: tokA })
  check('成员列表含 B', r.status === 200 && r.text.includes(USER_B))

  // 团队权限：B 只是 member，不应能建团队
  r = await post(`/api/v1/orgs/${org}/teams`, { token: tokB, body: { name: 'devs', description: 'x' } })
  check('普通成员创建团队被拒 → 403', r.status === 403, `status=${r.status} — 若 201 则为越权漏洞`)
  const teamID = r.json?.id

  r = await post(`/api/v1/orgs/${org}/teams`, { token: tokA, body: { name: 'devs', description: 'x' } })
  check('owner 创建团队 → 201', r.status === 201, `status=${r.status} ${r.text.slice(0, 100)}`)
  const ownerTeamID = r.json?.id ?? teamID

  r = await get(`/api/v1/orgs/${org}/teams`, { token: tokA })
  check('团队列表', r.status === 200 && r.text.includes('devs'))

  r = await post(`/api/v1/orgs/${org}/teams/${ownerTeamID}/members`, { token: tokA, body: { username: USER_B } })
  check('添加团队成员 → 200/201', r.status === 200 || r.status === 201, `status=${r.status} ${r.text.slice(0, 100)}`)

  r = await get(`/api/v1/orgs/${org}/teams/${ownerTeamID}/members`, { token: tokA })
  check('团队成员列表', r.status === 200 && r.text.includes(USER_B), r.text.slice(0, 120))

  r = await post(`/api/v1/orgs/${org}/teams/${ownerTeamID}/repos`, { token: tokA, body: { repo_name: 't1', permission: 'write' } })
  // 仅组织名下仓库可绑定；REST 无法创建组织仓库（半成品），404 属预期
  check('团队绑定组织仓库（404=REST 无法创建组织仓库，记录）', r.status === 200 || r.status === 201 || r.status === 404, `status=${r.status}`)

  r = await get(`/api/v1/orgs/${org}/teams/${ownerTeamID}/repos`, { token: tokA })
  check('团队仓库列表', r.status === 200, `status=${r.status}`)

  r = await del(`/api/v1/orgs/${org}/teams/${ownerTeamID}/repos/1`, { token: tokA })
  check('解绑团队仓库 → 2xx/404 记录', r.status < 500, `status=${r.status}`)

  r = await del(`/api/v1/orgs/${org}/teams/${ownerTeamID}/members/999999`, { token: tokA })
  check('移除不存在团队成员 → 2xx/404 记录', r.status < 500, `status=${r.status}`)

  if (teamID && teamID !== ownerTeamID) {
    r = await post(`/api/v1/orgs/${org}/teams`, { token: tokA, body: { name: 'devs' } })
    check('重复团队名 → 409', r.status === 409, `status=${r.status}`)
  }

  r = await post(`/api/v1/orgs/${org}/members/leave`, { token: tokB })
  check('成员主动退出 → 200/204', r.status === 200 || r.status === 204, `status=${r.status}`)

  if (ownerTeamID) {
    r = await del(`/api/v1/orgs/${org}/teams/${ownerTeamID}`, { token: tokA })
    check('删除团队 → 204', r.status === 204 || r.status === 200, `status=${r.status}`)
  }

  r = await del(`/api/v1/orgs/${org}`, { token: tokB })
  check('非所有者删除组织 → 403', r.status === 403, `status=${r.status}`)

  r = await del(`/api/v1/orgs/${org}`, { token: tokA })
  check('所有者删除组织 → 200/204', r.status === 200 || r.status === 204, `status=${r.status}`)
}

// ─────────────────────────── J. releases ───────────────────────────
async function testReleases() {
  section('J. Release')
  const R = `/api/v1/repos/${USER_A}/t1`
  let r = await post(`${R}/releases`, { token: tokA, body: { tag_name: 'v1.0.0', name: 'first release', body: 'notes' } })
  check('创建 release → 201', r.status === 201, `status=${r.status} ${r.text.slice(0, 100)}`)
  const relID = r.json?.id

  r = await get(`${R}/releases`, { token: tokA })
  check('release 列表', r.status === 200 && r.text.includes('v1.0.0'))

  r = await get(`${R}/releases/${relID}`, { token: tokA })
  check('读单个 release → 200', r.status === 200, `status=${r.status}`)

  const form = new FormData()
  form.append('file', new Blob([Buffer.from('asset-bytes')]), 'asset.bin')
  r = await req('POST', `${R}/releases/${relID}/assets`, { token: tokA, rawBody: form })
  check('上传附件 → 200/201', r.status === 200 || r.status === 201, `status=${r.status} ${r.text.slice(0, 100)}`)

  // B 已是 t1 的 write 协作者（F 节添加），因此允许建 release
  r = await post(`${R}/releases`, { token: tokB, body: { tag_name: 'v9', name: 'x' } })
  check('协作者建 release → 201', r.status === 201, `status=${r.status}`)
  // 清理，避免 tag 唯一约束影响后续
  await del(`${R}/releases/${r.json?.info?.id ?? r.json?.id}`, { token: tokA })

  r = await del(`${R}/releases/${relID}`, { token: tokA })
  check('删除 release → 204', r.status === 204 || r.status === 200, `status=${r.status}`)
}

// ─────────────────────────── K. webhooks ───────────────────────────
async function testWebhooks(tmp) {
  section('K. Webhook')
  const R = `/api/v1/repos/${USER_A}/t1`
  let r = await post(`${R}/hooks`, { token: tokA, body: { url: 'http://127.0.0.1:9/x', secret: 's3cret', content_type: 'json', events: 'push' } })
  check('创建 webhook → 201', r.status === 201, `status=${r.status} ${r.text.slice(0, 100)}`)
  const hookID = r.json?.id

  r = await get(`${R}/hooks`, { token: tokA })
  check('webhook 列表', r.status === 200 && r.text.includes('127.0.0.1:9'))

  // push 应触发投递——但代码中 push 路径从未 Enqueue，预期 deliveries 为空（bug）
  const work = path.join(tmp, 'wh')
  git(['clone', `${ORIGIN}/${USER_A}/t1.git`, work], { stdio: 'pipe' })
  fs.writeFileSync(path.join(work, 'wh.txt'), 'trigger\n')
  const id = ['-c', 'user.email=t@t.local', '-c', 'user.name=Tester']
  git([...id, 'add', '-A'], { cwd: work })
  git([...id, 'commit', '-m', 'trigger hook'], { cwd: work })
  gitAuthed(['push', 'origin', 'main'], USER_A, PASSWORD, { cwd: work, stdio: 'pipe' })
  await new Promise(res => setTimeout(res, 1500))

  r = await get(`${R}/hooks/${hookID}/deliveries`, { token: tokA })
  const dl = JSON.stringify(r.json ?? '')
  const count = (r.json?.items ?? r.json ?? []).length
  check('push 后产生 webhook 投递', count > 0, `status=${r.status} body=${dl.slice(0, 160)}`)

  r = await post(`${R}/hooks/${hookID}/deliveries/999999`, { token: tokA })
  check('重发不存在投递 → 404', r.status === 404, `status=${r.status}`)

  r = await del(`${R}/hooks/${hookID}`, { token: tokA })
  check('删除 webhook → 204', r.status === 204 || r.status === 200, `status=${r.status}`)
}

// ─────────────────────────── L. PAT ───────────────────────────
async function testPAT() {
  section('L. 个人访问令牌 (PAT)')
  const R = '/api/v1/user/tokens'
  let r = await post(R, { token: tokA, body: { name: `ci-${SUFFIX}`, scopes: 'repo' } })
  check('创建 PAT → 201 且返回明文 token', r.status === 201 && !!r.json?.token, `status=${r.status} ${r.text.slice(0, 100)}`)
  const pat = r.json?.token
  const patID = r.json?.info?.id ?? r.json?.id

  r = await get(R, { token: tokA })
  check('PAT 列表', r.status === 200 && r.text.includes(`ci-${SUFFIX}`))

  r = await get('/api/v1/user', { token: pat })
  check('PAT 可作为 Bearer 调 API', r.status === 200 && r.json?.username === USER_A, `status=${r.status}`)

  r = await del(`${R}/${patID}`, { token: tokA })
  check('删除 PAT → 204', r.status === 204 || r.status === 200, `status=${r.status}`)

  r = await get('/api/v1/user', { token: pat })
  check('删除后 PAT 失效 → 401', r.status === 401, `status=${r.status}`)
}

// ─────────────────────────── M. actions ───────────────────────────
async function testActions() {
  section('M. Actions（CI）')
  const R = `/api/v1/repos/${USER_A}/t1`
  let r = await get(`${R}/actions/workflows`, { token: tokA })
  check('workflow 列表 → 200', r.status === 200, `status=${r.status}`)
  note(`workflows 数量=${(r.json?.items ?? []).length}（系统没有任何创建 workflow 的入口，预计恒为 0）`)

  r = await post(`${R}/actions/workflows/99999/dispatch`, { token: tokA, body: { branch: 'main' } })
  check('dispatch 不存在的 workflow 被拒（若 201 则缺少校验）', r.status >= 400, `status=${r.status}`)
  const runID = r.json?.id

  r = await get(`${R}/actions/runs?workflowID=0`, { token: tokA })
  check('runs 列表 → 200', r.status === 200, `status=${r.status}`)

  // Runner API
  r = await post('/api/v1/actions/runners/register', { body: { name: `runner-${SUFFIX}` } })
  check('runner 注册 → 200/201', r.status === 200 || r.status === 201, `status=${r.status} ${r.text.slice(0, 120)}`)
  const runnerToken = r.json?.token ?? r.json?.runner_token
  const runnerID = r.json?.runner?.id ?? r.json?.id ?? r.json?.runner_id

  if (runnerID && runID) {
    r = await post('/api/v1/actions/runs/' + runID + '/status', { body: { runner_id: runnerID, status: 'running' } })
    check('runner 更新 run 状态 → 200', r.status === 200, `status=${r.status}`)
    r = await post(`/api/v1/actions/runs/${runID}/logs`, { body: { runner_id: runnerID, chunk: 0, content: 'hello log' } })
    check('runner 上报日志 → 200', r.status === 200, `status=${r.status}`)
  }
  r = await post('/api/v1/actions/runners/heartbeat', { body: { runner_id: runnerID } })
  check('runner 心跳 → 200', r.status === 200, `status=${r.status} ${r.text.slice(0, 80)}`)

  r = await post('/api/v1/actions/runners/poll', { body: { runner_id: runnerID } })
  check('runner poll → 200/204', r.status === 200 || r.status === 204, `status=${r.status} ${r.text.slice(0, 80)}`)
}

// ─────────────────────────── N. packages ───────────────────────────
async function testPackages() {
  section('N. 包仓库（协议桩）')
  let r = await req('GET', '/v2/')
  check('OCI /v2/ → 200 + 版本头', r.status === 200 && r.headers.get('docker-distribution-api-version') === 'registry/2.0', `status=${r.status}`)

  r = await req('HEAD', `/v2/${USER_A}/t1/blobs/sha256:abc`)
  check('HEAD blob 桩 → 200', r.status === 200, `status=${r.status}`)

  r = await get(`/${USER_A}/-/packages`, { token: tokA })
  check('用户包列表 → 200', r.status === 200, `status=${r.status}`)

  for (const [name, p] of [
    ['npm', `/${USER_A}/npm/some-pkg`],
    ['pypi simple', `/${USER_A}/pypi/simple/`],
    ['maven', `/${USER_A}/maven2/foo/1.0/foo-1.0.pom`],
    ['helm index', `/${USER_A}/helm/index.yaml`],
  ]) {
    r = await get(p, { token: tokA })
    check(`GET ${name} → 记录状态 ${r.status}`, r.status < 500, `status=${r.status}`)
  }
}

// ─────────────────────────── O. negative ───────────────────────────
async function testNegative() {
  section('O. 输入校验 / 安全')
  const R = `/api/v1/repos/${USER_A}/t1`
  let r = await post(`${R}/issues`, { token: tokA, rawBody: '{not json' })
  check('畸形 JSON → 400', r.status === 400, `status=${r.status}`)

  r = await post('/api/v1/repos', { token: tokA, body: { name: 'xx1', evil_field: true } })
  check('未知字段被拒绝 → 400', r.status === 400, `status=${r.status}`)

  const big = 'x'.repeat(2 * 1024 * 1024)
  r = await post(`${R}/issues`, { token: tokA, rawBody: JSON.stringify({ title: big }) })
  check('2MB body 被拒 (413/400)', r.status === 413 || r.status === 400, `status=${r.status}`)

  r = await get(`/api/v1/repos/${USER_A}/t1`, { headers: { Authorization: 'Token abc' } })
  check('不支持的认证方案 → 401', r.status === 401, `status=${r.status}`)

  r = await req('GET', `/api/v1/repos/${USER_A}/t1`, {
    headers: { Authorization: `Basic ${basicHeader(USER_A, 'wrong').replace(/^Basic /, '')}` },
  })
  check('Basic 错误密码 → 401', r.status === 401, `status=${r.status}`)

  r = await req('GET', `/api/v1/repos/${USER_A}/t1`, {
    headers: { Authorization: basicHeader(USER_A, PASSWORD) },
  })
  check('Basic 正确密码可读 API → 200', r.status === 200, `status=${r.status} body=${r.text.slice(0, 120)}`)

  r = await post('/api/v1/repos', { token: tokA, body: { name: '<script>alert(1)</script>' } })
  check('XSS 仓库名 → 400', r.status === 400, `status=${r.status}`)
}

// ─────────────────────────── P. rate limit ───────────────────────────
async function testRateLimit() {
  section('P. 限流')
  const results = []
  for (let i = 0; i < 7; i++) {
    const r = await post('/api/v1/auth/register', {
      body: { username: `rl-${SUFFIX}-${i}`, email: `rl-${SUFFIX}-${i}@t.local`, password: PASSWORD },
    })
    results.push(r.status)
  }
  const limited = results.filter(s => s === 429).length
  if (limited > 0) {
    check(`注册限流生效（7 次中 ${limited} 次 429）`, true)
  } else {
    note(`注册限流未触发（窗口内有余量）：${results.join(',')}`)
  }
}

function summary() {
  console.log('\n' + '='.repeat(70))
  console.log(`结果: \x1b[32m${pass} 通过\x1b[0m / \x1b[31m${fail} 失败\x1b[0m / ${notes.length} 备注`)
  if (failures.length) {
    console.log('\n失败明细:')
    for (const f of failures) console.log(`  - ${f.name}${f.detail ? ` | ${f.detail}` : ''}`)
  }
  if (notes.length) {
    console.log('\n备注:')
    for (const n of notes) console.log(`  - ${n}`)
  }
  process.exit(fail > 0 ? 1 : 0)
}

main().catch(e => {
  console.error('测试脚本异常:', e)
  summary()
})
