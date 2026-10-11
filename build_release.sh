#!/bin/bash

#==============================================
# Mayfly-Go Release Build Tool
# 前后端打包编译至指定目录，快速制作发行版
#==============================================

set -e  # 遇到错误立即退出

#----------------------------------------------
# 全局配置
#----------------------------------------------
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_DIR="${PROJECT_ROOT}/server"
FRONTEND_DIR="${PROJECT_ROOT}/frontend"
BINARY_NAME="mayfly-go"

# 构建目标配置：名称|操作系统|架构
BUILD_TARGETS=(
    "linux-amd64|linux|amd64"
    "linux-arm64|linux|arm64"
    "windows|windows|amd64"
    "mac|darwin|amd64"
)

#----------------------------------------------
# 工具函数
#----------------------------------------------
print_header() {
    echo -e "\033[1;33m$1\033[0m"
}

print_success() {
    echo -e "\033[1;32m$1\033[0m"
}

print_error() {
    echo -e "\033[1;31m$1\033[0m" >&2
}

print_info() {
    echo -e "\033[1;34m$1\033[0m"
}

to_lower() {
    echo "$1" | tr '[:upper:]' '[:lower:]'
}

#----------------------------------------------
# 推送前置检查
#----------------------------------------------

# 从 docker config.json 读取字符串字段（无 jq 依赖），字段不存在时输出空串
read_docker_config_str() {
    local config_file="$1"
    local key="$2"

    [ -f "${config_file}" ] || return 0

    grep -o "\"${key}\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" "${config_file}" \
        | head -1 \
        | sed -E 's/.*:[[:space:]]*"([^"]*)"/\1/'
}

# 校验镜像所属 registry 本地是否已有登录凭据，避免多架构构建全部跑完后才因鉴权失败白费时间
# 返回 0=已找到凭据记录；1=未发现任何凭据记录
ensure_registry_login() {
    local image_ref="$1"
    local registry="${image_ref%%/*}"
    local config_file="${DOCKER_CONFIG:-${HOME}/.docker}/config.json"
    local creds_store helper
    local registry_key_pattern="\"(https?://)?${registry}/?\""

    # 凭据助手（Docker Desktop 默认 credsStore=desktop），list 输出只含用户名不含密码
    creds_store="$(read_docker_config_str "${config_file}" credsStore)"
    if [ -n "${creds_store}" ]; then
        helper="docker-credential-${creds_store}"
        if command -v "${helper}" >/dev/null 2>&1; then
            if "${helper}" list 2>/dev/null | grep -Eq "${registry_key_pattern}"; then
                print_info "    Registry login found: ${registry}"
                return 0
            fi
        fi
    fi

    # 未启用凭据助手时，登录记录会内联在 config.json 的 auths 中
    if grep -Eq "${registry_key_pattern}" "${config_file}" 2>/dev/null; then
        print_info "    Registry login found: ${registry}"
        return 0
    fi

    print_error "Error: no login credentials found for registry '${registry}'."
    print_error "       Run first: docker login ${registry} -u <account UIN>"
    print_error "       The password is the one set in the container registry console"
    print_error "       (access credentials), NOT the API SecretId/SecretKey."
    return 1
}

#----------------------------------------------
# 构建函数
#----------------------------------------------
build_frontend() {
    print_header "\n>>> Building frontend..."
    
    cd "${FRONTEND_DIR}"
    npm run build
    
    # 拷贝到 server 静态目录
    print_success ">>> Copying frontend assets to server/static/static"
    rm -rf "${SERVER_DIR}/static/static"
    mkdir -p "${SERVER_DIR}/static/static"
    cp -r "${FRONTEND_DIR}/dist/"* "${SERVER_DIR}/static/static/"
    
    cd "${PROJECT_ROOT}"
}

