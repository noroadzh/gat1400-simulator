# Design: vue-componentize-and-dockerize

## Overview

将 gat1400-simulator 的前端从手写 CDN 单文件 SPA 升级为 Vue 3 + Vite 组件化项目，同时引入 Docker 多阶段容器化。

---

## 1. Frontend Architecture

### 1.1 Tech Stack

```
web/
├── package.json          vue@^3.4 + vue-router@^4 + element-plus@^2
│                         vite@^5 + typescript@^5 + @vitejs/plugin-vue
│                         vue-tsc + @types/node
├── tsconfig.json         strict: true, target: ES2020
├── vite.config.ts        outDir: ../internal/ui/dist, base: /
│                         server.proxy /api → http://localhost:14080
├── index.html            Vite entry template
└── src/
    ├── main.ts           createApp + ElementPlus + router
    ├── App.vue           layout shell (Sidebar + Topbar + RouterView)
    ├── router.ts         7 routes
    ├── api/
    │   ├── control.ts    fetch 封装 /api/control/*
    │   └── ws.ts         WebSocket 自动重连
    ├── components/
    │   ├── Sidebar.vue       侧边导航（可折叠）
    │   ├── Topbar.vue        顶栏（标题 + WS 状态指示）
    │   ├── StatCard.vue      统计卡片（大数字 + 渐变）
    │   ├── LiveCaptureTable.vue  实时抓包表（auto-refresh）
    │   └── NodeFormDialog.vue    节点表单对话框
    └── views/
        ├── DashboardView.vue    仪表盘（卡片 + 抓包 feed）
        ├── NodesView.vue        节点管理（表格 + 表单）
        ├── ScenariosView.vue    场景管理（表格 + Start/Stop）
        ├── ResourcesView.vue    资源类型表（只读）
        ├── SubscriptionsView.vue  订阅/Dispositions JSON 展示
        ├── CapturesView.vue     抓包列表（过滤 + 导出）
        └── ConfigView.vue       配置 JSON 展示
```

### 1.2 Build Output

- `npm run build` → `../internal/ui/dist/`（相对于 `web/`）
- 产物：`index.html` + `assets/`（JS/CSS chunks）
- `.gitignore` 排除 `/internal/ui/dist/`

### 1.3 Development Proxy

`vite.config.ts` dev server 代理 `/api/*` → `http://localhost:14080`，让 `npm run dev` 时前端独立运行、调真实 backend。

### 1.4 Component Design

#### StatCard

```vue
<!-- props: label, value, color (primary|success|warn|danger), unit? -->
<div class="stat-card glass-card">
  <div class="stat-label">{{ label }}</div>
  <div class="stat-value gradient-text" :style="{ color }">
    {{ value }}{{ unit }}
  </div>
</div>
```

#### LiveCaptureTable

- `setInterval` 每 3s 调 `GET /api/control/captures`
- WebSocket 消息 `capture_added` 时自动刷新
- 列：Timestamp / Direction / Method / Path / Status / Duration

#### NodeFormDialog

- `el-dialog` + `el-form`
- 字段：ID / Name / Role / ListenPort / UpstreamURL / Capabilities
- 支持 Create / Update（POST 或 PUT）

### 1.5 UI Design System

**视觉语言**：Cyberpunk Neon + Glassmorphism（暗色仪表盘）

| 元素 | 值 |
|------|------|
| 背景 | `linear-gradient(135deg, #0b1020, #11183a)` |
| 卡片背景 | `rgba(255,255,255,0.05)` + `backdrop-filter: blur(14px)` |
| 边框 | `1px solid rgba(255,255,255,0.08)` |
| 圆角 | `14px` |
| 主色 | `#5b8cff`（蓝紫） |
| 辅助色 | `#8a5bff`（紫） |
| 成功色 | `#3ad29f`（青绿） |
| 警告色 | `#f7b955`（橙黄） |
| 危险色 | `#ff6b6b`（珊瑚红） |
| 文字主色 | `#e6e8f0` |
| 文字次色 | `#8c93b0` |
| 字体 | Inter, PingFang SC, system-ui |

---

## 2. Backend Changes

### 2.1 Embed 移除

