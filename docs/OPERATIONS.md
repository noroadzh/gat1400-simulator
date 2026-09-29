# 运维指南 — GAT 1400 协议模拟器

> 文档配套代码版本：v0.1.0
> 适用读者：运维、SRE、平台部署工程师

## 一、部署形态

### 1.1 单机二进制

```bash
# 编译（-s -w 去除符号表与 DWARF 信息）
go build -ldflags="-s -w" -o gat1400-simulator ./cmd/gat1400-sim/

# 启动
./gat1400-simulator --config configs/simulator.yaml
```

### 1.2 Docker（可选）

```dockerfile
# 构建阶段
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -ldflags="-s -w" -o gat1400-simulator ./cmd/gat1400-sim/

# 运行阶段
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/gat1400-simulator /usr/local/bin/
COPY --from=builder /app/configs /configs
COPY --from=builder /app/data /data
EXPOSE 19000 19001
CMD ["gat1400-simulator", "--config", "/configs/simulator.yaml"]
```

### 1.3 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `GAT1400_SERVER_PROTOCOL` | `:19001` | 协议服务端绑定地址 |
| `GAT1400_SERVER_CONTROL` | `:19000` | BFF 绑定地址 |
| `GAT1400_AUTH_REALM` | `viid` | Digest realm |
| `GAT1400_AUTH_USERNAME` | `admin` | Digest 用户名 |
| `GAT1400_AUTH_PASSWORD` | `admin` | Digest 密码 |
| `GAT1400_STORAGE_DATADIR` | `./data` | 数据目录 |
| `GAT1400_LOG_LEVEL` | `info` | 日志级别 |
| `GAT1400_IDS_SITECODE` | `41000000` | 默认区划码 |
| `GAT1400_IDS_INDCODE` | `30` | 默认行业代码 |

## 二、TLS 终止

模拟器不内置 TLS。生产部署请放在反向代理之后：

```
[nginx / 负载均衡]  ──HTTPS──►  :19001 (协议)  +  :19000 (BFF)
```

### 2.1 nginx 反向代理配置

```nginx
server {
    listen 443 ssl;
    server_name gat1400.example.com;

    ssl_certificate /etc/ssl/certs/gat1400.crt;
    ssl_certificate_key /etc/ssl/private/gat1400.key;

    # Web 控制面（含 SPA + WS）
    location / {
        proxy_pass http://127.0.0.1:19000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 60s;
    }

    # GA/T 1400.4 REST API
    location /viid/ {
        proxy_pass http://127.0.0.1:19001/viid/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_request_buffering off;
    }
}
```

## 三、监控

### 3.1 健康检查

```bash
curl http://localhost:19000/api/control/system/health
```

### 3.2 结构化日志

默认输出 JSON 行（机器可解析）：

```json
{"time":"2026-09-28T10:30:00Z","level":"INFO","msg":"server started","addr":":19001","component":"httpapi"}
```

可切换为人类可读：

```bash
./gat1400-simulator --log.format=text
```

### 3.3 关键指标

| 指标 | 采集方式 |
|------|---------|
| 节点总数 | `GET /api/control/stats` → `nodeCount` |
| 运行中场景数 | `GET /api/control/stats` → `scenarioRunningCount` |
| 抓包数据行数 | SQLite: `SELECT COUNT(*) FROM captures` |
| 各 Kind 资源数 | `GET /api/control/resources/:kind` → 列表长度 |

### 3.4 告警建议

| 触发条件 | 告警方式 |
|---------|---------|
| 节点离线 > 5 分钟 | WebSocket 推送 `node.status == error` |
| 抓包库超过 10 万行 | 通过 BFF 接口统计后触发 |
| 健康检查失败 | HTTP 探测 |

## 四、升级

1. 停止当前进程：`kill $(cat data/simulator.pid)`
2. 备份数据：`cp -r data data.bak.$(date +%Y%m%d)`
3. 替换新二进制
4. 重启：`./gat1400-simulator --config configs/simulator.yaml`

> 备注：SQLite schema 仅做 append-only 写入，升级无需迁移；nonce 表在启动时会自动清理 1 小时前的条目。

## 五、备份

```bash
# 抓包数据库
cp data/captures.db "data/captures-$(date +%Y%m%d).db"

# nonce 数据库（非关键）
cp data/nonces.db "data/nonces-$(date +%Y%m%d).db"
```

## 六、systemd 进程管理

```ini
[Unit]
Description=GAT 1400 Protocol Simulator
After=network.target

[Service]
Type=simple
User=noroadzh
ExecStart=/home/noroadzh/gat1400-simulator --config /home/noroadzh/configs/simulator.yaml
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal
WorkingDirectory=/home/noroadzh

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
| 启动卡住 | 端口被占用 | `lsof -i :19001` / `lsof -i :19000` |
| Digest 一直 401 | 用户名 / 密码不匹配 | 检查配置 `auth.username` / `auth.password` |
| 抓包为空 | Capture 中间件被禁用 | 确认 `capture.enabled: true` |
| WS 频繁断开 | 客户端读超时（30s） | 客户端重连；事件不会回放 |
| CPU 高 | 资源推送间隔太小 | 增大场景 YAML 中的 `interval` |

---

> 下一节建议阅读：[docs/TESTING.md](./TESTING.md)（测试方法、覆盖率）