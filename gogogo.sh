#!/bin/bash
if [ -z "${BASH_VERSION:-}" ]; then
    exec bash "$0" "$@"
fi
# ─────────────────────────────────────────────────────────────────────────────
# NACP gogogo.sh — 统一入口脚本
# 用法: ./gogogo.sh [选项]
#
# 部署流程：
#   默认（选项1）: git push → GitHub Actions 构建镜像 → 服务器 pull + restart
#   紧急（选项6）: 本地 docker build → push 到 GHCR → 服务器 pull + restart
# ─────────────────────────────────────────────────────────────────────────────

set +e  # Don't exit on error — we handle errors explicitly
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ─── 配置 ─────────────────────────────────────────────────────────────────────
IMAGE="ghcr.io/al90slj23/nacp:main"
DEPLOY_SERVER="nacp.m.srl"
DEPLOY_USER="root"
DEPLOY_DIR="/opt/nacp"
COMPOSE_FILE="docker-compose.yml"
CONTAINER_NAME="nacp"

# ─── 本地开发固定端口 ─────────────────────────────────────────────────────────
LOCAL_BACKEND_PORT="${NACP_LOCAL_BACKEND_PORT:-23900}"
LOCAL_FRONTEND_PORT="${NACP_LOCAL_FRONTEND_PORT:-23901}"
LOCAL_FRONTEND_HOST="${NACP_LOCAL_FRONTEND_HOST:-127.0.0.1}"
LOCAL_BACKEND_BIN="${NACP_LOCAL_BACKEND_BIN:-.tmp/nacp-local-backend}"
LOCAL_BACKEND_LOG="${NACP_LOCAL_BACKEND_LOG:-.tmp/nacp-local-backend.log}"
LOCAL_GO_BUILD_CACHE="${NACP_LOCAL_GO_BUILD_CACHE:-${SCRIPT_DIR}/.tmp/go-build-cache}"
ONLINE_DB_HOST="${NACP_ONLINE_DB_HOST:-${DEPLOY_SERVER}}"
ONLINE_DB_PORT="${NACP_ONLINE_DB_PORT:-3306}"

# ─── 颜色 ─────────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step()  { echo -e "${CYAN}[STEP]${NC} $1"; }

# ─── SSH 辅助 ─────────────────────────────────────────────────────────────────
SSH_KEY="${HOME}/.ssh/al90slj23"

remote_exec() {
    ssh -i "${SSH_KEY}" -o StrictHostKeyChecking=no -o ConnectTimeout=10 "${DEPLOY_USER}@${DEPLOY_SERVER}" "$@"
}

# ─── 菜单 ─────────────────────────────────────────────────────────────────────
show_menu() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  NACP — NewAPI Classic Plus  v0.1.0${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
    echo ""
    echo "  1) 部署（推送代码 → GitHub 构建 → 服务器更新）"
    echo "  2) 服务器状态"
    echo "  3) 服务器日志"
    echo "  4) 仅更新服务器（跳过构建，直接 pull 最新镜像）"
    echo "  5) 停止/重启服务器"
    echo "  6) 紧急部署（本地构建 → push 镜像 → 服务器更新）"
    echo "  7) 运行测试"
    echo "  8) 黑盒外部暴露面扫描"
    echo "  9) 本地黑盒启动并扫描"
    echo "  0) 本地开发环境"
    echo ""
    echo -e "  ${YELLOW}输入选项编号:${NC}"
}

# ─── 选项 0: 本地开发环境 ─────────────────────────────────────────────────────
ensure_web_dist() {
    # web/dist must exist for Go embed to compile (minimal placeholder is enough)
    if [ ! -d "web/dist" ]; then
        log_info "web/dist 不存在，创建占位目录（开发模式使用 Vite dev server）..."
        mkdir -p web/dist
        echo "<html><body>Use Vite dev server at :${LOCAL_FRONTEND_PORT}</body></html>" > web/dist/index.html
        log_info "占位目录已创建"
    fi
}

