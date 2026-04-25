#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WIKI_DIR="$ROOT_DIR/docs/wiki"
FUMADOCS_DIR="$ROOT_DIR/docs/fumadocs"
TARGET_DIR="$FUMADOCS_DIR/content/docs"

if [[ ! -d "$WIKI_DIR" ]]; then
  echo "Wiki directory not found: $WIKI_DIR" >&2
  exit 1
fi

if [[ ! -d "$FUMADOCS_DIR" ]]; then
  echo "Fumadocs directory not found: $FUMADOCS_DIR" >&2
  exit 1
fi

mkdir -p "$TARGET_DIR"

# Keep index.mdx managed by Fumadocs, refresh markdown docs and sidebar metadata.
find "$TARGET_DIR" -maxdepth 1 -type f -name '*.md' -delete

escape_yaml_double_quoted() {
  printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'
}

escape_json_string() {
  printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'
}

extract_title() {
  local file="$1"
  local title=""

  title="$(sed -n 's/^# \{1,\}\(.*\)$/\1/p' "$file" | head -n 1 | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')"
  if [[ -z "$title" ]]; then
    local base
    base="$(basename "$file" .md)"
    title="$(printf '%s' "$base" | sed 's/-/ /g')"
  fi
  printf '%s' "$title"
}

for src in "$WIKI_DIR"/*.md; do
  base="$(basename "$src")"
  if [[ "$base" == "_Sidebar.md" ]]; then
    continue
  fi

  dest="$TARGET_DIR/$base"
  first_line="$(sed -n '1p' "$src")"
  if [[ "$first_line" == "---" ]]; then
    cp "$src" "$dest"
    continue
  fi

  title="$(extract_title "$src")"
  escaped_title="$(escape_yaml_double_quoted "$title")"
  tmp="$(mktemp)"
  {
    printf '%s\n' '---'
    printf 'title: "%s"\n' "$escaped_title"
    printf '%s\n\n' '---'
    cat "$src"
  } > "$tmp"
  mv "$tmp" "$dest"
done

sidebar="$WIKI_DIR/_Sidebar.md"
if [[ -f "$sidebar" ]]; then
  pages=()
  while IFS= read -r page; do
    pages+=("$page")
  done < <(
    sed -n 's/^[[:space:]]*[*-][[:space:]]*\[[^]]*\](\([^):#][^)]*\)).*$/\1/p' "$sidebar" |
      awk 'NF && !seen[$0]++'
  )

  if [[ "${#pages[@]}" -gt 0 ]]; then
    meta="$TARGET_DIR/meta.json"
    tmp="$(mktemp)"
    {
      printf '%s\n' '{'
      printf '  "title": "NginxPulse Docs",\n'
      printf '  "pages": [\n'
      for i in "${!pages[@]}"; do
        page="$(escape_json_string "${pages[$i]}")"
        if [[ "$i" -eq "$((${#pages[@]} - 1))" ]]; then
          printf '    "%s"\n' "$page"
        else
          printf '    "%s",\n' "$page"
        fi
      done
      printf '  ]\n'
      printf '%s\n' '}'
    } > "$tmp"
    mv "$tmp" "$meta"
  fi
fi

echo "Synced wiki markdown files to $TARGET_DIR"
