#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WEBAPP_DIR="$ROOT_DIR/webapp"
WEBAPP_MOBILE_DIR="$ROOT_DIR/webapp_mobile"
WEBUI_DIST_DIR="$ROOT_DIR/internal/webui/dist"
WEBUI_DIST_MOBILE_DIR="$ROOT_DIR/internal/webui/dist_mobile"
BIN_DIR="$ROOT_DIR/bin"
CONFIG_SRC="$ROOT_DIR/configs/nginxpulse_config.json"
GZ_LOG_SRC="$ROOT_DIR/var/log/gz-log-read-test"
VERSION="${VERSION:-$(git -C "$ROOT_DIR" describe --tags --abbrev=0 2>/dev/null || echo "dev")}"
BUILD_TIME="${BUILD_TIME:-$(date "+%Y-%m-%d %H:%M:%S")}"
GIT_COMMIT="${GIT_COMMIT:-$(git -C "$ROOT_DIR" rev-parse --short=7 HEAD 2>/dev/null || echo "unknown")}"
LDFLAGS="-s -w -X 'github.com/qianlima-666/nginxpulse/internal/version.Version=${VERSION}' -X 'github.com/qianlima-666/nginxpulse/internal/version.BuildTime=${BUILD_TIME}' -X 'github.com/qianlima-666/nginxpulse/internal/version.GitCommit=${GIT_COMMIT}'"
USER_GOOS="${GOOS:-}"
USER_GOARCH="${GOARCH:-}"
PLATFORMS="${PLATFORMS:-}"
CGO_ENABLED="${CGO_ENABLED:-0}"

if [[ -z "$PLATFORMS" ]]; then
  if [[ -n "$USER_GOOS" || -n "$USER_GOARCH" ]]; then
    GOOS="${USER_GOOS:-$(go env GOOS)}"
    GOARCH="${USER_GOARCH:-$(go env GOARCH)}"
    PLATFORMS="${GOOS}/${GOARCH}"
  else
    PLATFORMS="linux/amd64,linux/arm64"
  fi
fi

if [[ ! -d "$WEBAPP_DIR" ]]; then
  echo "webapp directory not found." >&2
  exit 1
fi

if [[ ! -f "$WEBAPP_DIR/package.json" ]]; then
  echo "webapp/package.json not found." >&2
  exit 1
fi

if [[ ! -d "$WEBAPP_MOBILE_DIR" ]]; then
  echo "webapp_mobile directory not found." >&2
  exit 1
fi

if [[ ! -f "$WEBAPP_MOBILE_DIR/package.json" ]]; then
  echo "webapp_mobile/package.json not found." >&2
  exit 1
fi

if [[ ! -f "$CONFIG_SRC" ]]; then
  echo "Missing config file: $CONFIG_SRC" >&2
  exit 1
fi

if [[ ! -d "$GZ_LOG_SRC" ]]; then
  echo "Gzip sample folder not found: $GZ_LOG_SRC" >&2
  exit 1
fi

copy_support_files() {
  local target_dir="$1"
  local tmp_config

  mkdir -p "$target_dir/configs"
  cp "$CONFIG_SRC" "$target_dir/configs/nginxpulse_config.json"
  tmp_config="$(mktemp)"
  awk '
    BEGIN { in_server=0; updated=0 }
    {
      if ($0 ~ /"server"[[:space:]]*:/) { in_server=1 }
      if (in_server && $0 ~ /"Port"[[:space:]]*:/) {
        sub(/"Port"[[:space:]]*:[[:space:]]*"[^"]*"/, "\"Port\": \":8088\"")
        updated=1
      }
      if (in_server && $0 ~ /}/) { in_server=0 }
      print
    }
    END {
      if (!updated) {
        exit 1
      }
    }
  ' "$target_dir/configs/nginxpulse_config.json" > "$tmp_config"
  if [[ $? -ne 0 ]]; then
    rm -f "$tmp_config"
    echo "Failed to update server port in config for ${target_dir}." >&2
    exit 1
  fi
  mv "$tmp_config" "$target_dir/configs/nginxpulse_config.json"

  mkdir -p "$target_dir/var/log"
  rm -rf "$target_dir/var/log/gz-log-read-test"
  cp -R "$GZ_LOG_SRC" "$target_dir/var/log/"
}

echo "Building frontend..."
(cd "$WEBAPP_DIR" && pnpm install --frozen-lockfile && pnpm run build)

echo "Building mobile frontend..."
(cd "$WEBAPP_MOBILE_DIR" && pnpm install --frozen-lockfile && pnpm run build)

echo "Preparing embedded assets..."
rm -rf "$WEBUI_DIST_DIR"
mkdir -p "$WEBUI_DIST_DIR"
cp -R "$WEBAPP_DIR/dist/." "$WEBUI_DIST_DIR/"

rm -rf "$WEBUI_DIST_MOBILE_DIR"
mkdir -p "$WEBUI_DIST_MOBILE_DIR"
cp -R "$WEBAPP_MOBILE_DIR/dist/." "$WEBUI_DIST_MOBILE_DIR/"

echo "Target platforms: ${PLATFORMS}"
IFS=',' read -r -a platform_list <<< "$PLATFORMS"
platform_count="${#platform_list[@]}"
built_targets=()

for platform in "${platform_list[@]}"; do
  platform="${platform//[[:space:]]/}"
  if [[ -z "$platform" || "$platform" != */* ]]; then
    echo "Invalid platform: ${platform}" >&2
    exit 1
  fi
  target_os="${platform%%/*}"
  target_arch="${platform##*/}"
  if [[ -z "$target_os" || -z "$target_arch" ]]; then
    echo "Invalid platform: ${platform}" >&2
    exit 1
  fi

  out_dir="$BIN_DIR"
  if [[ "$platform_count" -gt 1 ]]; then
    out_dir="$BIN_DIR/${target_os}_${target_arch}"
  fi
  mkdir -p "$out_dir"

  bin_name="nginxpulse"
  if [[ "$target_os" == "windows" ]]; then
    bin_name="nginxpulse.exe"
  fi

  echo "Building single binary (version: ${VERSION}, target: ${target_os}/${target_arch})..."
  (cd "$ROOT_DIR" && CGO_ENABLED="$CGO_ENABLED" GOOS="$target_os" GOARCH="$target_arch" \
    go build -tags embed -ldflags="${LDFLAGS}" -o "${out_dir}/${bin_name}" ./cmd/nginxpulse/main.go)

  echo "Copying default config and gz samples to ${out_dir}..."
  copy_support_files "$out_dir"
  built_targets+=("${out_dir}/${bin_name}")
done

if [[ "$platform_count" -eq 1 ]]; then
  echo "Done: bin/nginxpulse"
else
  echo "Done:"
  for target in "${built_targets[@]}"; do
    echo " - ${target}"
  done
fi