ensure_docker_ready() {
    if docker info >/dev/null 2>&1; then
        return 0
    fi
    if ! command -v docker >/dev/null 2>&1; then
        log_error "未找到 docker CLI，请先安装 Docker Desktop"
        exit 1
    fi
    log_info "Docker 未运行，正在启动 Docker Desktop..."
    if ! start_docker_desktop; then
        log_error "无法自动启动 Docker Desktop，请手动启动 Docker 后重试"
        exit 1
    fi
    local retries=60
    while ! docker info >/dev/null 2>&1; do
        retries=$((retries - 1))
        if [ $retries -le 0 ]; then
            log_error "Docker Desktop 启动超时，请手动启动后重试"
            exit 1
        fi
        sleep 2
    done
    log_info "Docker Desktop 已就绪"
}

start_docker_desktop() {
    if ! command -v open >/dev/null 2>&1; then
        return 1
    fi

    local app
    for app in \
        "/Applications/Docker.app" \
        "${HOME}/Applications/Docker.app" \
        "/Applications/Docker Desktop.app" \
        "${HOME}/Applications/Docker Desktop.app"; do
        if [ -d "$app" ] && open -gj "$app" >/dev/null 2>&1; then
            return 0
        fi
    done

    if open -b com.docker.docker >/dev/null 2>&1; then
        return 0
    fi

    if command -v mdfind >/dev/null 2>&1; then
        app=$(mdfind "kMDItemCFBundleIdentifier == 'com.docker.docker'" 2>/dev/null | head -n 1)
        if [ -n "$app" ] && [ -d "$app" ] && open -gj "$app" >/dev/null 2>&1; then
            return 0
        fi
    fi

    open -a Docker >/dev/null 2>&1 || open -a "Docker Desktop" >/dev/null 2>&1
}

port_pids() {
    local port=$1
    lsof -nP -tiTCP:${port} -sTCP:LISTEN 2>/dev/null
}

# 清理占用端口的进程
kill_port() {
    local port=$1
    local pids=$(port_pids "${port}")
    if [ -n "$pids" ]; then
        log_warn "端口 ${port} 被占用 (PID: ${pids//$'\n'/, })，正在清理..."
        kill -9 $pids 2>/dev/null
        sleep 1
    fi
}

kill_go_build_children_for_cwd() {
    local current_dir
    current_dir="$(pwd)"
    local pid cwd
    for pid in $(pgrep -f "/go-build.*/exe/main" 2>/dev/null); do
        cwd="$(lsof -a -p "$pid" -d cwd -Fn 2>/dev/null | sed -n 's/^n//p' | tail -n 1)"
        if [ "$cwd" = "$current_dir" ]; then
            log_warn "发现旧 go run 子进程 (PID: ${pid})，正在清理..."
            kill -9 "$pid" 2>/dev/null
        fi
    done
}

stop_local_backend_processes() {
    pkill -f "go run main.go" 2>/dev/null && log_info "后端 go run 已停止" || true
    pkill -f "$(pwd)/${LOCAL_BACKEND_BIN}" 2>/dev/null && log_info "后端本地二进制已停止" || true
    kill_go_build_children_for_cwd
    kill_port "${LOCAL_BACKEND_PORT}"
}

stop_local_frontend_processes() {
    pkill -f "bun run dev" 2>/dev/null && log_info "前端 bun dev 已停止" || true
    pkill -f "vite.*--port ${LOCAL_FRONTEND_PORT}" 2>/dev/null
    kill_port "${LOCAL_FRONTEND_PORT}"
}

build_local_backend() {
    mkdir -p "$(dirname "${LOCAL_BACKEND_BIN}")"
    mkdir -p "${LOCAL_GO_BUILD_CACHE}"
    log_info "构建本地后端: ${LOCAL_BACKEND_BIN}"
    if ! GOCACHE="${LOCAL_GO_BUILD_CACHE}" go build -o "${LOCAL_BACKEND_BIN}" main.go; then
        log_error "本地后端构建失败"
        exit 1
    fi
}

print_backend_log_tail() {
    if [ -f "${LOCAL_BACKEND_LOG}" ]; then
        log_warn "后端日志尾部 (${LOCAL_BACKEND_LOG}):"
        tail -n 120 "${LOCAL_BACKEND_LOG}"
    else
        log_warn "后端日志文件不存在: ${LOCAL_BACKEND_LOG}"
    fi
}

