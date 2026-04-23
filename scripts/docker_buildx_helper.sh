#!/usr/bin/env bash
set -euo pipefail

buildx_available() {
  docker buildx version >/dev/null 2>&1
}

retry_command() {
  local attempts="${1:-3}"
  local delay_seconds="${2:-2}"
  shift 2

  local attempt=1
  while true; do
    if "$@"; then
      return 0
    fi

    if (( attempt >= attempts )); then
      return 1
    fi

    sleep "$delay_seconds"
    attempt=$((attempt + 1))
  done
}

docker_http_probe() {
  local name="$1"
  local url="$2"
  local expected_codes="$3"
  local http_code
  local attempts="${DOCKER_NET_RETRIES:-3}"
  local delay_seconds="${DOCKER_NET_RETRY_DELAY:-2}"

  if ! command -v curl >/dev/null 2>&1; then
    return 0
  fi

  if ! http_code="$(
    retry_command "$attempts" "$delay_seconds" \
      curl -sS -L -o /dev/null -w '%{http_code}' --connect-timeout 5 --max-time 15 "$url"
  )"; then
    echo "Failed to reach $name: $url" >&2
    return 1
  fi

  case ",$expected_codes," in
    *",$http_code,"*)
      return 0
      ;;
  esac

  echo "Unexpected response from $name: HTTP $http_code ($url)" >&2
  return 1
}

dockerfile_base_images() {
  local dockerfile="$1"

  awk '
    BEGIN { IGNORECASE = 1 }
    /^FROM[[:space:]]+/ {
      for (i = 2; i <= NF; i++) {
        if ($i ~ /^--/) {
          continue
        }
        print $i
        break
      }
    }
  ' "$dockerfile" | sort -u
}

check_docker_hub_connectivity() {
  docker_http_probe \
    "Docker Hub auth service" \
    "https://auth.docker.io/token?service=registry.docker.io&scope=repository:library%2Fnginx:pull" \
    "200" || return 1
  docker_http_probe \
    "Docker Hub registry" \
    "https://registry-1.docker.io/v2/" \
    "200,401" || return 1
}

check_buildx_registry_access_from_dockerfile() {
  local dockerfile="$1"
  local image_ref
  local failed=()
  local attempts="${DOCKER_NET_RETRIES:-3}"
  local delay_seconds="${DOCKER_NET_RETRY_DELAY:-2}"

  if ! buildx_available; then
    return 0
  fi

  while IFS= read -r image_ref; do
    if [[ -z "$image_ref" || "$image_ref" == *'$'* ]]; then
      continue
    fi
    if ! retry_command "$attempts" "$delay_seconds" docker buildx imagetools inspect "$image_ref" >/dev/null 2>&1; then
      failed+=("$image_ref")
    fi
  done < <(dockerfile_base_images "$dockerfile")

  if (( ${#failed[@]} == 0 )); then
    return 0
  fi

  echo "Docker buildx could not resolve required base image metadata:" >&2
  printf '  - %s\n' "${failed[@]}" >&2
  echo "This usually means the active builder cannot reach auth.docker.io or registry-1.docker.io." >&2
  echo "Try: docker login, restart Docker Desktop, recreate the buildx builder, or check proxy/VPN settings." >&2
  return 1
}

current_buildx_driver() {
  docker buildx inspect 2>/dev/null | awk -F': +' '/^Driver:/ {print $2; exit}'
}

builder_driver() {
  local builder_name="$1"
  docker buildx inspect "$builder_name" 2>/dev/null | awk -F': +' '/^Driver:/ {print $2; exit}'
}

builder_supports_platforms() {
  local builder_name="$1"
  local platforms_csv="$2"
  local available
  local platform

  available="$(docker buildx inspect "$builder_name" 2>/dev/null | awk -F': +' '/^Platforms:/ {print $2; exit}')"
  if [[ -z "$available" ]]; then
    return 1
  fi
  available="${available//[[:space:]]/}"

  IFS=',' read -r -a requested_platforms <<< "$platforms_csv"
  for platform in "${requested_platforms[@]}"; do
    platform="${platform//[[:space:]]/}"
    if [[ -z "$platform" ]]; then
      continue
    fi
    case ",$available," in
      *",$platform,"*)
        ;;
      *)
        return 1
        ;;
    esac
  done

  return 0
}

builder_exists() {
  local builder_name="$1"
  docker buildx inspect "$builder_name" >/dev/null 2>&1
}

ensure_container_buildx_builder() {
  local requested_platforms="${1:-linux/amd64,linux/arm64}"
  local builder_name="${2:-nginxpulse-container-builder}"
  local driver
  local current_name
  local candidate

  if ! buildx_available; then
    echo "Docker buildx is required but not available." >&2
    return 1
  fi

  driver="$(current_buildx_driver || true)"
  current_name="$(docker buildx inspect 2>/dev/null | awk -F': +' '/^Name:/ {print $2; exit}')"
  if [[ -n "$current_name" && ( "$driver" == "docker" || "$driver" == "docker-container" ) ]] && builder_supports_platforms "$current_name" "$requested_platforms"; then
    docker buildx inspect --bootstrap >/dev/null
    printf '%s\n' "$current_name"
    return 0
  fi

  for candidate in desktop-linux default; do
    if builder_exists "$candidate" && [[ "$(builder_driver "$candidate")" == "docker" ]] && builder_supports_platforms "$candidate" "$requested_platforms"; then
      if ! docker buildx use "$candidate" >/dev/null 2>&1; then
        continue
      fi
      docker buildx inspect --bootstrap >/dev/null
      printf '%s\n' "$candidate"
      return 0
    fi
  done

  if [[ "$driver" == "docker" ]]; then
    docker buildx inspect --bootstrap >/dev/null
    docker buildx inspect --bootstrap | awk -F': +' '/^Name:/ {print $2; exit}'
    return 0
  fi

  if [[ "$driver" == "docker-container" ]]; then
    docker buildx inspect --bootstrap >/dev/null
    docker buildx inspect --bootstrap | awk -F': +' '/^Name:/ {print $2; exit}'
    return 0
  fi

  if builder_exists "$builder_name"; then
    docker buildx use "$builder_name" >/dev/null
  else
    docker buildx create --name "$builder_name" --driver docker-container --use >/dev/null
  fi

  docker buildx inspect --bootstrap >/dev/null
  printf '%s\n' "$builder_name"
}
