#!/bin/bash
# ─────────────────────────────────────────────────────────────────────────────
# NACP gogogo.sh — 统一入口脚本
# 用法: ./gogogo.sh [选项]
# ─────────────────────────────────────────────────────────────────────────────

set -e

# ─── 配置 ─────────────────────────────────────────────────────────────────────
REGISTRY="ghcr.io"
IMAGE="ghcr.io/al90slj23/nacp"
COMPOSE_FILE="docker-compose.yml"
CONTAINER_NAME="nacp"

# ─── 颜色 ─────────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# ─── 菜单 ─────────────────────────────────────────────────────────────────────
show_menu() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════${NC}"
    echo -e "${BLUE}  NACP — NewAPI Classic Plus${NC}"
    echo -e "${BLUE}═══════════════════════════════════════${NC}"
    echo ""
    echo "  1) 部署/更新（拉取最新镜像并重启）"
    echo "  2) 查看状态"
    echo "  3) 查看日志"
    echo "  4) 停止服务"
    echo "  5) 重启服务"
    echo "  6) 本地构建并推送"
    echo ""
    echo -e "  ${YELLOW}输入选项编号:${NC}"
}

# ─── 选项 1: 部署/更新 ────────────────────────────────────────────────────────
deploy() {
    log_info "拉取最新镜像..."
    docker pull "${IMAGE}:main"

    if [ -f "$COMPOSE_FILE" ]; then
        log_info "使用 docker compose 部署..."
        docker compose -f "$COMPOSE_FILE" up -d
    else
        log_warn "未找到 ${COMPOSE_FILE}，使用 docker run..."
        docker stop "$CONTAINER_NAME" 2>/dev/null || true
        docker rm "$CONTAINER_NAME" 2>/dev/null || true
        docker run -d \
            --name "$CONTAINER_NAME" \
            --restart unless-stopped \
            -p 3000:3000 \
            -v ./data:/data \
            --env-file .env \
            "${IMAGE}:main"
    fi

    log_info "部署完成！"
    docker ps | grep "$CONTAINER_NAME"
}

# ─── 选项 2: 查看状态 ─────────────────────────────────────────────────────────
status() {
    log_info "容器状态:"
    docker ps -a | grep "$CONTAINER_NAME" || log_warn "未找到容器"
    echo ""
    log_info "镜像信息:"
    docker images | grep "nacp" || log_warn "未找到镜像"
}

# ─── 选项 3: 查看日志 ─────────────────────────────────────────────────────────
logs() {
    log_info "最近 100 行日志:"
    docker logs --tail 100 -f "$CONTAINER_NAME"
}

# ─── 选项 4: 停止服务 ─────────────────────────────────────────────────────────
stop() {
    log_info "停止服务..."
    if [ -f "$COMPOSE_FILE" ]; then
        docker compose -f "$COMPOSE_FILE" down
    else
        docker stop "$CONTAINER_NAME" 2>/dev/null || true
    fi
    log_info "已停止"
}

# ─── 选项 5: 重启服务 ─────────────────────────────────────────────────────────
restart() {
    log_info "重启服务..."
    if [ -f "$COMPOSE_FILE" ]; then
        docker compose -f "$COMPOSE_FILE" restart
    else
        docker restart "$CONTAINER_NAME"
    fi
    log_info "已重启"
}

# ─── 选项 6: 本地构建并推送 ───────────────────────────────────────────────────
build_and_push() {
    log_info "本地构建 Docker 镜像..."
    docker build -t "${IMAGE}:main" .
    log_info "推送到 GHCR..."
    docker push "${IMAGE}:main"
    log_info "构建并推送完成！"
}

# ─── 主逻辑 ───────────────────────────────────────────────────────────────────
main() {
    local choice="${1:-}"

    if [ -z "$choice" ]; then
        show_menu
        read -r choice
    fi

    case "$choice" in
        1) deploy ;;
        2) status ;;
        3) logs ;;
        4) stop ;;
        5) restart ;;
        6) build_and_push ;;
        *)
            log_error "未知选项: $choice"
            show_menu
            exit 1
            ;;
    esac
}

main "$@"