```go
// internal/ui/static.go — 删除整段
// internal/ui/server.go — 删 spaHandler 和 /* catch-all
//                         （frontend 不再由 backend serve）
```

**注意**：SPA 路由在 Change 2 已实现（`spaHandler`）。Change 3 删除 `static.go` 后，backend 变成**纯 API 后端**（不再 serve 前端）。前端由 nginx:alpine 在 8080 端口独立提供。

### 2.2 路由调整

删除 `server.go` 中的 `s.e.GET("/*", spaHandler)` 和 `spaHandler` 函数（如果存在）。server.go 只保留 API + WS 路由。

---

## 3. Docker Architecture

### 3.1 Dockerfile.frontend

```dockerfile
FROM node:20-alpine AS build
WORKDIR /app
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM nginx:alpine
COPY --from=build /app/dist/ /usr/share/nginx/html/
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

### 3.2 Dockerfile.backend

```dockerfile
FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o gat1400-sim ./cmd/gat1400-sim/

FROM gcr.io/distroless/static-debian12
COPY --from=build /app/gat1400-sim /gat1400-sim
EXPOSE 14080
ENTRYPOINT ["/gat1400-sim"]
CMD ["--config", "/data/config.yaml"]
```

### 3.3 nginx.conf

```nginx
server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;

    # SPA fallback
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Proxy API to backend
    location /api/ {
        proxy_pass http://backend:14080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # Proxy WebSocket
    location /ws/ {
        proxy_pass http://backend:14080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### 3.4 docker-compose.yml

```yaml
services:
  backend:
    build:
      context: .
      dockerfile: Dockerfile.backend
    ports: ["14080:14080"]
    volumes:
      - ./data:/data
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:14080/api/control/system/health"]
      interval: 10s
      timeout: 5s
      retries: 3

  frontend:
    build:
      context: .
      dockerfile: Dockerfile.frontend
    ports: ["8080:80"]
    depends_on:
      backend:
        condition: service_healthy

networks:
  default:
    name: gat1400-net

volumes:
  data:
    driver: local
```

---

## 4. CI Pipeline

### 4.1 Updated ci.yml

```yaml
jobs:
  frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: web/package-lock.json
      - run: npm ci
        working-directory: web
      - run: npm run typecheck
        working-directory: web
      - run: npm run build
        working-directory: web
      - uses: actions/upload-artifact@v4
        with:
          name: frontend-dist
          path: web/dist/

  test:
    runs-on: ubuntu-latest
    needs: [frontend]  # 先确保 build 成功
    steps:
      - uses: actions/checkout@v4
      - run: go test -race -count=1 ./...
      - uses: actions/download-artifact@v4
        with:
          name: frontend-dist
      - run: ls dist/ && test -f dist/index.html

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: go vet ./...
```

---

## 5. Verification

| 验证项 | 命令 | 预期 |
|--------|------|------|
| 前端 build | `cd web && npm ci && npm run build` | 产出 `internal/ui/dist/index.html` + `assets/` |
| TypeScript 类型 | `npm run typecheck` | 0 错误 |
| 后端编译 | `go build ./...` | 成功（不依赖 dist/） |
| 后端测试 | `go test -race ./...` | 全绿 |
| docker-compose 语法 | `docker-compose config` | 无错误 |
| CI 拉取前端产物 | `actions/download-artifact` | dist/ 存在且含 index.html |

---

## 6. Implementation Notes

1. **npm build 产物位置**：输出到 `../internal/ui/dist/`（与旧 embed 路径一致），这样 `cmd/gat1400-sim/main.go` 如果将来要切回 embed 模式，只需加回 `//go:embed all:dist`

2. **本地开发**：开发时 `npm run dev`（Vite dev server 带 proxy），生产时 `docker-compose up`

3. **Element Plus 加载**：在 `main.ts` 中 `import ElementPlus from 'element-plus'` + `'element-plus/dist/index.css'`；全局注册组件用 `app.use(ElementPlus)`；不需要按需 import（首版简单优先）

4. **WS 重连**：`ws.ts` 用指数退避重连（max 3 次，间隔 1s/2s/4s），`onopen` 时触发 `requestAnimationFrame` 刷新数据