build_backend() {
    local output_dir="$1"
    local os_name="$2"
    local arch="$3"
    local copy_resources="$4"
    
    local binary_file="${BINARY_NAME}"
    local target_name="${os_name}-${arch}"
    
    print_header "\n>>> Building backend: ${target_name}"
    
    # Windows 需要 .exe 后缀
    if [ "${os_name}" = "windows" ]; then
        binary_file="${BINARY_NAME}.exe"
    fi
    
    # 编译
    cd "${SERVER_DIR}"
    go mod tidy
    CGO_ENABLED=0 GOOS="${os_name}" GOARCH="${arch}" \
        go build -trimpath -ldflags="-w" -o "${binary_file}" main.go
    
    # 准备输出目录
    local bin_dir="${output_dir}/bin"
    if [ -d "${output_dir}" ]; then
        print_info "    Output directory exists, cleaning..."
        rm -rf "${output_dir}"
    fi
    mkdir -p "${bin_dir}"
    
    # 移动二进制文件到 bin 目录
    mv "${SERVER_DIR}/${binary_file}" "${bin_dir}/"
    
    # 拷贝资源文件
    if [ "${copy_resources}" = "1" ]; then
        print_info "    Copying config and scripts..."
        cp "${SERVER_DIR}/config.yml.example" "${output_dir}/config.yml"
        cp "${SERVER_DIR}/readme.txt" "${output_dir}/"
        cp "${SERVER_DIR}/readme_en.txt" "${output_dir}/"
        cp "${SERVER_DIR}/resources/script/mayfly-go.sh" "${output_dir}/"
        chmod +x "${output_dir}/mayfly-go.sh"
    fi
    
    print_success ">>> Build complete: ${target_name}"
    cd "${PROJECT_ROOT}"
}

build_docker() {
    local version="$1"
    local use_buildx="$2"
    local image_name
    local build_cmd
    
    if [ "${use_buildx}" = "1" ]; then
        image_name="ccr.ccs.tencentyun.com/mayfly/mayfly-go:${version}"
        # CCR 不兼容 buildx 默认附带的 OCI attestation manifest，关闭 provenance/sbom 后再推
        build_cmd="docker buildx build --no-cache --push --platform linux/amd64,linux/arm64 --provenance=false --sbom=false"
        print_header "\n>>> Building Docker image (multi-arch): ${image_name}"

        if ! ensure_registry_login "${image_name}"; then
            return 1
        fi
    else
        image_name="mayfly/mayfly-go:${version}"
        build_cmd="docker build --no-cache --platform linux/amd64"
        print_header "\n>>> Building Docker image: ${image_name}"
    fi
    
    ${build_cmd} --build-arg MAYFLY_GO_VERSION="${version}" -t "${image_name}" "${PROJECT_ROOT}"
    print_success ">>> Docker image built: ${image_name}"
}

cleanup_frontend() {
    print_info "\n>>> Cleaning up temporary frontend assets..."
    rm -rf "${SERVER_DIR}/static/static/"{assets,config.js,index.html}
    print_success ">>> Cleanup complete"
}

compress_package() {
    local source_dir="$1"
    local output_dir="$2"
    local package_name
    
    package_name="$(basename "${source_dir}")"
    
    print_header "\n>>> Compressing package: ${package_name}"
    
    cd "${output_dir}"
    
    # 统一使用 zip 格式，跨平台兼容性最好
    zip -r "${package_name}.zip" "${package_name}"/
    rm -rf "${package_name}"
    print_success ">>> Compressed: ${package_name}.zip"
    
    cd "${PROJECT_ROOT}"
}