wait_backend_ready() {
    local url="http://localhost:${LOCAL_BACKEND_PORT}/api/status"
    local retries="${1:-180}"
    local backend_pid="${2:-}"
    local waited=0

    log_info "等待后端就绪: ${url}"
    while [ "$waited" -lt "$retries" ]; do
        if [ -n "$backend_pid" ] && ! kill -0 "$backend_pid" 2>/dev/null; then
            log_error "后端进程已退出，前端不会继续启动"
            print_backend_log_tail
            return 1
        fi
        if curl -fsS "$url" >/dev/null 2>&1; then
            log_info "后端已就绪"
            return 0
        fi
        sleep 1
        waited=$((waited + 1))
        if [ $((waited % 10)) -eq 0 ]; then
            log_info "后端仍在启动中（${waited}s）..."
        fi
    done

    log_error "后端 ${retries}s 内未就绪"
    print_backend_log_tail
    return 1
}

wait_frontend_ready() {
    local url="http://localhost:${LOCAL_FRONTEND_PORT}/"
    local retries="${1:-45}"
    local waited=0

    log_info "等待前端就绪: ${url}"
    while [ "$waited" -lt "$retries" ]; do
        if curl -fsS "$url" >/dev/null 2>&1; then
            log_info "前端已就绪"
            return 0
        fi
        sleep 1
        waited=$((waited + 1))
        if [ $((waited % 10)) -eq 0 ]; then
            log_info "前端仍在启动中（${waited}s）..."
        fi
    done

    log_error "前端 ${retries}s 内未就绪，请检查 Vite 启动日志"
    return 1
}

export_local_dev_env() {
    # 本地开发直接使用 nacp.m.srl 测试站数据库；数据库结构变更也在测试站库上验证。
    export_online_test_db_env
    export PORT="$LOCAL_BACKEND_PORT"
    export VITE_BACKEND_PORT="$LOCAL_BACKEND_PORT"
    export VITE_FRONTEND_PORT="$LOCAL_FRONTEND_PORT"
    export SESSION_SECRET="${SESSION_SECRET:-local_dev_secret}"
    export MEMORY_CACHE_ENABLED="${MEMORY_CACHE_ENABLED:-true}"
    export ERROR_LOG_ENABLED="${ERROR_LOG_ENABLED:-true}"
    export SKIP_DB_MIGRATION="${NACP_LOCAL_SKIP_DB_MIGRATION:-true}"
}

export_blackbox_dev_env() {
    export_local_dev_env
    export NACP_SECURITY_PROFILE=blackbox
    export NACP_BLACKBOX_LOGIN_PATH="${NACP_BLACKBOX_LOGIN_PATH:-/client-login}"
    export NACP_BLACKBOX_MASK_HEADERS=true
    export NACP_BLACKBOX_MASK_UNAUTH_RELAY=true
    export NACP_BLACKBOX_PUBLIC_REGISTER=false
    export NACP_BLACKBOX_PUBLIC_OAUTH=false
}

remote_env_value() {
    local key="$1"
    remote_exec "cd ${DEPLOY_DIR} && grep -E '^${key}=' .env 2>/dev/null | tail -n 1 | cut -d= -f2-" 2>/dev/null
}

local_env_value() {
    local key="$1"
    [ -f .env ] || return 0
    grep -E "^${key}=" .env 2>/dev/null | tail -n 1 | cut -d= -f2-
}

strip_env_quotes() {
    local value="$1"
    value="${value%\"}"
    value="${value#\"}"
    value="${value%\'}"
    value="${value#\'}"
    printf '%s' "$value"
}

rewrite_dsn_host() {
    local dsn="$1"
    printf '%s' "$dsn" | sed -E "s#@tcp\\([^)]*\\)#@tcp(${ONLINE_DB_HOST}:${ONLINE_DB_PORT})#"
}

mask_dsn() {
    printf '%s' "$1" | sed -E 's#(//?)[^/@]+@#\1***@#; s#(^[^:]+:)[^@]+@#\1***@#'
}

