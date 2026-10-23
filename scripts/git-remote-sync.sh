#!/usr/bin/env bash

# 开启严格模式：
# -e  任一命令失败时让脚本尽快停止
# -u  访问未定义变量时报错，避免配置漏填
# -o pipefail  管道中任一环节失败都视为失败
set -euo pipefail

# 计算脚本自身所在目录，便于默认从脚本同目录读取配置文件。
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# 允许把配置文件路径作为第一个参数传入；不传时默认读取脚本同目录下的 git-remote-sync.conf。
CONFIG_FILE="${1:-$SCRIPT_DIR/git-remote-sync.conf}"

if [[ ! -r "$CONFIG_FILE" ]]; then
  echo "配置文件不存在: $CONFIG_FILE" >&2
  exit 1
fi

# 加载用户配置。
# shellcheck source=/dev/null
source "$CONFIG_FILE"

# ROOT_DIRS 是必填项，表示要递归扫描哪些工作区目录。
if ! declare -p ROOT_DIRS >/dev/null 2>&1; then
  echo "配置缺失: ROOT_DIRS" >&2
  exit 1
fi

# 这两个 token 也是必填项：
# PULL_URL_TOKEN  用于切到“拉取用”的 remote 地址
# PUSH_URL_TOKEN  用于切到“推送用”的 remote 地址
: "${PULL_URL_TOKEN:?配置缺失: PULL_URL_TOKEN}"
: "${PUSH_URL_TOKEN:?配置缺失: PUSH_URL_TOKEN}"

# 以下都是可选配置，给出默认值后，用户可以按需在配置文件中覆盖。
REMOTE_NAME="${REMOTE_NAME:-origin}"
SOURCE_BRANCH="${SOURCE_BRANCH:-main}"
TARGET_BRANCH="${TARGET_BRANCH:-saas/feat-nsMain-v0.0.0.0}"
SYNC_MODE="${SYNC_MODE:-pull-push}"
DRY_RUN="${DRY_RUN:-false}"
ALLOW_DIRTY="${ALLOW_DIRTY:-false}"
RESTORE_REMOTE_ON_FAILURE="${RESTORE_REMOTE_ON_FAILURE:-true}"
VERBOSE="${VERBOSE:-true}"

# 把配置中的数组统一复制到脚本内部变量中，便于后续处理。
declare -a ROOT_DIRS_ARRAY=("${ROOT_DIRS[@]}")
declare -a ONLY_PROJECTS_ARRAY=()
declare -a EXCLUDE_PROJECTS_ARRAY=()
declare -a URL_TOKEN_CANDIDATES_ARRAY=()

if declare -p ONLY_PROJECTS >/dev/null 2>&1; then
  ONLY_PROJECTS_ARRAY=("${ONLY_PROJECTS[@]}")
fi

if declare -p EXCLUDE_PROJECTS >/dev/null 2>&1; then
  EXCLUDE_PROJECTS_ARRAY=("${EXCLUDE_PROJECTS[@]}")
fi

if declare -p URL_TOKEN_CANDIDATES >/dev/null 2>&1; then
  URL_TOKEN_CANDIDATES_ARRAY=("${URL_TOKEN_CANDIDATES[@]}")
fi

log() {
  # 普通日志由 VERBOSE 控制，方便批量执行时减少输出噪声。
  if [[ "$VERBOSE" == "true" ]]; then
    echo "$*"
  fi
}

log_cmd() {
  # 在真正执行前，把命令完整打印出来，方便定位失败发生在哪一步。
  printf '%s' 'EXEC:'
  printf ' %q' "$@"
  printf '\n'
}

error() {
  # 错误日志统一输出到 stderr，便于重定向或日志采集。
  echo "$*" >&2
}

run_cmd() {
  # DRY_RUN 模式下只打印命令，不实际执行，适合第一次跑批前做确认。
  # 这里刻意以前缀 "# " 输出，避免有人把 dry-run 输出再次粘贴到 shell 时被当成命令执行。
  if [[ "$DRY_RUN" == "true" ]]; then
    printf '%s' '# DRY RUN:'
    printf '\n'
    printf ' %q' "$@"
    printf '\n'
    return 0
  fi

  log_cmd "$@"
  "$@"
}

contains_substring() {
  # Bash 原生子串判断封装，避免重复写条件表达式。
  local text="$1"
  local part="$2"
  [[ -n "$part" && "$text" == *"$part"* ]]
}

