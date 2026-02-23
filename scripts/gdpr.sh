#!/usr/bin/env bash
# GDPR scripts: Data Export (DSAR) and Account Deletion (Right to Erasure)
#
# Prerequisites:
#   - DATABASE_URL env var set (or pass Neon connection string)
#   - WORKOS_API_KEY env var set (sk_... key)
#
# Usage:
#   ./gdpr-scripts.sh export <email>
#   ./gdpr-scripts.sh delete <email>

set -euo pipefail

DB="${DATABASE_URL:?Set DATABASE_URL}"
WORKOS_KEY="${WORKOS_API_KEY:?Set WORKOS_API_KEY}"

ACTION="${1:-}"
EMAIL="${2:-}"

if [[ -z "$ACTION" || -z "$EMAIL" ]]; then
  echo "Usage: $0 <export|delete> <email>"
  exit 1
fi

# Look up user by email
get_user() {
  psql "$DB" -t -A -F$'\t' -c "
    SELECT id, workos_id, email, first_name, last_name, plan, created_at, updated_at
    FROM users WHERE email = '$EMAIL' LIMIT 1
  "
}

USER_ROW=$(get_user)
if [[ -z "$USER_ROW" ]]; then
  echo "No user found with email: $EMAIL"
  exit 1
fi

USER_ID=$(echo "$USER_ROW" | cut -f1)
WORKOS_ID=$(echo "$USER_ROW" | cut -f2)

echo "Found user: id=$USER_ID workos_id=$WORKOS_ID"

# ─── DATA EXPORT (DSAR) ─────────────────────────────────────────────

do_export() {
  OUTDIR="dsar-export-${EMAIL}-$(date +%Y%m%d)"
  mkdir -p "$OUTDIR"

  echo "Exporting data to $OUTDIR/ ..."

  # 1. User profile (local DB)
  psql "$DB" -c "
    SELECT id, workos_id, email, first_name, last_name, plan, created_at, updated_at
    FROM users WHERE id = $USER_ID
  " --csv > "$OUTDIR/user.csv"

  # 2. Saved opportunities
  psql "$DB" -c "
    SELECT so.opportunity_id, o.title, o.solicitation_number, so.created_at
    FROM saved_opportunities so
    LEFT JOIN opportunities o ON o.id = so.opportunity_id
    WHERE so.user_id = $USER_ID
    ORDER BY so.created_at DESC
  " --csv > "$OUTDIR/saved_opportunities.csv"

  # 3. Saved searches
  psql "$DB" -c "
    SELECT id, name, filters, created_at, updated_at
    FROM saved_searches WHERE user_id = $USER_ID
    ORDER BY updated_at DESC
  " --csv > "$OUTDIR/saved_searches.csv"

  # 4. Account requests
  psql "$DB" -c "
    SELECT id, request_type, status, created_at
    FROM account_requests WHERE user_id = $USER_ID
    ORDER BY created_at DESC
  " --csv > "$OUTDIR/account_requests.csv"

  # 5. Contact messages (by email, not user_id)
  psql "$DB" -c "
    SELECT id, name, subject, message, created_at
    FROM contact_messages WHERE email = '$EMAIL'
    ORDER BY created_at DESC
  " --csv > "$OUTDIR/contact_messages.csv"

  # 6. WorkOS profile
  curl -s "https://api.workos.com/user_management/users/$WORKOS_ID" \
    -H "Authorization: Bearer $WORKOS_KEY" \
    | python3 -m json.tool > "$OUTDIR/workos_profile.json" 2>/dev/null || echo '{"error": "could not fetch"}' > "$OUTDIR/workos_profile.json"

  echo ""
  echo "Export complete:"
  ls -la "$OUTDIR/"
  echo ""
  echo "Review and send to user within 30 days of request."
}

# ─── ACCOUNT DELETION (RIGHT TO ERASURE) ────────────────────────────

do_delete() {
  echo ""
  echo "This will PERMANENTLY delete user $EMAIL ($WORKOS_ID) from:"
  echo "  1. GovTrove database (user + saved opps + saved searches + account requests)"
  echo "  2. WorkOS (user profile + auth sessions)"
  echo ""
  echo "Contact messages will be anonymized (email removed)."
  echo "Stripe retains financial records per legal obligation (GDPR Art. 6(1)(c))."
  echo ""
  read -p "Type 'DELETE' to confirm: " CONFIRM
  if [[ "$CONFIRM" != "DELETE" ]]; then
    echo "Aborted."
    exit 1
  fi

  echo ""

  # 1. Revoke WorkOS sessions
  echo "Revoking WorkOS sessions..."
  SESSIONS=$(curl -s "https://api.workos.com/user_management/users/$WORKOS_ID/sessions" \
    -H "Authorization: Bearer $WORKOS_KEY" \
    | python3 -c "import sys,json; [print(s['id']) for s in json.load(sys.stdin).get('data',[])]" 2>/dev/null || true)

  for SID in $SESSIONS; do
    curl -s -X POST "https://api.workos.com/user_management/sessions/revoke" \
      -H "Authorization: Bearer $WORKOS_KEY" \
      -H "Content-Type: application/json" \
      -d "{\"session_id\": \"$SID\"}" > /dev/null
    echo "  Revoked session $SID"
  done

  # 2. Delete from WorkOS
  echo "Deleting user from WorkOS..."
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE \
    "https://api.workos.com/user_management/users/$WORKOS_ID" \
    -H "Authorization: Bearer $WORKOS_KEY")
  echo "  WorkOS deletion response: $HTTP_CODE"

  # 3. Anonymize contact messages (keep for operational records, strip PII)
  echo "Anonymizing contact messages..."
  psql "$DB" -c "
    UPDATE contact_messages
    SET name = 'deleted', email = 'deleted@deleted.invalid'
    WHERE email = '$EMAIL'
  "

  # 4. Delete user from DB (cascades to saved_opportunities, saved_searches)
  echo "Deleting user from database..."
  psql "$DB" -c "DELETE FROM account_requests WHERE user_id = $USER_ID"
  psql "$DB" -c "DELETE FROM users WHERE id = $USER_ID"

  echo ""
  echo "Deletion complete for $EMAIL."
  echo ""
  echo "Remaining obligations:"
  echo "  - Stripe: financial records retained per legal obligation (inform user)"
  echo "  - Backups: automatically purged within 90 days per privacy policy"
}

# ─── DISPATCH ────────────────────────────────────────────────────────

case "$ACTION" in
  export) do_export ;;
  delete) do_delete ;;
  *) echo "Unknown action: $ACTION. Use 'export' or 'delete'." ; exit 1 ;;
esac
