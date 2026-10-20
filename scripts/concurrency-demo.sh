#!/usr/bin/env bash
set -euo pipefail

base_url="${BASE_URL:-http://localhost:8081}"
attempts="${ATTEMPTS:-150}"
parallel="${PARALLEL:-20}"

export base_url

seq 1 "$attempts" | xargs -P "$parallel" -I{} bash -c '
  order_id=$(curl -sS -X POST "$base_url/api/orders" \
    -H "Content-Type: application/json" \
    --data-binary @db/demo-order.json | jq -r .id)

  if [ -z "$order_id" ] || [ "$order_id" = "null" ]; then
    echo create_error
    exit 0
  fi

  curl -sS -o /dev/null -w "%{http_code}\n" \
    -X POST "$base_url/api/orders/$order_id/reserve"
' | sort | uniq -c
