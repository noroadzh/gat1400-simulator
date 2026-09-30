# Tasks

## 1. CI 配置升级

- [x] 1.1 修改 `.github/workflows/ci.yml` 的 Lint job，把 `version: v1.61.0` 改为 `version: v2.7.2`；保留 `golangci/golangci-lint-action@v6.5.1` 不变；用 `grep -n 'version:' .github/workflows/ci.yml` 确认字段已更新
- [x] 1.2 删除仓库根目录空目录 `.golangci.yml/`（`rmdir .golangci.yml/`）并用 `ls -la .golangci*` 确认目录已消失

## 2. errcheck 修复（19 处）

- [x] 2.1 修 `cmd/gat1400-sim/main.go:118,124` —— `defer captureStore.Close()` 和 `defer nonceStore.Close()` 改为 `defer func() { _ = captureStore.Close() }()` / `defer func() { _ = nonceStore.Close() }()`；用 `grep -nE 'captureStore.Close|nonceStore.Close' cmd/gat1400-sim/main.go` 确认 defer 包装
- [x] 2.2 修 `internal/adapter/storage/capture_reader.go:91` —— `defer rows.Close()` 改为 `defer func() { _ = rows.Close() }()`
- [x] 2.3 修 `internal/adapter/storage/storage_test.go:175,176,177` —— `cs.Append(...)` 三处改为 `_ = cs.Append(...)`
- [x] 2.4 修 `internal/adapter/storage/storage_test.go:218,250` —— `defer os.Remove(tmp)` 两处改为 `defer func() { _ = os.Remove(tmp) }()`
- [x] 2.5 修 `internal/adapter/wire/client.go:134,155` —— 两处 `defer resp.Body.Close()` 改为 `defer func() { _ = resp.Body.Close() }()`
- [x] 2.6 修 `internal/ui/resources.go:196` —— `defer resp.Body.Close()` 改为 `defer func() { _ = resp.Body.Close() }()`
- [x] 2.7 修 `internal/ui/ws.go:121,123` —— 两处 `c.conn.SetReadDeadline(...)` 改为 `_ = c.conn.SetReadDeadline(...)`
- [x] 2.8 修 `test/e2e/capture_e2e_test.go:53` —— `ts.Listener.Close()` 改为 `_ = ts.Listener.Close()`
- [x] 2.9 修 `test/e2e/capture_e2e_test.go:60` —— `nodeSvc.UpsertNode(ctx, node.Node{...})` 返回 `(*node.Node, error)`，改为 `_, _ = nodeSvc.UpsertNode(...)`
- [x] 2.10 修 `test/e2e/capture_e2e_test.go:75` —— `io.ReadAll(resp.Body)` 改为 `_, _ = io.ReadAll(resp.Body)`
- [x] 2.11 修 `test/e2e/protocol_e2e_test.go:75` —— `ts.Listener.Close()` 改为 `_ = ts.Listener.Close()`
- [x] 2.12 修 `test/e2e/protocol_e2e_test.go:190` —— `defer conn.Close()` 改为 `defer func() { _ = conn.Close() }()`
- [x] 2.13 修 `test/e2e/protocol_e2e_test.go:200` —— `conn.SetReadDeadline(...)` 改为 `_ = conn.SetReadDeadline(...)`
- [x] 2.14 修 `internal/adapter/storage/storage_test.go:213` —— 实测补充：`cs.Append` 改为 `_ = cs.Append(...)`（原 plan 位置清单未列，实跑 lint 才发现）
- [x] 2.15 修 `internal/adapter/storage/storage_test.go:245` —— 实测补充：`cs.Append` 改为 `_ = cs.Append(...)`
- [x] 2.16 修 `test/e2e/protocol_e2e_test.go:262` —— 实测补充：辅助函数内 `defer resp.Body.Close()` 改为 `defer func() { _ = resp.Body.Close() }()`

## 3. ineffassign 修复（1 处）