export_online_test_db_env() {
    local remote_sql_dsn remote_log_sql_dsn
    remote_sql_dsn="$(strip_env_quotes "$(local_env_value SQL_DSN)")"
    if [ -z "$remote_sql_dsn" ]; then
        remote_sql_dsn="$(strip_env_quotes "$(remote_env_value SQL_DSN)")"
    fi
    if [ -z "$remote_sql_dsn" ]; then
        log_error "无法从本地 .env 或 ${DEPLOY_SERVER}:${DEPLOY_DIR}/.env 读取 SQL_DSN"
        exit 1
    fi
    export SQL_DSN
    SQL_DSN="$(rewrite_dsn_host "$remote_sql_dsn")"

    remote_log_sql_dsn="$(strip_env_quotes "$(local_env_value LOG_SQL_DSN)")"
    if [ -z "$remote_log_sql_dsn" ]; then
        remote_log_sql_dsn="$(strip_env_quotes "$(remote_env_value LOG_SQL_DSN)")"
    fi
    if [ -n "$remote_log_sql_dsn" ]; then
        export LOG_SQL_DSN
        LOG_SQL_DSN="$(rewrite_dsn_host "$remote_log_sql_dsn")"
    else
        unset LOG_SQL_DSN
    fi
}

load_local_env_defaults() {
    [ -f .env ] || return 0
    while IFS='=' read -r key value; do
        case "$key" in
            SESSION_SECRET|MEMORY_CACHE_ENABLED|ERROR_LOG_ENABLED|SKIP_DB_MIGRATION|SYNC_FREQUENCY)
                if [ -n "${!key}" ]; then
                    continue
                fi
                value="${value%\"}"
                value="${value#\"}"
                value="${value%\'}"
                value="${value#\'}"
                export "$key=$value"
                ;;
        esac
    done < .env
}

stop_local_dev_processes() {
    log_info "停止本地开发进程..."
    stop_local_backend_processes
    stop_local_frontend_processes
}

local_dev() {
    local sub_choice="${1:-}"

    if [ -z "$sub_choice" ]; then
        echo "  a) 启动后端 (本地构建后运行)"
        echo "  b) 启动前端 (bun run dev)"
        echo "  c) 同时启动后端+前端 [默认]"
        echo "  r) 重启后端+前端（使用线上测试数据库）"
        echo "  m) 同时启动后端+前端，并执行数据库迁移"
        echo "  d) 停止本地开发"
        read -p "选择 [c]: " sub_choice
        sub_choice="${sub_choice:-c}"
    fi

    # Load non-DB env defaults safely, then force online test DB in export_local_dev_env.
    load_local_env_defaults

    case "$sub_choice" in
        a)
            ensure_web_dist
            stop_local_backend_processes
            export_local_dev_env
            build_local_backend
            log_info "启动后端 (端口 ${LOCAL_BACKEND_PORT})..."
            log_info "使用线上测试数据库: $(mask_dsn "$SQL_DSN")"
            log_info "按 Ctrl+C 停止"
            "${LOCAL_BACKEND_BIN}"
            ;;
        b)
            stop_local_frontend_processes
            log_info "启动前端开发服务器 (端口 ${LOCAL_FRONTEND_PORT}, 热更新)..."
            log_info "按 Ctrl+C 停止"
            (cd web && bun install --frozen-lockfile 2>/dev/null; bun run dev -- --host "${LOCAL_FRONTEND_HOST}" --port "${LOCAL_FRONTEND_PORT}")
            ;;
        c|r|restart|m|migrate)
            ensure_web_dist
            stop_local_dev_processes
            export_local_dev_env
            if [ "$sub_choice" = "m" ] || [ "$sub_choice" = "migrate" ]; then
                export SKIP_DB_MIGRATION=false
            fi
            build_local_backend
            log_info "启动后端+前端（热更新模式）..."
            log_info "后端: :${LOCAL_BACKEND_PORT} | 前端: :${LOCAL_FRONTEND_PORT} | 数据库: ${DEPLOY_SERVER}"
            log_info "浏览器访问: http://localhost:${LOCAL_FRONTEND_PORT}"
            log_info "后端 API: http://localhost:${LOCAL_BACKEND_PORT}"
            log_info "数据库 DSN: $(mask_dsn "$SQL_DSN")"
            log_info "后端日志: ${LOCAL_BACKEND_LOG}"
            log_info "数据库迁移: SKIP_DB_MIGRATION=${SKIP_DB_MIGRATION}"
            if [ -n "$LOG_SQL_DSN" ]; then
                log_info "日志数据库 DSN: $(mask_dsn "$LOG_SQL_DSN")"
            else
                log_warn "未配置 LOG_SQL_DSN，本地开发将使用业务库作为日志库"
            fi
            log_info "前端修改即时生效（Vite HMR），后端修改需重启本地后端"
            log_info "按 Ctrl+C 停止所有进程"
            echo ""
            (cd web && bun install --frozen-lockfile 2>/dev/null)
            : > "${LOCAL_BACKEND_LOG}"
            "${LOCAL_BACKEND_BIN}" > "${LOCAL_BACKEND_LOG}" 2>&1 &
            GO_PID=$!
            if ! wait_backend_ready 120 "$GO_PID"; then
                kill "$GO_PID" 2>/dev/null
                exit 1
            fi
            (cd web && bun run dev -- --host "${LOCAL_FRONTEND_HOST}" --port "${LOCAL_FRONTEND_PORT}") &
            BUN_PID=$!
            if ! wait_frontend_ready 60; then
                kill "$GO_PID" "$BUN_PID" 2>/dev/null
                exit 1
            fi
            if ! wait_backend_ready 120; then
                kill "$GO_PID" "$BUN_PID" 2>/dev/null
                exit 1
            fi
            trap "kill $GO_PID $BUN_PID 2>/dev/null; exit" INT TERM
            wait
            ;;
        d)
            stop_local_dev_processes
            ;;
        *)
            log_error "未知选项"
            ;;
    esac
}