#----------------------------------------------
# 主流程
#----------------------------------------------
main() {
    # 显示菜单
    print_header "========================================"
    print_header "   Mayfly-Go Release Build Tool"
    print_header "========================================"
    echo ""
    echo "Build Options:"
    echo "  [0] All Platforms (linux-amd64, linux-arm64, windows, mac)"
    echo "  [1] Linux AMD64"
    echo "  [2] Linux ARM64"
    echo "  [3] Windows"
    echo "  [4] macOS"
    echo "  [5] Docker Image"
    echo "  [6] Docker Multi-arch (buildx)"
    echo ""
    
    read -p "Select build option [0-6] (default: 0): " build_type
    build_type=${build_type:-0}
    
    # 验证输入
    if ! [[ "${build_type}" =~ ^[0-6]$ ]]; then
        print_error "Error: Invalid option. Please enter a number between 0 and 6."
        exit 1
    fi
    
    # 初始化配置
    local output_dir="."
    local docker_version="latest"
    local copy_resources="1"
    local compress_output="0"
    local is_docker=0
    
    # Docker 构建
    if [[ "${build_type}" == "5" || "${build_type}" == "6" ]]; then
        is_docker=1
        echo ""
        read -p "Enter Docker image version (default: latest): " docker_version
        docker_version=${docker_version:-latest}
    else
        # 二进制构建
        echo ""
        read -p "Enter output directory (default: current): " output_dir
        output_dir=${output_dir:-.}
        
        # 验证并获取绝对路径
        if [ "${output_dir}" != "." ] && [ ! -d "${output_dir}" ]; then
            print_error "Error: Directory '${output_dir}' does not exist."
            exit 1
        fi
        output_dir="$(cd "${output_dir}" && pwd)"
        
        echo ""
        read -p "Copy config & scripts? [Y/n] (default: Y): " copy_input
        if [ "$(to_lower "${copy_input}")" = "n" ]; then
            copy_resources="0"
        fi
        
        echo ""
        read -p "Compress package? [y/N] (default: N): " compress_input
        if [ "$(to_lower "${compress_input}")" = "y" ]; then
            compress_output="1"
        fi
        
        # 构建前端
        echo ""
        build_frontend
    fi
    
    # 显示配置摘要
    echo ""
    print_header "Build Configuration:"
    
    # 获取构建类型名称
    local type_names=("All Platforms" "Linux AMD64" "Linux ARM64" "Windows" "macOS" "Docker Image" "Docker Multi-arch")
    echo "  Type: ${type_names[${build_type}]}"
    
    if [ "${is_docker}" = "1" ]; then
        echo "  Version: ${docker_version}"
    else
        echo "  Output: ${output_dir}"
        echo "  Resources: $([ "${copy_resources}" = "1" ] && echo "Yes" || echo "No")"
        echo "  Compress: $([ "${compress_output}" = "1" ] && echo "Yes" || echo "No")"
    fi
    echo ""
    
    # 确认构建
    read -p "Continue? [Y/n] (default: Y): " confirm
    if [ "$(to_lower "${confirm}")" = "n" ]; then
        print_info "Build cancelled."
        exit 0
    fi
    
    # 执行构建
    echo ""
    print_header "Starting build..."
    
    case "${build_type}" in
        "1"|"2"|"3"|"4")
            # 单个平台构建
            local target="${BUILD_TARGETS[$((build_type-1))]}"
            IFS='|' read -r name os arch <<< "${target}"
            build_backend "${output_dir}/mayfly-go-${name}" "${os}" "${arch}" "${copy_resources}"
            ;;
        "5")
            build_docker "${docker_version}" "0"
            ;;
        "6")
            build_docker "${docker_version}" "1"
            ;;
        *)
            # 构建所有平台
            print_info "Building all platforms..."
            for target in "${BUILD_TARGETS[@]}"; do
                IFS='|' read -r name os arch <<< "${target}"
                build_backend "${output_dir}/mayfly-go-${name}" "${os}" "${arch}" "${copy_resources}"
            done
            ;;
    esac
    
    # 清理临时文件
    if [ "${is_docker}" = "0" ]; then
        cleanup_frontend
    fi
    
    # 压缩输出
    if [ "${compress_output}" = "1" ] && [ "${is_docker}" = "0" ]; then
        case "${build_type}" in
            "1"|"2"|"3"|"4")
                local target="${BUILD_TARGETS[$((build_type-1))]}"
                IFS='|' read -r name os arch <<< "${target}"
                compress_package "${output_dir}/mayfly-go-${name}" "${output_dir}"
                ;;
            *)
                print_info "Compressing all packages..."
                for target in "${BUILD_TARGETS[@]}"; do
                    IFS='|' read -r name os arch <<< "${target}"
                    compress_package "${output_dir}/mayfly-go-${name}" "${output_dir}"
                done
                ;;
        esac
    fi
    
    # 完成
    echo ""
    print_success "========================================"
    print_success "   Build Completed Successfully!"
    print_success "========================================"
}

# 执行主函数
main
