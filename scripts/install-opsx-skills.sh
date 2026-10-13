#!/usr/bin/env bash
#===============================================================================
# install-opsx-skills.sh — 将 OpenSpec CLI 官方 skill 安装到 Claude Code
#
# 用法:
#   ./scripts/install-opsx-skills.sh          # 安装全部 12 个（首次/同步）
#   ./scripts/install-opsx-skills.sh --dry-run # 预览哪些会被安装/更新
#   ./scripts/install-opsx-skills.sh openspec-propose openspec-apply-change
#                                             # 只装指定的 skill
#
# 源: ~/.agentbro/skills/<name>/SKILL.md  (OpenSpec CLI 1.13.1 官方 skill)
# 目标: ~/.claude/skills/<name>/SKILL.md
#
# ⚠️  注意: openspec/ 目录本身由 .gitignore 排除，不进版本管理。
#           但如果你想把这个脚本本身纳入 git，脚本要进 .gitignore。
#===============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC_DIR="${HOME}/.agentbro/skills"
DST_DIR="${HOME}/.claude/skills"

# 所有 12 个官方 skill（OpenSpec CLI 1.13.1）
ALL_SKILLS=(
    openspec-apply-change
    openspec-archive-change
    openspec-bulk-archive-change
    openspec-continue-change
    openspec-explore
    openspec-ff-change
    openspec-new-change
    openspec-onboard
    openspec-propose
    openspec-sync-specs
    openspec-update-change
    openspec-verify-change
)

# ----------------------------------------------------------------------------
# 帮助信息
# ----------------------------------------------------------------------------
usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS] [SKILL...]

将 OpenSpec CLI 官方 skill 安装到 Claude Code。

Arguments:
  SKILL...    可选：只安装指定的 skill（默认全部 12 个）

Options:
  --dry-run   只显示将要执行的操作，不实际写入
  --help      显示本帮助

Examples:
  $(basename "$0")                      # 安装全部 12 个 skill
  $(basename "$0") --dry-run            # 预览
  $(basename "$0") openspec-propose     # 只安装 propose
  $(basename "$0") openspec-propose openspec-apply-change openspec-verify-change

Source: ${SRC_DIR}
Target: ${DST_DIR}
EOF
    exit 0
}

# ----------------------------------------------------------------------------
# 解析参数
# ----------------------------------------------------------------------------
DRY_RUN=false
SELECTED_SKILLS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --dry-run) DRY_RUN=true; shift ;;
        --help)    usage ;;
        -*)        echo "未知选项: $1"; usage ;;
        *)         SELECTED_SKILLS+=("$1"); shift ;;
    esac
done

# 默认全部
if [[ ${#SELECTED_SKILLS[@]} -eq 0 ]]; then
    SELECTED_SKILLS=("${ALL_SKILLS[@]}")
fi

# ----------------------------------------------------------------------------
# 前置检查
# ----------------------------------------------------------------------------
if [[ ! -d "$SRC_DIR" ]]; then
    echo "❌ 错误: 源目录不存在: $SRC_DIR" >&2
    echo "   请先确认 openspec CLI 已正确安装。" >&2
    exit 1
fi

if [[ ! -d "$DST_DIR" ]]; then
    echo "❌ 错误: 目标目录不存在: $DST_DIR" >&2
    echo "   请先启动 Claude Code 以初始化目录结构。" >&2
    exit 1
fi

# ----------------------------------------------------------------------------
# 安装逻辑
# ----------------------------------------------------------------------------
install_skill() {
    local name="$1"
    local src="${SRC_DIR}/${name}/SKILL.md"
    local dst="${DST_DIR}/${name}/SKILL.md"

    if [[ ! -f "$src" ]]; then
        echo "  ⚠ ${name}: 源文件不存在，跳过"
        return
    fi

    local action="installed"
    if [[ -L "$dst" || -d "$dst" || -f "$dst" ]]; then
        if diff -q "$src" "$dst" >/dev/null 2>&1; then
            echo "  ✓ ${name}: 已同步，无需更新"
            return
        fi
        action="updated"
        rm -rf "$dst"
    fi

    if $DRY_RUN; then
        echo "  → [dry-run] ${name}: 将被 ${action}"
    else
        mkdir -p "$(dirname "$dst")"
        cp "$src" "$dst"
        echo "  ✓ ${name}: 已 ${action} ($(wc -c < "$src" | tr -d ' ') bytes)"
    fi
}

echo "OpenSpec skills 安装脚本"
echo "========================"
echo "Source: $SRC_DIR"
echo "Target: $DST_DIR"
echo "Mode:   $(if $DRY_RUN; then echo "dry-run（预览）"; else echo "live（实际写入）"; fi)"
echo
echo "Skills: ${SELECTED_SKILLS[*]}"
echo

count=0
for skill in "${SELECTED_SKILLS[@]}"; do
    install_skill "$skill"
    ((count++)) || true
done

echo
echo "完成: ${count}/${#SELECTED_SKILLS[@]} skill 处理完毕"

if $DRY_RUN; then
    echo "(这是 dry-run 模式，无实际写入)"
fi