# ─── 选项 1: 标准部署 ─────────────────────────────────────────────────────────
deploy() {
    log_step "1/4 提交并推送代码到 GitHub..."
    git add -A
    if git diff --cached --quiet; then
        log_info "没有新的更改需要提交"
    else
        read -p "提交信息: " commit_msg
        git commit -m "${commit_msg:-update}"
    fi
    git push origin main

    log_step "2/4 等待 GitHub Actions 构建..."
    log_info "构建已触发，通常需要 5 分钟"
    log_info "查看进度: https://github.com/al90slj23/NACP/actions"
    echo ""
    read -p "构建完成后按回车继续部署到服务器（或 Ctrl+C 取消）..."

    log_step "3/4 服务器拉取最新镜像..."
    remote_exec "docker pull ${IMAGE}"

    log_step "4/4 重启服务..."
    remote_exec "cd ${DEPLOY_DIR} && docker compose up -d"

    log_info "✅ 部署完成！"
    remote_exec "docker ps | grep ${CONTAINER_NAME}"
}

# ─── 选项 2: 服务器状态 ───────────────────────────────────────────────────────
server_status() {
    log_info "容器状态:"
    remote_exec "docker ps -a | grep -E '${CONTAINER_NAME}|nacp-mysql'" || log_warn "未找到容器"
    echo ""
    log_info "磁盘使用:"
    remote_exec "df -h / | tail -1"
    echo ""
    log_info "内存使用:"
    remote_exec "free -h | head -2"
}

# ─── 选项 3: 服务器日志 ───────────────────────────────────────────────────────
server_logs() {
    log_info "NACP 最近日志 (Ctrl+C 退出):"
    remote_exec "docker logs --tail 50 -f ${CONTAINER_NAME}"
}

# ─── 选项 4: 仅更新服务器 ─────────────────────────────────────────────────────
server_update() {
    log_step "拉取最新镜像..."
    remote_exec "docker pull ${IMAGE}"

    log_step "重启服务..."
    remote_exec "cd ${DEPLOY_DIR} && docker compose up -d"

    log_info "✅ 更新完成！"
    remote_exec "docker ps | grep ${CONTAINER_NAME}"
}

# ─── 选项 5: 停止/重启 ───────────────────────────────────────────────────────
server_control() {
    echo "  a) 重启服务"
    echo "  b) 停止服务"
    echo "  c) 启动服务"
    read -p "选择: " sub_choice
    case "$sub_choice" in
        a) remote_exec "cd ${DEPLOY_DIR} && docker compose restart" && log_info "已重启" ;;
        b) remote_exec "cd ${DEPLOY_DIR} && docker compose down" && log_info "已停止" ;;
        c) remote_exec "cd ${DEPLOY_DIR} && docker compose up -d" && log_info "已启动" ;;
        *) log_error "未知选项" ;;
    esac
}

