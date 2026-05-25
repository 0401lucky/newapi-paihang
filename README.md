# NewAPI 排行榜

> 给 NewAPI 用户做的社区向外部排行榜。9 个有趣榜单（土豪/吃货/熬夜冠军…），支持 iframe 嵌入到 NewAPI 首页，部署到 Zeabur。

[![CI](https://github.com/yourname/newapi-leaderboard/actions/workflows/ci.yml/badge.svg)](https://github.com/yourname/newapi-leaderboard/actions)

## 特性

- 9 个核心榜单（土豪 / 散财 / 充值 / 吃货 / 死忠粉 / 美食家 / 单笔王 / 吞噬 / 熬夜冠军）
- 完整版主站 + 紧凑嵌入版 widget（iframe 任意域名）
- 轻量后台管理面板（缓存清理、临时隐藏用户、状态查看）
- 内存缓存 + singleflight 防击穿，对 NewAPI 主库压力极小
- 只读直连数据库，强制使用只读账号
- 单 Docker 镜像（~25MB），Zeabur 一键部署

## Zeabur 部署步骤

1. **建只读 MySQL 账号**（在 NewAPI 的 MySQL 实例执行）：
   ```sql
   CREATE USER 'lb_ro'@'%' IDENTIFIED BY '强密码';
   GRANT SELECT ON newapi.* TO 'lb_ro'@'%';
   FLUSH PRIVILEGES;
   ```
2. Zeabur 控制台 → New Project（或已有项目）→ **Add Service** → **Git** → 选本仓库
3. 自动识别 Dockerfile 构建
4. 设置环境变量（最少 `MYSQL_DSN` + `ADMIN_TOKEN`，参考 `.env.example`）
5. 绑定持久卷到 `/app/data`（存临时隐藏用户列表）
6. 绑定子域名（如 `leaderboard.your.com`）
7. 在 NewAPI 后台首页添加 iframe：
   ```html
   <iframe src="https://leaderboard.your.com/embed?tabs=rich,foodie,nightowl"
           width="100%" height="420" frameborder="0" loading="lazy"
           style="border-radius:14px"></iframe>
   ```

## 嵌入版 URL 参数

| 参数 | 默认 | 说明 |
|---|---|---|
| `tabs` | `EMBED_TABS_DEFAULT` | 逗号分隔，显示哪几个 Tab |
| `range` | `7d` | 默认时间窗 |
| `limit` | `5` | 每榜显示行数（上限 10） |
| `theme` | 跟随系统 | `light` / `dark` |
| `site` | `SITE_URL` | "查看完整版"跳转地址 |

## 环境变量

参考 [`.env.example`](.env.example)。最少：

- `MYSQL_DSN`（必填）— 强烈建议只读账号
- `ADMIN_TOKEN`（启用后台时必填）

## 本地开发

```bash
# 一键起 MySQL + 应用
docker compose up --build

# 或手动分别跑：
docker run -d --rm -p 3306:3306 \
  -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=newapi_test \
  -v $(pwd)/test/fixtures/seed.sql:/docker-entrypoint-initdb.d/seed.sql \
  --name lb_mysql mysql:8.0

# 后端（8080）
export MYSQL_DSN="root:root@tcp(localhost:3306)/newapi_test?parseTime=true"
export ADMIN_TOKEN=dev
go run ./cmd/leaderboard

# 前端开发热更新（5173，proxy /api → 8080）
cd web && pnpm install && pnpm dev
```

## 测试

```bash
# 后端单测
make test

# 集成测试（需 Docker，会启临时 MySQL）
make test-int
```

## 安全

- **强制只读账号**：`MYSQL_DSN` 仅授予 `SELECT` 权限
- **接口白名单**：仅返回 `user_id / name / value / extra`，不泄露 email/token 等
- **Admin 鉴权**：`crypto/subtle.ConstantTimeCompare` 比对 token
- **限流**：公开 API 200 req/min/IP
- **任意嵌入**：CORS `*` + `Content-Security-Policy: frame-ancestors *`

## 设计与计划文档

- 设计：[`docs/superpowers/specs/2026-05-24-newapi-leaderboard-design.md`](docs/superpowers/specs/2026-05-24-newapi-leaderboard-design.md)
- 计划：[`docs/superpowers/plans/2026-05-24-newapi-leaderboard.md`](docs/superpowers/plans/2026-05-24-newapi-leaderboard.md)

## License

MIT