- [x] 3.1 修 `internal/adapter/scenario/factory.go:47` —— 读函数上下文确认 `now` 变量未使用；若确认无用则改为直接表达式 `time.Now().UTC().Format(...)` 传给后续使用方，否则保留并加 `//nolint:ineffassign` 注释；用 `grep -n 'now' internal/adapter/scenario/factory.go` 确认

## 4. staticcheck 修复（3 处）

- [x] 4.1 修 `internal/adapter/wire/digest_test.go:103` —— 读测试上下文，处理空 `if` 分支：反转条件合并到正向分支，或删除该 `if`，或加 `t.Fatal("unexpected qop=auth")`；修改后用 `grep -nE 'Contains.*qop' internal/adapter/wire/digest_test.go` 确认
- [x] 4.2 修 `internal/app/logging/logger.go:168` —— `var closer func() error = func() error { return nil }` 改为 `closer := func() error { return nil }`；用 `grep -n 'closer' internal/app/logging/logger.go` 确认
- [x] 4.3 修 `internal/domain/ids/ids_test.go:43` —— 读测试循环结构，把 `regexp.MatchString(\`^[0-9a-f]+$\`, n)` 提到循环外 `re := regexp.MustCompile(\`^[0-9a-f]+$\`)`，循环内改 `re.MatchString(n)`；用 `grep -nE 'regexp.MustCompile|MatchString' internal/domain/ids/ids_test.go` 确认编译只发生一次

## 5. unused 修复（4 处）

- [x] 5.1 修 `internal/adapter/httpapi/cascade.go:81` —— 删除私有方法 `func (r *subscribeRepo) get(id string) (map[string]any, bool)`；先用 `grep -rn '\\.get(' --include='*.go' internal/adapter/httpapi` 确认无外部调用者
- [x] 5.2 修 `internal/adapter/httpapi/cascade.go:122` —— 同上检查并删除 `func (r *subscribeRepo) getDisposition(...)`
- [x] 5.3 修 `internal/adapter/scenario/factory_test.go:83` —— 删除未引用的 `func mapsEqual(a, b map[string]any) bool`；先用 `grep -n 'mapsEqual' internal/adapter/scenario/factory_test.go` 确认仅 1 处定义
- [x] 5.4 修 `internal/adapter/scenario/loader.go:33` —— 删除结构体中未使用的字段 `mu sync.Mutex`；先用 `grep -nE '\\.mu|mu sync' internal/adapter/scenario/` 确认仅 1 处定义

## 6. 验证

- [x] 6.1 用兼容 linter 复跑 `/tmp/golangci-lint-2.7.2-darwin-amd64/golangci-lint run --timeout=5m ./...`，确认输出 `0 issues`（`echo "EXIT=${PIPESTATUS[0]}"` 取退出码 0）
- [x] 6.2 用 `go vet ./...` 确认零警告（无 errcheck 副作用）
- [x] 6.3 用 `go build ./...` 确认全部包构建成功
- [x] 6.4 用 `go test -race -count=1 ./...` 确认全部测试通过（含 e2e）

## 7. 提交与归档

- [x] 7.1 用 `git status` 确认所有改动落在预期文件清单内（CI 配置 + 11 个代码文件 + 空目录删除）；无意外文件
- [x] 7.2 用 `git add .github/workflows/ci.yml <修改的 go 文件>` 后 `git commit -m "fix(ci): 升级 golangci-lint 到 v2.7.2 兼容 go 1.25.5 + 修 27 个 lint issues"`；完成后 `git log --oneline -1` 显示新 commit
- [x] 7.3 `git push` 推送到 origin/main；完成后用 `git log --oneline origin/main -1` 确认
- [ ] 7.4 `openspec archive fix-ci-lint-version-mismatch` 将本 change 归档到 `openspec/changes/archive/2026-09-30-...` 并合并 spec delta 到主 spec；完成后用 `ls openspec/changes/archive/` 确认