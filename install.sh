#!/usr/bin/env bash
# 统一安装入口：预构建包优先，源码模式用于开发机/构建机。
# bash install.sh --archive /root/gost-panel-v0.1.0-linux-amd64.tar.gz --host 0.0.0.0
# bash install.sh --package-dir /root/gost-panel-v0.1.0-linux-amd64
# bash install.sh（在完整源码目录执行，需 Go 与 npm）
set -euo pipefail
die() { printf '[ERROR] %s\n' "$*" >&2; exit 1; }
SCRIPT_DIR=''
if [[ -n "${BASH_SOURCE[0]:-}" && -f "${BASH_SOURCE[0]}" ]]; then
  SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
fi
archive=''; package_dir=''; deploy_args=()
while (($#)); do
  case "$1" in
    --archive|--package-dir)
      (($# >= 2)) || die "$1 缺少参数值"
      if [[ "$1" == --archive ]]; then archive="$2"; else package_dir="$2"; fi
      shift 2 ;;
    --install-dir|--dir|--panel-port|--port|--panel-host|--host|--user|--panel-pass|--gost-api-pass)
      (($# >= 2)) || die "$1 缺少参数值"
      deploy_args+=("$1" "$2"); shift 2 ;;
    -h|--help) sed -n '2,5p' "$0"; exit 0 ;;
    *) die "未知参数: $1" ;;
  esac
done
[[ $EUID == 0 ]] || die '请使用 root 运行'
[[ -z "$archive" || -z "$package_dir" ]] || die '--archive 与 --package-dir 不能同时使用'
work_dir="$(mktemp -d /tmp/gost-panel-setup.XXXXXX)"
trap 'rm -rf -- "$work_dir"' EXIT

if [[ -n "$archive" ]]; then
  [[ -f "$archive" ]] || die "发布包不存在: $archive"
  # 发布包必须只包含一个顶层目录，拒绝路径穿越、绝对路径和符号链接。
  command -v python3 >/dev/null || die '解包需要 python3'
  python3 - "$archive" "$work_dir" <<'PY'
import pathlib, sys, tarfile
with tarfile.open(sys.argv[1], 'r:gz') as archive:
    members = archive.getmembers()
    for entry in members:
        path = pathlib.PurePosixPath(entry.name)
        if path.is_absolute() or '..' in path.parts or not (entry.isdir() or entry.isfile()):
            raise SystemExit('不安全的发布包路径或文件类型')
    roots = {pathlib.PurePosixPath(entry.name).parts[0] for entry in members}
    if len(roots) != 1:
        raise SystemExit('发布包必须只含一个顶层目录')
    archive.extractall(sys.argv[2], **({'filter': 'data'} if hasattr(tarfile, 'data_filter') else {}))
PY
  package_dir="$(find "$work_dir" -mindepth 1 -maxdepth 1 -type d -print -quit)"
elif [[ -z "$package_dir" ]]; then
  source_dir="$SCRIPT_DIR"
  if [[ ! -f "$source_dir/backend/go.mod" ]]; then
    command -v git >/dev/null || die '请先安装 git，或上传发布包并使用 --archive'
    source_dir="$work_dir/source"
    git clone --depth 1 https://github.com/kingchun/gost-panel.git "$source_dir"
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
[[ -f "$package_dir/deploy.sh" ]] || die '发布包缺少 deploy.sh'
bash "$package_dir/deploy.sh" "${deploy_args[@]}"
