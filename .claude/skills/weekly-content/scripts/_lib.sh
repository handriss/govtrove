# Shared helpers for the weekly-content pipeline. Source this: . _lib.sh
set -euo pipefail

SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(git -C "$SCRIPTS_DIR" rev-parse --show-toplevel)"
VENV="$HOME/.config/govtrove/content-venv"
STATE_DIR="$HOME/.config/govtrove"

# psql from libpq (not on PATH by default on this Mac)
export PATH="/opt/homebrew/opt/libpq/bin:$PATH"

# Load .env WITHOUT shell-evaluating values (values contain & ? / which `source` would break).
load_env() {
  local f="$ROOT/.env" key val
  [ -f "$f" ] || { echo "ERROR: $f not found" >&2; return 1; }
  while IFS= read -r line || [ -n "$line" ]; do
    case "$line" in ''|\#*) continue;; esac
    case "$line" in *=*) : ;; *) continue;; esac
    key="${line%%=*}"; val="${line#*=}"
    key="$(printf '%s' "$key" | tr -d '[:space:]')"
    # .env keeps NEON_DATABASE_URL single-quoted so `source .env` survives the `&`.
    # We don't shell-evaluate, so the quotes would otherwise land inside the value.
    case "$val" in
      \'*\')  val="${val#\'}";  val="${val%\'}"  ;;
      \"*\")  val="${val#\"}";  val="${val%\"}"  ;;
    esac
    export "$key=$val"
  done < "$f"
}
load_env

require() {
  local missing=0 v
  for v in "$@"; do
    if [ -z "${!v:-}" ]; then echo "MISSING env: $v" >&2; missing=1; fi
  done
  return $missing
}
