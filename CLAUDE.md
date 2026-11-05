 # GA/T-1400模拟器系统规范

 ## 技术栈
 - 协议服务端: Go 1.25 / [labstack/echo](https://echo.labstack.com/)
 - 持久化: [modernc.org/sqlite](https://modernc.org/sqlite/)（纯 Go，无 CGO）
 - 控制面 BFF: Go 1.25 + Echo + WebSocket Hub（`:14080`，JSON API + WS） 
 - 前端 SPA: Vue 3.4 + `<script setup>` + Element Plus 2 + Vite 5 + TypeScript 5 
 - 前端容器: nginx:alpine，自带 SPA fallback + `/api` `/ws` 反向代理 
 - 后端容器: distroless/static-debian12，二进制 ~20MB 
 - 日志: `log/slog`（结构化 JSON / text 可切换） 
 - 协议: HTTP Digest 认证（RFC 2617，qop=auth）、`application/VIID+JSON`、Nonce 重放保护 
 - 并发同步: `golang.org/x/sync/singleflight`（BFF 缓存防击穿） 
 - 测试: `go test -race` / 黄金样本契约 / 真实 TCP socket e2e / `vue-tsc --noEmit` 

## 分层架构(严格单向依赖)
- cmd/  &rarr;  可执行程序入口
- configs/  &rarr;  配置文件，默认配置文件、YAML场景文件都在里面
- data/  &rarr;  SQLite数据库存放目录，运行时生成，gitignored中忽略
- docs/  &rarr;  项目文档按阶段分目录（索引见 docs/README.md）；国标 PDF 在 docs/07-standards/
- .cursor/rules/  &rarr;  Cursor Agent 规则（*.mdc）；根目录 .cursorrules 为摘要
- openspec/  &rarr;  OpenSpec变更管理文档记录所有功能、bug相关文档的描述、设计、任务等，gitignored中忽略，仅本地维护
- web/  &rarr;  前端页面工程
- internal/  &rarr;  服务端golang源码实现，企业级代码结构
- test/  &rarr;  集成测试、e2e、契约、单元测试

## 常用命令
### 1. 本地构建

```bash
# 后端
make build           # 输出 bin/gat1400-sim
make test            # 单元 + 集成测试（-race）
make e2e             # 端到端测试（需要空闲端口）

# 前端
cd web && npm install && npm run build   # 输出 web/dist/
cd web && npm run dev                     # 开发模式（自动代理 /api → :14080）
```

### 2. 本地启动（开发模式）

```bash
# 终端 1：启动后端（含协议端 :14000 + BFF :14080）
./bin/gat1400-sim --config configs/default.yaml

# 终端 2：启动前端开发服务器（端口 5173，自动代理 /api 到 :14080）
cd web && npm run dev

# 浏览器访问
#   http://localhost:5173  → 前端开发服务器
#   http://localhost:14080/api/control/system/health  → BFF 健康检查
#   http://localhost:14000/VIID/System/Time           → 协议端直连
```

### 3. Docker 部署（生产模式）

```bash
# 一键拉起整套服务
docker-compose up -d

# 端口映射
#   http://localhost:8080  → 前端 SPA（nginx，自动反代 /api /ws 到 backend）
#   http://localhost:14080 → 后端 BFF（JSON + WS，仅容器间通信也可用）

# 健康检查（容器内执行）
docker-compose exec backend ./gat1400-sim -healthcheck
```

浏览器访问 `http://localhost:8080` 即可使用 Web 控制台。

## 测试服务器

- 服务器ip: 10.96.1.125
- 用户名: root
- 项目所在服务器路径: /slow2/ga1400-simulator
- 部署方式: 基于docker容器部署