# ─── 选项 6: 紧急部署（本地构建）──────────────────────────────────────────────
emergency_deploy() {
    log_warn "紧急部署：本地构建 → push 到 GHCR → 服务器更新"
    log_warn "需要 Docker Desktop 运行中"
    echo ""
    ensure_docker_ready

    log_step "1/4 本地构建 Docker 镜像 (linux/amd64)..."
    docker build --platform linux/amd64 -t "${IMAGE}" .

    log_step "2/4 登录 GHCR..."
    echo "$(gh auth token)" | docker login ghcr.io -u al90slj23 --password-stdin

    log_step "3/4 推送镜像到 GHCR..."
    docker push "${IMAGE}"

    log_step "4/4 服务器拉取并重启..."
    remote_exec "docker pull ${IMAGE} && cd ${DEPLOY_DIR} && docker compose up -d"

    log_info "✅ 紧急部署完成！"
    remote_exec "docker ps | grep ${CONTAINER_NAME}"
}

# ─── 选项 7: 运行测试 ─────────────────────────────────────────────────────────
run_tests() {
    log_info "运行单元测试..."
    go test ./service/ -run "TestHealth|TestOnUser|TestOnProbe|TestShouldProbe|TestCheckRecovery|TestGetChannel|TestIsChannel|TestProperty|TestDefault|TestGetHealth" -count=1 -v 2>&1 | grep -E "PASS|FAIL|ok"
    echo ""
    log_info "编译检查..."
    go build ./... 2>&1 | grep -v "web/dist" || log_info "✅ 编译通过"
}

# ─── 选项 8: 黑盒外部暴露面扫描 ───────────────────────────────────────────────
blackbox_scan() {
    local base_url="${1:-${NACP_BLACKBOX_SCAN_BASE_URL:-http://localhost:${LOCAL_BACKEND_PORT}}}"
    local login_path="${2:-${NACP_BLACKBOX_LOGIN_PATH:-/client-login}}"
    if [ ! -x "bin/blackbox_scan.sh" ]; then
        chmod +x bin/blackbox_scan.sh 2>/dev/null
    fi
    log_info "扫描目标: ${base_url}"
    log_info "隐藏登录路径: ${login_path}"
    bin/blackbox_scan.sh "${base_url}" "${login_path}"
}

# ─── 选项 9: 本地黑盒启动并扫描 ───────────────────────────────────────────────
blackbox_local_scan() {
    local login_path="${1:-${NACP_BLACKBOX_LOGIN_PATH:-/client-login}}"
    ensure_web_dist
    stop_local_backend_processes
    load_local_env_defaults
    export_blackbox_dev_env
    export NACP_BLACKBOX_LOGIN_PATH="${login_path}"
    build_local_backend
    log_info "启动本地黑盒后端用于扫描..."
    log_info "后端: :${LOCAL_BACKEND_PORT} | 隐藏登录路径: ${NACP_BLACKBOX_LOGIN_PATH}"
    log_info "数据库 DSN: $(mask_dsn "$SQL_DSN")"
    : > "${LOCAL_BACKEND_LOG}"
    "${LOCAL_BACKEND_BIN}" > "${LOCAL_BACKEND_LOG}" 2>&1 &
    GO_PID=$!
    if ! wait_backend_ready 120 "$GO_PID"; then
        kill "$GO_PID" 2>/dev/null
        exit 1
    fi
    bin/blackbox_scan.sh "http://localhost:${LOCAL_BACKEND_PORT}" "${NACP_BLACKBOX_LOGIN_PATH}"
    SCAN_STATUS=$?
    kill "$GO_PID" 2>/dev/null
    exit "$SCAN_STATUS"
}

# ─── 主逻辑 ───────────────────────────────────────────────────────────────────
main() {
    local choice="${1:-}"

    if [ -z "$choice" ]; then
        show_menu
        read -p "" choice
        choice="${choice:-0}"
    fi

    case "$choice" in
        0) local_dev "${2:-}" ;;
        1) deploy ;;
        2) server_status ;;
        3) server_logs ;;
        4) server_update ;;
        5) server_control ;;
        6) emergency_deploy ;;
        7) run_tests ;;
        8) blackbox_scan "${2:-}" "${3:-}" ;;
        9) blackbox_local_scan "${2:-}" ;;
        *)
            log_error "未知选项: $choice"
            show_menu
            exit 1
            ;;
    esac
}

main "$@"
