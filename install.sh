#!/usr/bin/env bash
# 自动下载 GitHub Releases 中对应架构的预编译包，无需 Go / Node.js。
# curl -fL https://github.com/kingchun/gost-panel/releases/latest/download/install.sh -o install.sh
# sudo bash install.sh --host 0.0.0.0
set -euo pipefail
umask 077
die() { printf '[ERROR] %s\n' "$*" >&2; exit 1; }
info() { printf '[INFO] %s\n' "$*"; }
usage() {
  cat <<'HELP'
用法：bash install.sh [选项] [部署参数]
  默认                  自动识别 linux/amd64、linux/arm64，下载最新正式版并校验 SHA256
  --version v0.1.0      下载指定版本
  --download-only DIR   仅下载并校验发布包到 DIR，不安装服务（无需 root）
  --archive FILE        使用本地 tar.gz 发布包
  --package-dir DIR     使用已解包目录
  --source              从源码构建（需要 Go；无前端产物时还需要 npm）
  --host ADDRESS        面板监听地址，默认 127.0.0.1
  --port PORT           面板端口，默认 19090
  --dir DIR             安装目录，默认 /opt/gost-panel
  --user USER           首次安装的用户名，默认 admin
  --panel-pass PASS     首次安装的密码，默认随机生成
  --gost-api-pass PASS  首次安装的 gost API 密码，默认随机生成
HELP
}
SCRIPT_DIR=''
if [[ -n "${BASH_SOURCE[0]:-}" && -f "${BASH_SOURCE[0]}" ]]; then
  SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
