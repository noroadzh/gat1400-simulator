# Tasks

## 1. 重新统计真实覆盖率

- [ ] 1.1 跑 `go test -race -coverpkg=./internal/...,./cmd/... -coverprofile=/tmp/cov.out ./...`,记录每个包的真实覆盖率(行/语句)
- [ ] 1.2 对比 `openspec/specs/testing/spec.md` 当前数字,确认差异,把"现实数字"写入 `openspec/changes/refine-test-coverage/notes/actual-coverage.md`(临时笔记,归档前删除)

## 2. Engine 单元测试补齐(本 change 的实际代码改动)

- [ ] 2.1 读 `internal/adapter/scenario/engine.go` 完整内容,识别 4 类未覆盖路径:Start(已存在/不存在/失败)、Stop(不存在/已停止)、AutoStart 回调、关闭后回调不触发
- [ ] 2.2 新建或补全 `internal/adapter/scenario/engine_test.go`,用 fake 端口(`ports.NodeStore`、`ports.CaptureStore`、场景 loader)替代真实现
- [ ] 2.3 跑 `go test -race -coverpkg=./internal/adapter/scenario ./internal/adapter/scenario`,确认覆盖率 ≥ 80%
- [ ] 2.4 `go vet ./... && go build ./... && go test -race ./...` 全绿

## 3. CI 覆盖率归一

- [ ] 3.1 改 `.github/workflows/ci.yml` 的 Test job:`go test -race ./...` → `go test -race -coverpkg=./internal/...,./cmd/... -coverprofile=coverage.out ./...`
- [ ] 3.2 Test job 新 step:`actions/upload-artifact@v4` 上传 `coverage.out`,`retention-days: 7`
- [ ] 3.3 Test job 新 step:`go tool cover -func=coverage.out | tail` 输出到 `$GITHUB_STEP_SUMMARY`
- [ ] 3.4 如 `Makefile` 有 `test:` 目标,同步更新(若仅转调 CI 命令则无需改)

## 4. spec 修订

- [ ] 4.1 修订 `openspec/specs/testing/spec.md`:替换"≥90%"Requirement 为分层目标,保留"豁免 MUST 显式标注"
- [ ] 4.2 修订 `openspec/specs/ci-quality-gates/spec.md`:在「CI MUST 跑测试」Scenario 下追加覆盖率归一口径
- [ ] 4.3 新建 `openspec/specs/end-to-end-testing/spec.md`(从 delta 拷过去,删 delta 标记)

## 5. 本地验证

- [ ] 5.1 `go vet ./... && go build ./... && go test -race ./...` 全绿
- [ ] 5.2 `go test -coverpkg=./internal/...,./cmd/... -coverprofile=/tmp/cov.out ./...` 跑通,`/tmp/cov.out` 非空
- [ ] 5.3 `go tool cover -func=/tmp/cov.out | grep -E 'scenario|engine'` 确认 Engine 包覆盖率 ≥ 80%
- [ ] 5.4 `openspec validate refine-test-coverage --strict` 通过

## 6. 归档

- [ ] 6.1 `git status` 确认改动仅落在:`internal/adapter/scenario/engine_test.go`、`.github/workflows/ci.yml`、specs 三处、临时笔记
- [ ] 6.2 `git add` 相关文件后 `git commit -m "test(scenario): 补 Engine 单元测试覆盖 80% + CI 覆盖率归一"`,`git log --oneline -1` 确认
- [ ] 6.3 `git push` 推送到 origin/main
- [ ] 6.4 `openspec archive refine-test-coverage --yes`,归档后 delta spec 已合并到主 spec,`openspec/changes/refine-test-coverage/` 已移入 archive
