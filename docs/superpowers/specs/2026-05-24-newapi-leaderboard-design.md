# NewAPI 排行榜 · 设计文档

**版本**：v1.0
**日期**：2026-05-24
**状态**：待审查
**关联项目**：[QuantumNous/new-api](https://github.com/QuantumNous/new-api)、参考 [james-6-23/new_api_tools](https://github.com/james-6-23/new_api_tools) 的嵌入实现模式

---

## 1. 概述

### 1.1 目标
为基于 [NewAPI](https://github.com/QuantumNous/new-api) 搭建的服务，提供一个**有趣、可观赏、可嵌入**的用户排行榜外部服务。重点服务于"AI 角色扮演 / 酒馆"圈层用户社区，把"调用 / 充值 / 余额"等行为转化为带社区文化的榜单（土豪榜、吃货榜、熬夜冠军等），增强用户参与感和社区氛围。

### 1.2 用户场景
- **普通访客**：打开 NewAPI 首页就能在嵌入卡片里看到当前明星用户（"今天谁最能吃 Claude"）。
- **社区成员**：访问完整版主站，按时间维度切换查看各类榜单，输入自己的用户名查"我在哪"。
- **管理员**：通过后台面板配置隐藏用户、清理缓存、查看 API 调用统计。

### 1.3 约束
- 部署在 **Zeabur** 上，**不引入新数据库**，直连 NewAPI 同一个 MySQL 实例（只读账号）。
- 必须提供 **可嵌入式界面**（iframe），用于贴到 NewAPI 首页的"代码 / 网页链接"区域。
- 必须**保护 NewAPI 主库**：只读账号、不写入、限连接池、缓存兜底防压库。
- 性能优先：UI 设计选了毛玻璃风格但要求性能兼顾（局部模糊、移动端降级）。

### 1.4 非目标（v1 不做）
- 不做用户登录 / OAuth（嵌入版完全公开，主站完全公开）。
- 不做用户详情趋势图（v2 候选）。
- 不做 Redis / 分布式缓存（内存足够；如果用户量起来再加）。
- 不做新晋玩家、拉新王榜单（v2 候选）。
- 不做邮件 / 推送订阅。
- 不做多语言（v1 仅中文）。

---

## 2. 范围

### 2.1 9 个核心榜单

| 编号 | 名称 | 数据源 | 时间维度 |
|---|---|---|---|
| 1 | 💰 土豪榜 (rich) | `users.quota` | 实时（无时间维度，就是"现在"） |
| 2 | 💸 散财榜 (spender) | `users.used_quota` | 累计 / 7d / 30d（7d/30d 走 logs 聚合） |
| 3 | 💎 充值榜 (topup) | `top_ups.money` | 7d / 30d / 全部 |
| 4 | 🍴 吃货榜 (foodie) | `logs COUNT(*) WHERE type=2` | today / 7d / 30d |
| 5 | 🎯 死忠粉榜 (loyal) | `logs` 单模型占比 | 7d / 30d / 全部 |
| 6 | 🌈 美食家榜 (gourmet) | `logs COUNT(DISTINCT model_name)` | 7d / 30d / 全部 |
| 7 | 🔥 单笔王 (biteking) | `logs MAX(quota) WHERE type=2` | 7d / 30d / 全部 |
| 8 | ⚡ 吞噬榜 (tokens) | `logs SUM(prompt+completion)` | today / 7d / 30d |
| 9 | 🌙 熬夜冠军 (nightowl) | `logs COUNT(*) WHERE HOUR ∈ [0,5]` | 7d / 30d |

### 2.2 产品形态
- **完整版主站**：独立子域名（如 `leaderboard.example.com`），所有 9 个榜单 + 时间切换 + 个人查询 + Top 100 分页。
- **嵌入版 widget**：iframe 嵌入 NewAPI 首页，紧凑布局，顶部 Tab 切换 + Top 5。
- **管理面板**：`/admin` 路径，ADMIN_TOKEN 鉴权。

---

## 3. 整体架构

```
┌──────────────────────────────────────────────────────────────────┐
│ newapi 首页 (https://newapi.example.com)                          │
│  <iframe src="https://leaderboard.example.com/embed">             │
│      ┌──┬──┬──┬──┬──┐                                              │
│      │💰│💸│🍴│🎯│…│   ← 9 榜单 Tab + Top 5 + "查看完整版 →"        │
│      └──┴──┴──┴──┴──┘                                              │
└──────────────────────────────────────────────────────────────────┘
                       ↓ 点击 "查看完整版"
┌──────────────────────────────────────────────────────────────────┐
│ leaderboard.example.com                                           │
│ ┌──────────────────┐  ┌──────────────────────────────────┐        │
│ │ Go + Gin 进程    │  │ go:embed 前端静态资源              │        │
│ │ (单二进制)       │  │  / → index.html                   │        │
│ │ ┌──────────────┐ │  │  /embed → embed.html              │        │
│ │ │ 内存缓存层    │ │  │  /admin → admin.html              │        │
│ │ │ users TTL 60s│ │  └──────────────────────────────────┘        │
│ │ │ logs TTL 300s│ │  ┌──────────────────────────────────┐        │
│ │ └──────┬───────┘ │  │ HTTP API                          │        │
│ │ ┌──────────────┐ │  │  GET /api/leaderboard/:type       │        │
│ │ │ Repo 层 SQL  │ │  │  GET /api/rank/:keyword           │        │
│ │ └──────┬───────┘ │  │  GET /api/meta                    │        │
│ └────────┼─────────┘  │  POST /admin/*  (ADMIN_TOKEN)     │        │
│          ↓            └──────────────────────────────────┘        │
└──────────┼────────────────────────────────────────────────────────┘
           ↓ 只读 SELECT
┌──────────────────────────────────────────────────────────────────┐
│ NewAPI 同一 MySQL 数据库（不写入）                                 │
│  users / logs / top_ups 表                                        │
└──────────────────────────────────────────────────────────────────┘
```

**关键架构决策：**
1. **单二进制**：Go 编译产物把前端 dist 通过 `//go:embed` 嵌入，单镜像、单容器部署。
2. **Vite 三入口**：`index.html` / `embed.html` / `admin.html` 独立 bundle，互不干扰，嵌入版不加载主站的导航/页脚代码，bundle 更小。
3. **只读直连数据库**：性能、权限、运维都最简单。绝不写入 NewAPI 主库。
4. **分层内存缓存**：users 类榜单 60s TTL，logs 聚合 300s TTL；singleflight 防击穿。
5. **任意域名嵌入**：CORS 全开 + 不设 X-Frame-Options（避开 new_api_tools 的坑）+ CSP `frame-ancestors *`。

---

## 4. 数据库访问层

### 4.1 NewAPI Schema 关键假设

基于 [QuantumNous/new-api](https://github.com/QuantumNous/new-api) 的 `model/` 目录 Go 源码分析，本设计基于以下 schema 假设。**启动时会做字段存在性校验**（见 §13 错误处理）。

#### users 表
| 字段 | 类型 | 用途 |
|---|---|---|
| `id` | int | 用户 ID（主键，JOIN 关联键） |
| `username` | varchar | 登录名（display_name 为空时回退） |
| `display_name` | varchar | 昵称（优先显示） |
| `role` | int | 0/1/10/100，不过滤 |
| `status` | int | **必过滤 status=1**（启用） |
| `quota` | int | **必过滤**字段：当前剩余 token-quota，土豪榜数据源 |
| `used_quota` | int | 累计消费 token-quota，散财榜（累计）数据源 |
| `request_count` | int | 累计请求数（备用） |
| `created_at` | int64 | Unix 秒，注册时间 |
| `last_login_at` | int64 | Unix 秒，最后登录 |
| `deleted_at` | datetime/null | **必过滤 IS NULL**（软删除） |

**Quota 单位**：NewAPI 内部 `QuotaPerUnit = 500000`，即 `$1 = 500000 quota`。前端展示美元一律 `quota / 500000`。

#### logs 表
| 字段 | 类型 | 用途 |
|---|---|---|
| `id` | int | 主键 |
| `user_id` | int (index) | 聚合关键 |
| `created_at` | int64 (index) | **Unix 秒** |
| `type` | int | **必过滤 type=2**（消费）|
| `model_name` | varchar (index) | 死忠粉 / 美食家数据源 |
| `quota` | int | 本次消耗，单笔王数据源 |
| `prompt_tokens` | int | 吞噬榜累加项 |
| `completion_tokens` | int | 吞噬榜累加项 |

**logs.type 取值**：1=Topup, 2=Consume, 3=Manage, 4=System, 5=Error, 6=Refund。

#### top_ups 表
| 字段 | 类型 | 用途 |
|---|---|---|
| `id` | int | 主键 |
| `user_id` | int | 关联 |
| `money` | float | 美元金额，充值榜数据源 |
| `status` | varchar | **必过滤 status='success'** |
| `create_time` | int64 | Unix 秒 |

### 4.2 SQL 模板（核心）

所有 SQL 强制：
- `users` 必须 `WHERE status = 1 AND deleted_at IS NULL`
- `id NOT IN (?)` 应用 `HIDDEN_USER_IDS`（实施时：用 `sqlx.In()` 展开切片占位符，或 `HIDDEN_USER_IDS` 为空时省略整个 `AND id NOT IN (?)` 子句以避免空 IN 错误）
- 全部走参数化查询，禁止字符串拼接
- 时间窗 `created_at >= ?` 利用 `idx_created_at` 索引
- **时间窗定义（"today/7d/30d/all"）**：
  - `today` = 当前时间所在的 **+08:00 时区当日 0 点** 起到现在
  - `7d` = `now - 7 * 24h`
  - `30d` = `now - 30 * 24h`
  - `all` = 不加时间窗（仅 users 类榜单和 topup 的"全部"模式适用）

```sql
-- 1. 土豪榜
SELECT u.id, COALESCE(NULLIF(u.display_name,''), u.username) AS name, u.quota AS value
FROM users u
WHERE u.status = 1 AND u.deleted_at IS NULL AND u.id NOT IN (?)
ORDER BY u.quota DESC
LIMIT ? OFFSET ?;

-- 2. 散财榜（累计）
SELECT u.id, COALESCE(NULLIF(u.display_name,''), u.username) AS name, u.used_quota AS value
FROM users u
WHERE u.status = 1 AND u.deleted_at IS NULL AND u.id NOT IN (?)
ORDER BY u.used_quota DESC
LIMIT ? OFFSET ?;

-- 3. 充值榜
SELECT l.user_id AS id, COALESCE(NULLIF(u.display_name,''), u.username) AS name,
       SUM(l.money) AS value
FROM top_ups l JOIN users u ON l.user_id = u.id
WHERE l.status = 'success' AND l.create_time >= ?
  AND u.status = 1 AND u.deleted_at IS NULL AND u.id NOT IN (?)
GROUP BY l.user_id ORDER BY value DESC
LIMIT ? OFFSET ?;

-- 4. 吃货榜
SELECT l.user_id AS id, COALESCE(NULLIF(u.display_name,''), u.username) AS name,
       COUNT(*) AS value
FROM logs l JOIN users u ON l.user_id = u.id
WHERE l.type = 2 AND l.created_at >= ?
  AND u.status = 1 AND u.deleted_at IS NULL AND u.id NOT IN (?)
GROUP BY l.user_id ORDER BY value DESC
LIMIT ? OFFSET ?;

-- 5. 死忠粉榜：先聚合每个用户每个模型的调用次数，再算占比
SELECT id, name, top_model AS extra, ratio AS value FROM (
  SELECT l.user_id AS id,
         COALESCE(NULLIF(u.display_name,''), u.username) AS name,
         (SUBSTRING_INDEX(GROUP_CONCAT(l.model_name ORDER BY l.cnt DESC), ',', 1)) AS top_model,
         MAX(l.cnt) / SUM(l.cnt) AS ratio,
         SUM(l.cnt) AS total_calls
  FROM (
    SELECT user_id, model_name, COUNT(*) AS cnt
    FROM logs WHERE type = 2 AND created_at >= ?
    GROUP BY user_id, model_name
  ) l JOIN users u ON l.user_id = u.id
  WHERE u.status = 1 AND u.deleted_at IS NULL AND u.id NOT IN (?)
  GROUP BY l.user_id
  HAVING ratio >= ? AND total_calls >= ?  -- LOYAL_THRESHOLD, LOYAL_MIN_CALLS（防小样本噪音）
) t ORDER BY value DESC, total_calls DESC
LIMIT ? OFFSET ?;

-- 6. 美食家榜
SELECT l.user_id AS id, COALESCE(NULLIF(u.display_name,''), u.username) AS name,
       COUNT(DISTINCT l.model_name) AS value
FROM logs l JOIN users u ON l.user_id = u.id
WHERE l.type = 2 AND l.created_at >= ?
  AND u.status = 1 AND u.deleted_at IS NULL AND u.id NOT IN (?)
GROUP BY l.user_id ORDER BY value DESC
LIMIT ? OFFSET ?;

-- 7. 单笔王
SELECT l.user_id AS id, COALESCE(NULLIF(u.display_name,''), u.username) AS name,
       MAX(l.quota) AS value, MAX(l.model_name) AS extra
FROM logs l JOIN users u ON l.user_id = u.id
WHERE l.type = 2 AND l.created_at >= ?
  AND u.status = 1 AND u.deleted_at IS NULL AND u.id NOT IN (?)
GROUP BY l.user_id ORDER BY value DESC
LIMIT ? OFFSET ?;

-- 8. 吞噬榜
SELECT l.user_id AS id, COALESCE(NULLIF(u.display_name,''), u.username) AS name,
       SUM(l.prompt_tokens + l.completion_tokens) AS value
FROM logs l JOIN users u ON l.user_id = u.id
WHERE l.type = 2 AND l.created_at >= ?
  AND u.status = 1 AND u.deleted_at IS NULL AND u.id NOT IN (?)
GROUP BY l.user_id ORDER BY value DESC
LIMIT ? OFFSET ?;

-- 9. 熬夜冠军（+08:00 时区硬编码）
SELECT l.user_id AS id, COALESCE(NULLIF(u.display_name,''), u.username) AS name,
       COUNT(*) AS value
FROM logs l JOIN users u ON l.user_id = u.id
WHERE l.type = 2 AND l.created_at >= ?
  AND HOUR(CONVERT_TZ(FROM_UNIXTIME(l.created_at), @@session.time_zone, '+08:00')) BETWEEN 0 AND 5
  AND u.status = 1 AND u.deleted_at IS NULL AND u.id NOT IN (?)
GROUP BY l.user_id ORDER BY value DESC
LIMIT ? OFFSET ?;
```

> 注：第 5 项死忠粉用 `MySQL GROUP_CONCAT` 取占比最高的模型名作为 `extra`。如果 `model_name` 总长超过 `group_concat_max_len`（默认 1024 字节）需要在启动时 `SET SESSION group_concat_max_len = 65535`。

### 4.3 个人查询 SQL

`GET /api/rank/:keyword` 在 9 个榜单里查 keyword（先在 users 表 LIKE 匹配 display_name/username 拿到候选 user_id 集合，再对每个榜单计算该 user_id 的位次）：

```sql
-- 候选用户匹配
SELECT id, COALESCE(NULLIF(display_name,''), username) AS name
FROM users WHERE status=1 AND deleted_at IS NULL
  AND (display_name LIKE ? OR username LIKE ?)
LIMIT 10;

-- 单榜位次（示例：土豪榜）
SELECT COUNT(*) + 1 AS rank FROM users
WHERE status=1 AND deleted_at IS NULL
  AND quota > (SELECT quota FROM users WHERE id = ?);
```

---

## 5. 后端设计（Go + Gin）

### 5.1 技术栈
- Go 1.22+
- [Gin](https://github.com/gin-gonic/gin) HTTP 框架
- `database/sql` + [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql)（不引入 ORM，9 个 SQL 手写即可）
- [singleflight](https://pkg.go.dev/golang.org/x/sync/singleflight) 防缓存击穿
- [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate) 限流

### 5.2 模块划分（每模块单一职责）
```
internal/
├── config/       env 解析（强类型 Config 结构体）
├── db/           MySQL 连接池 + 启动时 schema 校验
├── repo/         9 个榜单 + 个人查询的 SQL（一榜一文件）
├── cache/        sync.Map + TTL + singleflight
├── service/      业务编排（缓存命中 → repo → 回写缓存）
├── handler/      Gin handler（参数校验 + 调 service + 拼响应）
├── middleware/   CORS / 限流 / admin auth / panic recovery
├── persist/      data/admin.json 读写（临时隐藏列表）
└── embed/        go:embed 前端静态资源 + 路由分发
```

### 5.3 HTTP API

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| GET | `/api/meta` | - | 9 个榜单元信息（emoji、名称、描述、支持的 range） |
| GET | `/api/leaderboard/:type` | - | 排行榜数据，`?range=&page=&page_size=` |
| GET | `/api/rank/:keyword` | - | 个人查询 |
| GET | `/api/health` | - | DB 连接 + 版本 |
| POST | `/admin/cache/clear` | ADMIN_TOKEN | 清缓存 |
| GET/POST/DELETE | `/admin/hidden-users` | ADMIN_TOKEN | 临时隐藏 ID 增删查 |
| GET | `/admin/stats` | ADMIN_TOKEN | 缓存命中率 / 调用次数 |

### 5.4 统一响应格式

```json
{
  "code": 0,
  "data": {
    "type": "rich",
    "range": "7d",
    "total": 1234,
    "page": 1,
    "page_size": 20,
    "list": [
      {
        "rank": 1,
        "user_id": 42,
        "name": "咸鱼想躺平",
        "value": 4216060000,
        "value_display": "$8,432.12",
        "extra": null
      }
    ],
    "updated_at": 1779612800,
    "cached": true
  }
}
```

- `value` 永远是原始数字（int64 token-quota 或 float USD），方便前端按榜单类型自定义格式
- `value_display` 后端预格式化字符串（默认展示）
- `extra` 因榜而异：死忠粉是 `{"model": "claude-sonnet-4"}`；单笔王是 `{"model": "..."}`；其余 null
- 错误：`{"code": 4xx, "msg": "...", "data": null}`

### 5.5 参数校验
- `type` 必须在 9 个白名单内，否则 400
- `range` 必须在 `today|7d|30d|all` 内，否则 400
- `page` ≥ 1, `page_size` ∈ [1, 100]
- 不在白名单的 query 参数忽略（不报错）

---

## 6. 前端设计（Vite + React）

### 6.1 技术栈
- Vite 5+
- React 18 + TypeScript
- Tailwind CSS 3
- TanStack Query (React Query) v5
- `react-router-dom` 仅主站用，嵌入版无路由
- 无状态管理库（URL `useSearchParams` + React Query 就够）

### 6.2 Vite 三入口
```ts
// web/vite.config.ts
export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      input: {
        main:  path.resolve(__dirname, 'index.html'),
        embed: path.resolve(__dirname, 'embed.html'),
        admin: path.resolve(__dirname, 'admin.html'),
      }
    }
  },
  server: {
    proxy: { '/api': 'http://localhost:8080', '/admin': 'http://localhost:8080' }
  }
})
```

打包后产物：`dist/{index,embed,admin}.html` + `dist/assets/*`，由 Go 服务器分发：
- `GET /` → `index.html`
- `GET /embed` → `embed.html`
- `GET /admin` → `admin.html`
- `GET /assets/*` → 静态资源（强缓存 1 年）
- `GET /api/*` → JSON API
- 其余路径 → 404

### 6.3 主站页面（`src/main.tsx` → `MainApp.tsx`）

```
┌──────────────────────────────────────────────────────┐
│  🏆 NewAPI 排行榜          [输入用户名查我在哪 →]    │
├──────────────────────────────────────────────────────┤
│  💰土豪│💸散财│💎充值│🍴吃货│🎯死忠│🌈美食│🔥单笔│⚡吞噬│🌙熬夜│
├──────────────────────────────────────────────────────┤
│  [今日][7天 *][30天][全部]            更新于 2 分钟前│
├──────────────────────────────────────────────────────┤
│  🥇  咸鱼想躺平                              $8,432.12│
│  🥈  克吹本吹                                $5,201.88│
│  🥉  小克的奴                                $3,667.40│
│  04   夜半敲键人                              $2,108.50│
│  ...                                                  │
│            [上一页] 第 1 / 5 页 [下一页]              │
└──────────────────────────────────────────────────────┘
```

- URL 状态化：`/?tab=foodie&range=7d&page=2` 可分享、可后退
- 默认 `tab=rich&range=7d&page=1`
- 每 60s 自动 refetch（与后端 users 缓存匹配）

### 6.4 组件清单（主站 + 嵌入版共用）
```
TabBar              榜单切换，支持横向滚动
RangeSwitcher       today/7d/30d/all
LeaderboardCard     单榜单容器（标题 + 行列表 + 分页/页脚）
RankRow             一行（rank/avatar/name/value），gold/silver/bronze 变体
Avatar              首字 + hash(user_id) 渐变色背景
PersonalRankWidget  主站专属，搜索框 + 9 榜结果卡
EmptyState          空数据插画 + 引导文案
ErrorState          错误提示 + 重试按钮
LoadingSkeleton     骨架屏
```

### 6.5 状态与数据获取
```ts
// hooks/useLeaderboard.ts
function useLeaderboard(type: string, range: string, page: number) {
  return useQuery({
    queryKey: ['lb', type, range, page],
    queryFn: () => api.getLeaderboard(type, range, page),
    staleTime: 60_000,         // 1 分钟内不重新拉
    refetchInterval: 60_000,   // 1 分钟后台自动 refetch
    retry: 3,
  })
}
```

**Tab 切换**：按需 fetch + TanStack Query 内存缓存。即用户切到某个 Tab 时才拉对应数据；之前看过的 Tab 数据在 `staleTime` 内复用，超过后下次切换重新拉。不做预加载（避免一进页面拉 9 次 SQL）。

### 6.6 样式（D 毛玻璃 + 性能优化）

**毛玻璃风格的边界**：
- ✅ 卡片头部（榜单标题区）用 `backdrop-filter: blur(8px)`
- ✅ 单行 `RankRow` 背景半透明（无 blur）
- ❌ **不**在外层容器用 backdrop-filter
- ❌ **不**用嵌套多层模糊

**渐变背景**：
- 用 CSS `radial-gradient` 在 body 上铺一层柔光底（紫粉色，跟选定风格 D 一致）
- 主题色 CSS 变量：`--brand-primary: #8b5cf6; --brand-secondary: #ec4899;`

**动效**：
- 只用 `transform` + `opacity`
- `motion-safe:` Tailwind 修饰，`prefers-reduced-motion` 时全部关闭
- 切换 Tab 用 `transition: transform 200ms ease-out`

**移动端 / 弱性能降级**：
```css
@media (max-width: 768px), (hover: none) {
  .glass { backdrop-filter: none !important; background: var(--card-solid); }
  .glow  { box-shadow: none !important; }
}
```

**长列表优化**：
- Top 100 列表用 `content-visibility: auto; contain-intrinsic-size: 56px;`
- 屏幕外的行跳过渲染，DOM 节点保留

### 6.7 头像方案
```ts
// 不走 Gravatar（避免 email 暴露 + 减少外部请求）
function avatarFor(userId: number, name: string) {
  const hue = (userId * 137.508) % 360  // 黄金分割
  return {
    background: `linear-gradient(135deg, hsl(${hue}, 70%, 60%), hsl(${(hue+40)%360}, 70%, 50%))`,
    text: (name?.[0] || '?').toUpperCase(),
  }
}
```

---

## 7. 嵌入式 widget 设计

### 7.1 入口
- 路径：`/embed`
- 入口文件：`embed.html` → `src/embed.tsx` → `EmbedApp.tsx`
- 极简壳：无导航、无页脚、无路由库、无 `react-router-dom`
- 推荐使用方式：
  ```html
  <iframe src="https://leaderboard.example.com/embed?tabs=rich,foodie,nightowl&range=7d"
          width="100%" height="420" frameborder="0" loading="lazy"
          style="border-radius:14px;overflow:hidden"></iframe>
  ```

### 7.2 URL 参数
| 参数 | 默认 | 说明 |
|---|---|---|
| `tabs` | `EMBED_TABS_DEFAULT` 后端配 | 逗号分隔，决定显示哪几个 Tab，例：`rich,foodie,nightowl` |
| `range` | `7d` | 默认时间窗 |
| `limit` | `5` | 每榜显示行数（嵌入版上限 10） |
| `theme` | 跟随系统 | `light` / `dark` |
| `site` | `SITE_URL` 后端配 | "查看完整版"跳转地址 |

### 7.3 与主站差异
- 不渲染：导航、个人查询、分页、页脚版权
- 简化：行高更紧凑、字号略小
- 底部一行 "查看完整版排行榜 →" `<a target="_top">`（防止在 iframe 内跳转）

### 7.4 跨域 / Frame 控制（重要）
- 后端 CORS：`Access-Control-Allow-Origin: *`、`Access-Control-Allow-Credentials: false`
- **不设** `X-Frame-Options`（这是 new_api_tools 的坑点）
- 代之以 `Content-Security-Policy: frame-ancestors *`（明确允许任意嵌入）
- 嵌入版 HTML 添加 `<meta http-equiv="X-Frame-Options" content="ALLOWALL">` 兜底（虽然实际无效，但是显式声明）

---

## 8. 管理面板（`/admin`）

### 8.1 鉴权
- 进入面板先要求输入 `ADMIN_TOKEN`
- 输入后存 `localStorage`，后续 API 请求带 `Authorization: Bearer <token>`
- 后端 `crypto/subtle.ConstantTimeCompare` 比对，防 timing attack
- 错误 3 次后前端冷静期 60s 防暴力（仅前端措施，后端限流保底）

### 8.2 功能
```
┌──────────────────────────────────────────────────────┐
│  🛠 NewAPI 排行榜 · 后台                  [退出登录] │
├──────────────────────────────────────────────────────┤
│  系统状态                                              │
│  ● DB 连接正常   缓存命中率 87.3%   24h API 调用 1,234│
├──────────────────────────────────────────────────────┤
│  缓存管理                                              │
│  [清空全部]  [按榜单清空 ▾]                           │
├──────────────────────────────────────────────────────┤
│  隐藏用户                                              │
│  环境变量隐藏（不可修改）：42, 17                      │
│  临时追加（持久化到 data/admin.json）：                │
│    [输入 user_id] [+ 添加]                            │
│    99 [×]    101 [×]                                  │
├──────────────────────────────────────────────────────┤
│  实时查询日志（最近 50 条）                            │
│  10:23:15  GET /api/leaderboard/rich?range=7d  45ms ✓ │
│  10:23:14  GET /api/leaderboard/foodie  102ms (miss)  │
└──────────────────────────────────────────────────────┘
```

### 8.3 持久化
- 临时隐藏列表 → `data/admin.json`（容器挂卷 `/app/data`）
- Zeabur 配置持久卷 mount 到 `/app/data`

---

## 9. 缓存策略

### 9.1 缓存层
```
内存缓存（sync.Map）
├── lb:rich:all:p1       → TTL 60s   (CACHE_TTL_USERS)
├── lb:spender:all:p1    → TTL 60s
├── lb:topup:7d:p1       → TTL 300s  (CACHE_TTL_LOGS)
├── lb:foodie:today:p1   → TTL 300s
├── ...
├── rank:夜半敲键人      → TTL 60s
└── meta:                → TTL 600s (元信息几乎不变)
```

### 9.2 缓存键格式
`lb:<type>:<range>:p<page>:s<page_size>`

### 9.3 singleflight 防击穿
同一 key 并发 N 个请求，只走 1 次 DB，其余等待结果广播。

### 9.4 旧数据兜底（stale-while-error）
DB 查询失败时，如果缓存里有旧数据（即使过期），返回旧数据 + `cached: true, stale: true`。前端可显示淡淡的"数据可能滞后"提示。

### 9.5 失效
- 自动：TTL 到期
- 手动：管理面板"清空全部" / "按榜单清空"

---

## 10. 性能与性能预算

### 10.1 性能目标
| 场景 | 目标 |
|---|---|
| API 缓存命中 | P95 < 50ms |
| API 缓存未命中（users 类） | P95 < 200ms |
| API 缓存未命中（logs 聚合，logs 表 100 万行） | P95 < 1.5s |
| 嵌入版首屏（FCP） | < 1.5s |
| 主站首屏（FCP） | < 2s |
| 前端 bundle（嵌入版 gzip） | < 60KB |
| 前端 bundle（主站 gzip） | < 120KB |

### 10.2 优化措施
- 后端：连接池 `max_open_conns=10`、prepared statement、singleflight
- 前端：Vite tree-shaking、`React.lazy` 拆分非首屏组件、`content-visibility` 跳过屏外渲染
- 网络：静态资源 `Cache-Control: public, max-age=31536000, immutable`、HTML `no-cache`、API `no-store`
- 移动端：自动降级毛玻璃和复杂动效（见 §6.6）

---

## 11. 安全

| 项 | 措施 |
|---|---|
| SQL 注入 | 100% 参数化查询，禁止字符串拼接 |
| XSS | React 默认转义，禁止 `dangerouslySetInnerHTML` |
| 数据库权限 | 强制只读账号 + 仅 `SELECT` 权限（README 显眼标注 + .env.example 注释） |
| 敏感字段泄露 | 接口 DTO 严格白名单：仅 `id/name/value/extra`，绝不输出 email/access_token/password/setting/aff_code 等 |
| Admin 鉴权 | `crypto/subtle.ConstantTimeCompare` 比对 token |
| 限流 | 公开 API 每 IP 200 req/min；admin API 不限流（已鉴权） |
| 持久化数据 | `data/admin.json` 权限 0600 |
| 日志 | 不打印 SQL 完整结果；不打印用户 PII；error 含 user_id 但不含 email |
| CORS | `*`，配合嵌入需求 |
| Frame | 不设 X-Frame-Options，CSP `frame-ancestors *` |

---

## 12. 部署

### 12.1 Dockerfile（多阶段）
```dockerfile
# Stage 1: 前端打包
FROM node:20-alpine AS frontend
WORKDIR /app
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

# Stage 2: Go 编译
FROM golang:1.22-alpine AS backend
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" \
    -o /out/leaderboard ./cmd/leaderboard

# Stage 3: 运行时
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
WORKDIR /app
COPY --from=backend /out/leaderboard .
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
  CMD wget -q --spider http://127.0.0.1:8080/api/health || exit 1
ENTRYPOINT ["/app/leaderboard"]
```

### 12.2 Go embed 写法
```go
import (
    "embed"
    "io/fs"
    "net/http"
)

//go:embed all:web/dist
var staticFS embed.FS

func registerStatic(r *gin.Engine) {
    sub, _ := fs.Sub(staticFS, "web/dist")
    r.GET("/", serveSPA(sub, "index.html"))
    r.GET("/embed", serveSPA(sub, "embed.html"))
    r.GET("/admin", serveSPA(sub, "admin.html"))
    r.GET("/assets/*filepath", gin.WrapH(
        http.StripPrefix("/", http.FileServer(http.FS(sub)))))
}
```

### 12.3 Zeabur 部署步骤（写入 README）
1. **建只读账号**：在 NewAPI 同一 MySQL 实例执行
   ```sql
   CREATE USER 'lb_ro'@'%' IDENTIFIED BY '<强密码>';
   GRANT SELECT ON newapi.* TO 'lb_ro'@'%';
   FLUSH PRIVILEGES;
   ```
2. **Zeabur 控制台** → New Project（或选已有项目）→ Add Service → Git → 选本仓库
3. Zeabur 自动识别 `Dockerfile` 构建
4. **环境变量**最少必填：`MYSQL_DSN`、`ADMIN_TOKEN`
5. **绑定持久卷**：mount 到 `/app/data`（存 admin.json）
6. **绑定子域名**（如 `leaderboard.example.com`）
7. NewAPI 同一项目部署时，`MYSQL_DSN` 用内网地址（如 `mysql.zeabur.internal:3306`）

### 12.4 GitHub Actions（CI + 镜像）
- `.github/workflows/ci.yml`：PR 触发，跑 `go test ./...` + `pnpm test` + `docker build`
- `.github/workflows/docker.yml`：push 到 main 或 tag 触发，构建多架构镜像（amd64+arm64）推送 `ghcr.io/<owner>/newapi-leaderboard:latest` + `:vX.Y.Z`

### 12.5 本地开发
- `docker-compose.yml`：拉一个临时 MySQL + 灌 `test/seed.sql`
- 后端 `go run ./cmd/leaderboard`（8080）
- 前端 `pnpm -C web dev`（5173，Vite proxy /api 和 /admin 到 8080）

---

## 13. 错误处理

### 13.1 启动期
| 错误 | 处理 |
|---|---|
| `MYSQL_DSN` 缺失 | Fatal，打印明确错误："MYSQL_DSN 未设置，参考 .env.example" |
| DB ping 失败 | Fatal，打印 DSN（脱敏密码）+ 错误详情，由 Zeabur 自动重启 |
| Schema 不兼容（关键字段缺失） | Fatal，列出缺失字段，提示对照 NewAPI 版本 |
| `ADMIN_TOKEN` 缺失 | Warn + 禁用 `/admin/*` 路由（不退出，仅前台可用） |
| `web/dist` 不存在 | Fatal："前端未构建，请先 cd web && pnpm build" |

### 13.2 运行期
| 错误 | HTTP 状态 | 行为 |
|---|---|---|
| 参数非法（type/range 不在白名单） | 400 | JSON `{code:400, msg:"invalid type"}` |
| DB 连接池超时 (>5s) | 503 | 优先返回缓存中的旧数据；无旧数据则 `{code:503, msg:"数据库繁忙"}` |
| 单次 SQL 失败 | 503 | 同上，stale-while-error |
| 个人查询未命中 | 200 | `{code:0, data:{found:false, keyword:"..."}}`（不算错） |
| 限流触发 | 429 | `{code:429, msg:"请求过于频繁，请稍后再试"}` |
| Admin 未授权 | 401 | `{code:401, msg:"unauthorized"}` |
| 未知 panic | 500 | recovery 中间件捕获，log + 通用错误响应 |

### 13.3 前端
- TanStack Query 默认 3 次指数退避重试
- 全部失败 → `<ErrorState>` 卡片 + "重试"按钮
- 网络断开（`navigator.onLine`）→ `<OfflineState>`，恢复后自动 refetch
- 用户友好文案：不直接显示 `Error: ECONNREFUSED`，而是"数据加载失败，请稍后再试"

---

## 14. 测试

### 14.1 后端
```
test/
├── unit/
│   ├── cache_test.go         # TTL 边界、并发安全、singleflight
│   ├── config_test.go        # env 解析、默认值、非法值
│   └── persist_test.go       # admin.json 读写、并发
├── integration/
│   ├── leaderboard_test.go   # dockertest 启 MySQL → seed.sql → 9 个榜单全跑一遍
│   ├── rank_test.go          # 个人查询
│   ├── admin_test.go         # 鉴权、清缓存、隐藏 ID
│   └── api_test.go           # 完整 HTTP 走一遍（含 CORS、限流）
└── fixtures/
    └── seed.sql              # 10 用户、200 logs、5 充值，覆盖各种边界
```

**关键覆盖：** 9 个榜单 SQL 全部有集成测试 — 任何 schema 改动会立刻被发现。

### 14.2 前端
```
web/src/__tests__/
├── components/
│   ├── RankRow.test.tsx
│   ├── TabBar.test.tsx
│   └── PersonalRankWidget.test.tsx
├── hooks/
│   └── useLeaderboard.test.ts
└── utils/
    └── avatar.test.ts
```
用 Vitest + React Testing Library + MSW（mock API）。

### 14.3 CI 流水线
GitHub Actions `ci.yml`：
1. checkout
2. setup Go + Node
3. `go test ./... -race -coverprofile=cover.out`
4. `cd web && pnpm install && pnpm test && pnpm build`
5. `docker build .` 验证构建通过
- 集成测试启动 MySQL 容器约慢 1-2 分钟，可接受

---

## 15. 项目目录结构

```
newapi-leaderboard/
├── cmd/leaderboard/main.go
├── internal/
│   ├── config/config.go
│   ├── db/{mysql.go, health.go, schema_check.go}
│   ├── repo/{rich.go, spender.go, topup.go, foodie.go, loyal.go,
│   │        gourmet.go, biteking.go, tokens.go, nightowl.go,
│   │        rank.go, repo.go}
│   ├── cache/{memory.go, singleflight.go}
│   ├── service/leaderboard.go
│   ├── handler/{leaderboard.go, rank.go, meta.go, admin.go, health.go}
│   ├── middleware/{cors.go, ratelimit.go, admin_auth.go, recovery.go}
│   ├── persist/admin_store.go
│   └── embed/static.go
├── web/
│   ├── src/
│   │   ├── main.tsx
│   │   ├── embed.tsx
│   │   ├── admin.tsx
│   │   ├── pages/{MainApp.tsx, EmbedApp.tsx, AdminApp.tsx}
│   │   ├── components/{TabBar, RangeSwitcher, LeaderboardCard,
│   │   │                RankRow, Avatar, PersonalRankWidget,
│   │   │                EmptyState, ErrorState, LoadingSkeleton}.tsx
│   │   ├── hooks/{useLeaderboard.ts, useUrlState.ts}
│   │   ├── api/client.ts
│   │   ├── styles/{globals.css, glass.css}
│   │   └── i18n/zh-CN.ts
│   ├── index.html
│   ├── embed.html
│   ├── admin.html
│   ├── vite.config.ts
│   ├── tailwind.config.ts
│   ├── tsconfig.json
│   ├── package.json
│   └── pnpm-lock.yaml
├── test/
│   ├── seed.sql
│   ├── unit/
│   ├── integration/
│   └── fixtures/
├── data/                       # 运行时持久化（gitignore）
├── docs/superpowers/specs/2026-05-24-newapi-leaderboard-design.md
├── .github/workflows/{ci.yml, docker.yml}
├── Dockerfile
├── docker-compose.yml
├── zeabur.toml
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---

## 16. 环境变量参考（`.env.example`）

```bash
# ===== 必填 =====
# NewAPI 数据库连接（强烈建议只读账号）
MYSQL_DSN=lb_ro:STRONG_PASSWORD@tcp(mysql.zeabur.internal:3306)/newapi?parseTime=true&loc=Asia%2FShanghai

# 后台管理令牌（不填则禁用 /admin 路由）
ADMIN_TOKEN=please-change-me-to-a-long-random-string

# ===== 常用可选 =====
PORT=8080
SITE_NAME=NewAPI 排行榜
SITE_URL=https://leaderboard.example.com
EMBED_TABS_DEFAULT=rich,foodie,nightowl

# 不上榜的用户 ID
HIDDEN_USER_IDS=

# 缓存（秒）
CACHE_TTL_USERS=60
CACHE_TTL_LOGS=300

# 死忠粉阈值（单模型调用占比 + 最少调用次数）
LOYAL_THRESHOLD=0.8
LOYAL_MIN_CALLS=10

# ===== 高级可选 =====
MYSQL_MAX_OPEN_CONNS=10
MYSQL_MAX_IDLE_CONNS=5
MYSQL_CONN_MAX_LIFETIME=300

RATE_LIMIT_PER_MIN=200

LOG_LEVEL=info
```

---

## 17. 验收标准（Definition of Done）

**功能**
- [ ] 9 个榜单全部能从 NewAPI MySQL 聚合出正确数据（seed.sql 验证）
- [ ] 时间维度 `today/7d/30d/all` 全部可用（对应榜单支持范围内）
- [ ] 个人查询能在 9 榜中返回正确排名；找不到时友好提示
- [ ] 嵌入版可被任意域名 iframe 嵌入；URL 参数 `tabs/range/limit/theme/site` 全部生效
- [ ] 管理面板 ADMIN_TOKEN 鉴权、缓存清理、临时隐藏增删均可用
- [ ] `HIDDEN_USER_IDS` 环境变量生效
- [ ] `LOYAL_THRESHOLD` 和 `LOYAL_MIN_CALLS` 环境变量生效

**质量**
- [ ] DB 只读账号在 README 和 `.env.example` 都有醒目标注
- [ ] 集成测试覆盖 9 个榜单 SQL
- [ ] CI（go test + pnpm test + docker build）通过
- [ ] 移动端 (375px) 布局不破，毛玻璃自动降级
- [ ] 性能指标达成（见 §10.1）

**交付**
- [ ] README 含 Zeabur 一图流部署步骤
- [ ] Docker 镜像可推到 GHCR
- [ ] 单 `docker-compose up` 起本地开发环境
- [ ] `.env.example` 完整、注释清晰

---

## 18. 风险与缓解

| 风险 | 影响 | 缓解 |
|---|---|---|
| NewAPI 升级改字段名 | 排行榜数据错误或挂掉 | 启动时 `SHOW COLUMNS` 校验关键字段；集成测试 |
| logs 表巨大（>1000 万行） | 聚合 SQL 慢、超时 | 全部时间窗 WHERE 走 `idx_created_at`；缓存 5 分钟兜底；SQL 超时 5s；stale-while-error |
| 公开排行榜引发用户隐私不满 | 投诉 | README 明确告知部署者风险；`HIDDEN_USER_IDS` + 管理面板让 admin 可自助退出 |
| 同名用户在排行榜混淆 | 用户体验 | display_name 不去重；个人查询用 LIKE 返回候选列表，user_id 是真正主键 |
| iframe 被浏览器拦 | NewAPI 首页嵌入失败 | 不设 X-Frame-Options；CSP `frame-ancestors *`；CORS 全开 |
| 缓存击穿瞬时高并发 | DB 瞬时高压 | singleflight 合并同 key 请求 |
| 误操作写入主库 | 数据破坏 | 强制只读账号 + 代码侧零 `Exec` 调用（只用 `Query`/`QueryRow`） |
| 死忠粉用 GROUP_CONCAT 超长 | SQL 报错 | 启动时 `SET SESSION group_concat_max_len = 65535` |
| 9 榜单略多导致界面臃肿 | UX | 主站 Tab 横滚；嵌入版默认 3 个 Tab，URL 可覆盖 |

---

## 19. 路线图

### v1（本次交付）
- 9 个核心榜单
- 主站 + 嵌入版 + 管理面板
- Docker 部署 + Zeabur 文档
- GitHub Actions CI + GHCR 镜像

### v2（候选）
- 🚀 新晋玩家榜、👥 拉新王榜（剩余 2 个 brainstorm 候选）
- 用户详情趋势图（点击用户名查最近 30 天调用 / 消费曲线）
- Redis 缓存层（用户量起来再加）
- E2E 测试（Playwright）
- 暗黑模式独立配色
- 多语言（英文）
- 邮件订阅"我上榜了"
- 排行榜历史快照（"上周冠军"、"本月新星"）

---

## 20. 参考资料

- NewAPI 仓库：https://github.com/QuantumNous/new-api
- NewAPI Models（用于本设计）：`model/user.go`、`model/log.go`、`model/topup.go`、`common/constants.go`
- 嵌入实现参考：https://github.com/james-6-23/new_api_tools（`frontend/embed.html` + `backend/internal/handler/model_status.go`）
- Zeabur 文档：https://zeabur.com/docs

---

**文档结束**
