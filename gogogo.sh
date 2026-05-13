#!/bin/bash
# ─────────────────────────────────────────────────────────────────────────────
# NACP gogogo.sh — 统一入口脚本
# 用法: ./gogogo.sh [选项]
#
# 部署流程：
#   默认（选项1）: git push → GitHub Actions 构建镜像 → 服务器 pull + restart
#   紧急（选项6）: 本地 docker build → push 到 GHCR → 服务器 pull + restart
# ─────────────────────────────────────────────────────────────────────────────

set -e

# ─── 配置 ─────────────────────────────────────────────────────────────────────
IMAGE="ghcr.io/al90slj23/nacp:main"
DEPLOY_SERVER="143.198.87.200"
DEPLOY_USER="root"
DEPLOY_DIR="/opt/nacp"
COMPOSE_FILE="docker-compose.yml"
CONTAINER_NAME="nacp"

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
SSH_KEY="~/.ssh/al90slj23"

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
    echo "  0) 本地开发环境"
    echo ""
    echo -e "  ${YELLOW}输入选项编号:${NC}"
}

# ─── 选项 0: 本地开发环境 ─────────────────────────────────────────────────────
local_dev() {
    echo "  a) 启动后端 (go run)"
    echo "  b) 启动前端 (bun run dev)"
    echo "  c) 同时启动后端+前端"
    echo "  d) 停止本地开发"
    read -p "选择: " sub_choice
    case "$sub_choice" in
        a)
            log_info "启动后端 (端口 3000)..."
            log_info "需要本地 MySQL 或设置 SQL_DSN 环境变量"
            log_info "按 Ctrl+C 停止"
            export SQL_DSN="${SQL_DSN:-root:@tcp(localhost:3306)/nacp?charset=utf8mb4&parseTime=True&loc=Local}"
            export SESSION_SECRET="${SESSION_SECRET:-local_dev_secret}"
            export MEMORY_CACHE_ENABLED="${MEMORY_CACHE_ENABLED:-true}"
            export ERROR_LOG_ENABLED="${ERROR_LOG_ENABLED:-true}"
            go run main.go
            ;;
        b)
            log_info "启动前端开发服务器 (端口 5173)..."
            log_info "按 Ctrl+C 停止"
            (cd web && bun run dev)
            ;;
        c)
            log_info "同时启动后端+前端..."
            log_info "后端: :3000  前端: :5173"
            log_info "按 Ctrl+C 停止"
            export SQL_DSN="${SQL_DSN:-root:@tcp(localhost:3306)/nacp?charset=utf8mb4&parseTime=True&loc=Local}"
            export SESSION_SECRET="${SESSION_SECRET:-local_dev_secret}"
            export MEMORY_CACHE_ENABLED="${MEMORY_CACHE_ENABLED:-true}"
            export ERROR_LOG_ENABLED="${ERROR_LOG_ENABLED:-true}"
            # Start backend in background
            go run main.go &
            GO_PID=$!
            # Start frontend
            (cd web && bun run dev) &
            BUN_PID=$!
            # Wait for either to exit
            trap "kill $GO_PID $BUN_PID 2>/dev/null; exit" INT TERM
            wait
            ;;
        d)
            log_info "停止本地开发进程..."
            pkill -f "go run main.go" 2>/dev/null && log_info "后端已停止" || log_warn "后端未运行"
            pkill -f "bun run dev" 2>/dev/null && log_info "前端已停止" || log_warn "前端未运行"
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

# ─── 主逻辑 ───────────────────────────────────────────────────────────────────
main() {
    local choice="${1:-}"

    if [ -z "$choice" ]; then
        show_menu
        read -r choice
    fi

    case "$choice" in
        0) local_dev ;;
        1) deploy ;;
        2) server_status ;;
        3) server_logs ;;
        4) server_update ;;
        5) server_control ;;
        6) emergency_deploy ;;
        7) run_tests ;;
        *)
            log_error "未知选项: $choice"
            show_menu
            exit 1
            ;;
    esac
}

main "$@"
