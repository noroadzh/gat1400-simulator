# Tasks

## 1. CI 配置升级

- [x] 1.1 修改 `.github/workflows/ci.yml` Lint job,把 `uses: golangci/golangci-lint-action@v6.5.1` 改为 `@v7.0.1`,保留 `version: v2.7.2`;用 `grep -nE 'golangci-lint-action|version:' .github/workflows/ci.yml` 确认 action 与 lint 版本字段均正确

## 2. Windows test 句柄修复

- [x] 2.1 修改 `internal/app/logging/daily_test.go::TestDailyWriter_RotateIsSerialized`,在 `t.Cleanup(func() { d.stop() })` 之后追加 `_ = rot.Close()`,确保 lumberjack 文件句柄在 TempDir cleanup 前关闭;用 `grep -n 'rot.Close' internal/app/logging/daily_test.go` 确认改动

## 3. 本地验证

- [x] 3.1 用本地兼容 linter(`/tmp/golangci-lint-2.7.2-darwin-amd64/golangci-lint run --timeout=5m ./...`)复跑,确认输出 `0 issues` 且退出码 0
- [x] 3.2 用 `go vet ./...` 确认零警告
- [x] 3.3 用 `go build ./...` 确认全部包构建成功
- [x] 3.4 用 `go test -race -count=1 ./internal/app/logging/...` 确认 `TestDailyWriter_RotateIsSerialized` 在 mac 上通过(本地无法跑 windows,但 mac 不复现即说明 cleanup 路径正确,Windows 平台修复留待 CI 验证)

## 4. openspec 归档

- [x] 4.1 `git status` 确认所有改动落在 `.github/workflows/ci.yml` 与 `internal/app/logging/daily_test.go` 两个文件,无意外文件
- [x] 4.2 `git add .github/workflows/ci.yml internal/app/logging/daily_test.go` 后 `git commit -m "fix(ci): 升级 golangci-lint-action 到 v7.0.1 兼容 lint v2 + 修 Windows lumberjack 句柄泄漏"`;`git log --oneline -1` 显示新 commit
- [x] 4.3 `git push` 推送到 origin/main,`git log --oneline origin/main -1` 确认
- [ ] 4.4 `/opsx:archive fix-ci-golangci-action-v7-windows-test` 将本 change 归档,合并 delta spec 到主 spec,提交归档 commit 并 push