matches_project() {
  # 判断某个仓库是否应该被本次任务处理。
  # 支持 3 类匹配：
  # 1. 仓库名，例如 ca-user
  # 2. 仓库绝对路径
  # 3. 路径后缀，例如 some/group/ca-user
  local repo_path="$1"
  local repo_name
  repo_name="$(basename "$repo_path")"
  local include_match=true

  if ((${#ONLY_PROJECTS_ARRAY[@]} > 0)); then
    include_match=false
    local item
    for item in "${ONLY_PROJECTS_ARRAY[@]}"; do
      if [[ "$item" == "$repo_name" || "$item" == "$repo_path" || "$repo_path" == */"$item" ]]; then
        include_match=true
        break
      fi
    done
  fi

  if [[ "$include_match" != "true" ]]; then
    return 1
  fi

  local excluded
  for excluded in "${EXCLUDE_PROJECTS_ARRAY[@]}"; do
    if [[ "$excluded" == "$repo_name" || "$excluded" == "$repo_path" || "$repo_path" == */"$excluded" ]]; then
      return 1
    fi
  done

  return 0
}

rewrite_remote_url() {
  # 按 token 替换 remote url：
  # - 如果当前 url 已经包含目标 token，则直接返回当前 url
  # - 否则在候选 token 列表中找到命中的 token，再替换成目标 token
  # 这样做的好处是，不依赖固定域名或固定仓库路径，只要求 url 中 token 可识别。
  local current_url="$1"
  local target_token="$2"

  if contains_substring "$current_url" "$target_token"; then
    printf '%s\n' "$current_url"
    return 0
  fi

  local candidates=()
  candidates+=("${URL_TOKEN_CANDIDATES_ARRAY[@]}")
  candidates+=("$PULL_URL_TOKEN" "$PUSH_URL_TOKEN")

  local token
  for token in "${candidates[@]}"; do
    if contains_substring "$current_url" "$token"; then
      printf '%s\n' "${current_url//$token/$target_token}"
      return 0
    fi
  done

  return 1
}

ensure_clean_worktree() {
  # 在需要切换本地分支时，默认要求“已纳入 git 管理的文件”没有未提交改动。
  # 这里刻意忽略未跟踪文件，因为很多仓库本来就会生成本地临时文件、日志或调试文件，
  # 它们不参与版本管理，也不应该阻塞这类批量同步任务。
  local repo_path="$1"
  local status
  status="$(git -C "$repo_path" status --porcelain --untracked-files=no)"
  if [[ -n "$status" ]]; then
    error "存在已跟踪但未提交的改动，跳过: $repo_path"
    return 1
  fi
}

discover_repositories() {
  # 递归扫描 ROOT_DIRS 下的 git 仓库。
  # 同时兼容两种仓库形态：
  # 1. 标准仓库目录，存在 .git/ 目录
  # 2. worktree / 子模块等场景，.git 可能是一个文件
  local root
  for root in "${ROOT_DIRS_ARRAY[@]}"; do
    if [[ ! -d "$root" ]]; then
      error "目录不存在，跳过: $root"
      continue
    fi

    find "$root" \( -type d -name .git -o -type f -name .git \) -prune -print 2>/dev/null |
      while IFS= read -r git_marker; do
        local repo_path
        # git_marker 是 .git 目录或文件，仓库根目录就是它的父目录。
        repo_path="$(cd -- "$(dirname -- "$git_marker")" && pwd -P)"
        if matches_project "$repo_path"; then
          printf '%s\n' "$repo_path"
        fi
      done
  done
}

process_repository() {
  # 处理单个仓库的完整流程：
  # 1. 读取当前 remote url
  # 2. 生成 pull url 和 push url
  # 3. 切到 pull url，拿到远端 main 的最新状态
  # 4. 用远端 main 重建/更新本地目标分支
  # 5. 切到 push url，把本地目标分支推上去
  local repo_path="$1"
  local original_url
  original_url="$(git -C "$repo_path" remote get-url "$REMOTE_NAME" 2>/dev/null)" || {
    error "仓库没有 remote '$REMOTE_NAME'，跳过: $repo_path"
    return 1
  }

  local pull_url
  pull_url="$(rewrite_remote_url "$original_url" "$PULL_URL_TOKEN")" || {
    error "无法把 remote url 改写为 pull 地址，跳过: $repo_path"
    error "当前 url: $original_url"
    return 1
  }

  local push_url
  push_url="$(rewrite_remote_url "$original_url" "$PUSH_URL_TOKEN")" || {
    error "无法把 remote url 改写为 push 地址，跳过: $repo_path"
    error "当前 url: $original_url"
    return 1
  }

  log ""
  log "==> 处理仓库: $repo_path"
  log "原始 remote: $original_url"
  log "pull remote: $pull_url"
  log "push remote: $push_url"
  log "source branch: $SOURCE_BRANCH"
  log "local target branch: $TARGET_BRANCH"

  # 记录 remote 是否已被改写。
  # 这样如果中途失败，可以按配置决定是否恢复成最初的 remote url。
  local remote_changed=false

  restore_original_remote() {
    # 失败恢复只恢复 remote url，不回退本地分支状态。
    # 这是有意为之：remote url 可无损恢复，但本地分支是否应回退需要更谨慎，不能擅自重置。
    if [[ "$RESTORE_REMOTE_ON_FAILURE" == "true" && "$remote_changed" == "true" ]]; then
      run_cmd git -C "$repo_path" remote set-url "$REMOTE_NAME" "$original_url" || true
    fi
  }

  # 第一步：切换到 pull remote。
  if ! run_cmd git -C "$repo_path" remote set-url "$REMOTE_NAME" "$pull_url"; then
    error "切换 pull remote 失败: $repo_path"
    return 1
  fi
  remote_changed=true

  if [[ "$SYNC_MODE" == "fetch-push" || "$SYNC_MODE" == "pull-push" ]]; then
    # 两种模式都会改动本地目标分支，因此默认都要求工作区干净。
    if [[ "$ALLOW_DIRTY" != "true" ]]; then
      if ! ensure_clean_worktree "$repo_path"; then
        restore_original_remote
        return 1
      fi
    fi

    # 第二步：从 pull remote 拉取远端源分支的最新提交。
    if ! run_cmd git -C "$repo_path" fetch "$REMOTE_NAME" "$SOURCE_BRANCH"; then
      error "fetch 失败: $repo_path"
      restore_original_remote
      return 1
    fi

    # 第三步：把本地目标分支直接重建到 remote/source_branch。
    # 例如：checkout -B saas/feat-nsMain-v0.0.0.0 origin/main
    # 这正是你要求的“拉远端 main 到本地目标分支，而不是本地 main”。
    if ! run_cmd git -C "$repo_path" checkout -B "$TARGET_BRANCH" "$REMOTE_NAME/$SOURCE_BRANCH"; then
      error "切换本地目标分支失败: $repo_path"
      restore_original_remote
      return 1
    fi

    if [[ "$SYNC_MODE" == "pull-push" ]]; then
      # 第四步：在本地目标分支上继续执行 pull。
      # 当前 HEAD 已经是 TARGET_BRANCH，因此 pull origin main 的结果会更新当前本地目标分支。
      # 这里保留 pull，是为了严格贴合“切 remote -> pull main -> 切回 -> push”的操作语义。
      if ! run_cmd git -C "$repo_path" pull --ff-only "$REMOTE_NAME" "$SOURCE_BRANCH"; then
        error "pull 失败: $repo_path"
        restore_original_remote
        return 1
      fi
    fi
  else
    error "不支持的 SYNC_MODE: $SYNC_MODE"
    restore_original_remote
    return 1
  fi

  # 第五步：切换到 push remote，准备把本地目标分支推到目标地址。
  if ! run_cmd git -C "$repo_path" remote set-url "$REMOTE_NAME" "$push_url"; then
    error "切换 push remote 失败: $repo_path"
    restore_original_remote
    return 1
  fi

  # 第六步：推送“本地目标分支 -> 远程同名目标分支”。
  # 显式写成 refspec，是为了避免当前分支名变化或 upstream 配置带来的歧义。
  if ! run_cmd git -C "$repo_path" push "$REMOTE_NAME" "$TARGET_BRANCH:refs/heads/$TARGET_BRANCH"; then
    error "push 失败: $repo_path"
    restore_original_remote
    return 1
  fi

  return 0
}

main() {
  # 汇总所有命中的仓库，并做去重、排序，保证批量执行顺序稳定。
  local -a repos=()
  local repo
  while IFS= read -r repo; do
    repos+=("$repo")
  done < <(discover_repositories | awk '!seen[$0]++' | sort)

  if ((${#repos[@]} == 0)); then
    error "没有找到符合条件的 git 仓库"
    exit 1
  fi

  log "共找到 ${#repos[@]} 个待处理仓库"

  # 逐仓库处理，单仓库失败不会影响后续仓库继续执行。
  local success_count=0
  local failed_count=0

  for repo in "${repos[@]}"; do
    if process_repository "$repo"; then
      success_count=$((success_count + 1))
    else
      failed_count=$((failed_count + 1))
    fi
  done

  log ""
  log "完成: 成功 ${success_count}，失败 ${failed_count}"

  # 只要存在失败仓库，整个脚本返回非 0，方便 CI / 调度器识别异常。
  if ((failed_count > 0)); then
    exit 1
  fi
}

# 脚本入口。
main "$@"
