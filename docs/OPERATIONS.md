# 运维指南 — GA/T 1400 协议模拟器

> 文档配套代码版本：v0.2.0
> 适用读者：运维、SRE、平台部署工程师

## 一、部署形态

### 1.1 Docker Compose（推荐）

```bash
# 一键拉起 backend (:14080) + frontend (:8080)
docker-compose up -d

# 验证
curl http://localhost:8080/api/control/system/health
curl http://localhost:14080/api/control/system/health
```

两个镜像均为多阶段构建：

| 镜像 | 构建阶段 | 运行阶段 | 说明 |
|------|---------|---------|------|
| `gat1400/backend` | `golang:1.25-alpine` | `gcr.io/distroless/static-debian12:nonroot` | 静态二进制，无 shell，最小攻击面 |
| `gat1400/frontend` | `node:20-alpine` | `nginx:alpine` | 自服务 SPA + 反代 `/api` `/ws` 到 backend |

- `data/` 目录通过 volume 挂载持久化（SQLite + 抓包数据）
- frontend 的 nginx 配置见 `nginx.conf`：SPA fallback + `/api/` `/ws/` 反向代理
- backend 健康检查（distroless 无 curl，使用 `/dev/tcp` 原生探测）

### 1.2 单机二进制

```bash
# 编译（-s -w 去除符号表与 DWARF 信息）
make build     # 输出 bin/gat1400-sim

# 启动
./bin/gat1400-sim --config configs/default.yaml
```

单机模式下，前端需自行构建（`cd web && npm run build`）并放到 nginx/Apache 上服务，或将 `web/dist/` 挂到任意静态服务器。BFF 与协议服务端口如下：

| 端口 | 用途 |
|------|------|
| `:14080` | 控制面 BFF（JSON API + WebSocket） |
| `:14000` | 协议服务端（GA/T 1400.4 REST API） |

### 1.3 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `GAT1400_PROTOCOL_LISTEN` | `:14000` | 协议服务端绑定地址 |
| `GAT1400_CONTROL_LISTEN` | `:14080` | BFF 绑定地址 |
| `GAT1400_AUTH_REALM` | `com.gat1400.simulator` | Digest realm |
| `GAT1400_AUTH_USERNAME` | `admin` | Digest 用户名 |
| `GAT1400_AUTH_PASSWORD` | `admin` | Digest 密码 |
| `GAT1400_STORAGE_PATH` | `./data/gat1400.db` | SQLite 文件路径 |
| `GAT1400_LOG_LEVEL` | `info` | 日志级别 |
| `GAT1400_CONFIG` | `configs/default.yaml` | 配置文件路径 |

## 二、TLS 终止

容器自身不内置 TLS。生产部署请在 nginx 反代或云负载均衡上配置 HTTPS：

```
[nginx / 负载均衡]  ──HTTPS──►  :8080 (前端)  +  :14080 (BFF)  +  :14000 (协议)
```

### 2.1 nginx 反向代理配置（参考）

```nginx
server {
    listen 443 ssl;
    server_name gat1400.example.com;

    ssl_certificate /etc/ssl/certs/gat1400.crt;
    ssl_certificate_key /etc/ssl/private/gat1400.key;

    # Web 控制面（frontend 容器自带 SPA fallback + /api /ws 反代）
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 60s;
    }

    # 直接访问 BFF（可选，一般走 frontend 的 /api 反代）
    location /api/ {
        proxy_pass http://127.0.0.1:14080;
        proxy_set_header Host $host;
    }

    # GA/T 1400.4 REST API
    location /viid/ {
        proxy_pass http://127.0.0.1:14000/viid/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_request_buffering off;
    }
}
```

## 三、监控

### 3.1 健康检查

```bash
# BFF
curl http://localhost:14080/api/control/system/health
# docker 模式经 frontend 反代
curl http://localhost:8080/api/control/system/health
```

### 3.2 结构化日志

默认输出 JSON 行（机器可解析）：

```json
{"time":"2026-09-29T10:30:00Z","level":"INFO","msg":"gat1400-simulator started","protocol_addr":":14000","control_addr":":14080"}
```

### 3.3 关键指标

| 指标 | 采集方式 |
|------|---------|
| 节点总数 | `GET /api/control/stats` → `nodeCount` |
| 在线节点数 | `GET /api/control/stats` → `onlineNodeCount` |
| 场景总数 | `GET /api/control/stats` → `scenarioCount` |
| 运行中场景数 | `GET /api/control/stats` → `runningScenarios` |
| 抓包数据行数 | SQLite: `SELECT COUNT(*) FROM captures` |

### 3.4 告警建议

| 触发条件 | 告警方式 |
|---------|---------|
| 节点离线 > 5 分钟 | WebSocket 推送 `node.status == error` |
| 抓包库超过 10 万行 | 通过 BFF 接口统计后触发 |
| 健康检查失败 | HTTP 探测 |

## 四、升级

### Docker 模式

```bash
docker-compose pull
docker-compose up -d
```

### 二进制模式

1. 停止当前进程：`kill $(cat data/simulator.pid)`
2. 备份数据：`cp -r data data.bak.$(date +%Y%m%d)`
3. 替换新二进制
4. 重启：`./bin/gat1400-sim --config configs/default.yaml`

> 备注：SQLite schema 仅做 append-only 写入，升级无需迁移；nonce 表在启动时会自动清理过期条目。

## 五、备份

```bash
# 抓包数据库
cp data/captures.db "data/captures-$(date +%Y%m%d).db"

# nonce 数据库（非关键）
cp data/nonces.db "data/nonces-$(date +%Y%m%d).db"

# 容器模式下 data/ 挂载在宿主机 ./data/，直接备份宿主机目录即可
```

## 六、systemd 进程管理

```ini
[Unit]
Description=GA/T 1400 Protocol Simulator
After=network.target

[Service]
Type=simple
User=noroadzh
ExecStart=/home/noroadzh/gat1400-simulator/bin/gat1400-sim --config /home/noroadzh/gat1400-simulator/configs/default.yaml
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal
WorkingDirectory=/home/noroadzh/gat1400-simulator

[Install]
WantedBy=multi-user.target
```

启用：`sudo systemctl enable gat1400-simulator`

## 七、性能容量参考

| 资源 | 经验值 |
|------|--------|
| CPU | 满载时每 50 个 device 节点约 1 核 |
| 内存 | 基础 50MB + 每个运行中场景约 10MB |
| 磁盘 | 每个活跃 device 节点每小时约 1MB 抓包数据 |
| SQLite | DB 超过 500MB 时建议手动 `VACUUM` |

```bash
# 压缩 SQLite
sqlite3 data/captures.db "VACUUM;"
```

## 八、故障排查

| 症状 | 诊断 | 处置 |
|------|------|------|
| 启动卡住 | 端口被占用 | `lsof -i :14000` / `lsof -i :14080` |
| Digest 一直 401 | 用户名 / 密码不匹配 | 检查配置 `auth.username` / `auth.password` |
| 抓包为空 | Capture 中间件被禁用 | 确认 `capture.enabled: true` |
| WS 频繁断开 | 客户端读超时（30s） | 客户端重连；事件不会回放 |
| CPU 高 | 资源推送间隔太小 | 增大场景 YAML 中的 `interval` |
| 前端 502 | backend 未启动 / healthcheck 未通过 | `docker-compose logs backend` |
| 前端资源 404 | dist 未构建 / nginx 容器未包含 | `docker-compose build --no-cache frontend` |

---

> 下一节建议阅读：[docs/TESTING.md](./TESTING.md)（测试方法、覆盖率）