fi
REPO=kingchun/gost-panel
version=latest; archive=''; package_dir=''; source_mode=0; download_dir=''; deploy_args=()
while (($#)); do
  case "$1" in
    --archive|--package-dir|--version|--download-only)
      (($# >= 2)) || die "$1 缺少参数值"
      [[ -n "$2" ]] || die "$1 参数值不能为空"
      case "$1" in
        --archive) archive="$2" ;;
        --package-dir) package_dir="$2" ;;
        --version) version="$2" ;;
        --download-only) download_dir="$2" ;;
      esac
      shift 2 ;;
    --source) source_mode=1; shift ;;
    --install-dir|--dir|--panel-port|--port|--panel-host|--host|--user|--panel-pass|--gost-api-pass)
      (($# >= 2)) || die "$1 缺少参数值"
      deploy_args+=("$1" "$2"); shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) die "未知参数: $1" ;;
  esac
done
[[ -n "$download_dir" || $EUID == 0 ]] || die '请使用 root 运行，或使用 --download-only 仅下载'
[[ "$(uname -s)" == Linux ]] || die '仅支持 Linux'
modes=$source_mode
[[ -z "$archive" ]] || modes=$((modes + 1))
[[ -z "$package_dir" ]] || modes=$((modes + 1))
((modes <= 1)) || die '--source、--archive、--package-dir 不能同时使用'
[[ -z "$download_dir" || $modes == 0 ]] || die '--download-only 只能用于在线下载发布包'

ensure_tools() {
  local cmd missing=()
  for cmd in "$@"; do command -v "$cmd" >/dev/null || missing+=("$cmd"); done
  ((${#missing[@]})) || return 0
  [[ $EUID == 0 ]] || die "缺少依赖: ${missing[*]}，请先安装"
  if command -v apt-get >/dev/null; then
    DEBIAN_FRONTEND=noninteractive apt-get update -qq
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq curl ca-certificates python3 coreutils
  elif command -v dnf >/dev/null; then
    dnf install -y curl ca-certificates python3 coreutils
  else
    die "缺少依赖: ${missing[*]}，请先安装 curl、python3、coreutils"
  fi
}
resolve_version() {
  if [[ "$version" == latest ]]; then
    curl -fLsS --retry 3 --connect-timeout 20 --max-time 60 \
      "https://api.github.com/repos/$REPO/releases/latest" -o "$work_dir/release.json"
    version="$(python3 -c 'import json, sys; print(json.load(open(sys.argv[1]))["tag_name"])' "$work_dir/release.json")"
  fi
  [[ "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][a-zA-Z0-9.-]+)?$ ]] || die '版本格式应为 v0.1.0'
}
work_dir="$(mktemp -d /tmp/gost-panel-setup.XXXXXX)"
trap 'rm -rf -- "$work_dir"' EXIT

if ((modes == 0)); then
  case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) die '仅支持 x86_64/amd64 和 aarch64/arm64' ;;
  esac
  ensure_tools curl python3 sha256sum
  resolve_version
  asset="gost-panel-$version-linux-$arch.tar.gz"
  base="https://github.com/$REPO/releases/download/$version"
  info "下载 $version / linux/$arch"
  curl -fLsS --retry 3 --connect-timeout 20 --max-time 600 "$base/$asset" -o "$work_dir/$asset"
  curl -fLsS --retry 3 --connect-timeout 20 --max-time 60 "$base/SHA256SUMS.txt" -o "$work_dir/SHA256SUMS.txt"
  awk -v name="$asset" '$2 == name { print; found++ } END { if (found != 1) exit 1 }' \
    "$work_dir/SHA256SUMS.txt" > "$work_dir/selected.sha256" || die '发布清单中未找到唯一匹配的校验和'
  (cd "$work_dir" && sha256sum -c selected.sha256) || die '发布包 SHA256 校验失败，已停止安装'
  archive="$work_dir/$asset"
fi

if [[ -n "$archive" ]]; then
  [[ -f "$archive" ]] || die "发布包不存在: $archive"
  # 发布包必须只包含一个顶层目录，拒绝路径穿越、绝对路径和符号链接。
  ensure_tools python3
  python3 - "$archive" "$work_dir" <<'PY'
import pathlib, sys, tarfile
with tarfile.open(sys.argv[1], 'r:gz') as archive:
    members = archive.getmembers()
    if not members:
        raise SystemExit('发布包为空')
    for entry in members:
        path = pathlib.PurePosixPath(entry.name)
        if not path.parts or path.is_absolute() or '..' in path.parts or not (entry.isdir() or entry.isfile()):
            raise SystemExit('不安全的发布包路径或文件类型')
    roots = {pathlib.PurePosixPath(entry.name).parts[0] for entry in members}
    if len(roots) != 1:
        raise SystemExit('发布包必须只含一个顶层目录')
    archive.extractall(sys.argv[2], **({'filter': 'data'} if hasattr(tarfile, 'data_filter') else {}))
PY
  package_dir="$(find "$work_dir" -mindepth 1 -maxdepth 1 -type d -print -quit)"
elif ((source_mode)); then
  source_dir="$SCRIPT_DIR"
  if [[ ! -f "$source_dir/backend/go.mod" ]]; then
    command -v git >/dev/null || die '请先安装 git，或上传发布包并使用 --archive'
    source_dir="$work_dir/source"
    ensure_tools curl python3
    resolve_version
    git clone --depth 1 --branch "$version" "https://github.com/$REPO.git" "$source_dir"
  fi
  [[ -f "$source_dir/backend/go.mod" ]] || die '仓库缺少 Go 后端源码；请发布完整代码或使用本地构建的 --archive 发布包'
  command -v go >/dev/null || die '源码模式需要 Go；服务器无需 Go 的安装方式：--archive 发布包'
  package_dir="$work_dir/package"
  mkdir -p "$package_dir"
  printf '[INFO] 构建 Go 后端\n'
  (cd "$source_dir/backend" && CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X gost-panel/internal/api.Version=${PANEL_VERSION:-v0.1.0}" \
    -o "$package_dir/gost-panel" ./cmd/gost-panel)
  if [[ ! -f "$source_dir/static/index.html" ]]; then
    command -v npm >/dev/null || die '没有前端产物，请先在本地 npm ci && npm run build，或上传完整发布包'
    (cd "$source_dir/web" && npm ci --no-audit --no-fund && npm run build)
  fi
  cp -R "$source_dir/static" "$package_dir/static"
  cp "$source_dir/scripts/release/deploy.sh" "$package_dir/deploy.sh"
fi
[[ -f "$package_dir/deploy.sh" && -f "$package_dir/gost-panel" && -f "$package_dir/static/index.html" ]] || die '发布包缺少 deploy.sh、后端二进制或前端产物'
if [[ -n "$download_dir" ]]; then
  mkdir -p -- "$download_dir"
  cp -- "$archive" "$download_dir/$asset"
  cp -- "$work_dir/selected.sha256" "$download_dir/SHA256SUMS.txt"
  info "发布包已下载并校验：$download_dir/$asset"
  exit 0
fi
bash "$package_dir/deploy.sh" "${deploy_args[@]}"
