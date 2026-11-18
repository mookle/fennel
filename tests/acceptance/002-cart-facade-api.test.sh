#!/usr/bin/env bash
# Acceptance test for task 002-cart-facade-api: cart serves POST /v1/carts,
# POST /v1/carts/{id}/items and POST /v1/carts/{id}/checkout with fixed
# responses, and contracts/cart.openapi.yaml lints clean. It starts cart with
# `docker compose up` and never runs `docker compose down` or removes volumes,
# because the developer's own stack may be running.
set -uo pipefail
cd "$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"

BASE="http://127.0.0.1:4000"
UUIDV7_RE='^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
failed=0
pass() { echo "  ok: $1"; }
fail() { echo "  FAIL: $1"; failed=1; }

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# req NAME METHOD PATH [JSON_BODY]
# Sends a request with no Authorization header and records the status in
# $TMP/NAME.status, the headers in $TMP/NAME.headers and the body in
# $TMP/NAME.body. With no JSON_BODY the request carries no body.
req() {
  local name=$1 method=$2 path=$3 data=${4-}
  local args=(-s --max-time 5 -X "$method" -H 'Authorization:' -D "$TMP/$name.headers" -o "$TMP/$name.body" -w '%{http_code}')
  if [ -n "$data" ]; then
    args+=(-H 'Content-Type: application/json' --data-binary "$data")
  fi
  curl "${args[@]}" "$BASE$path" >"$TMP/$name.status" 2>/dev/null || echo 000 >"$TMP/$name.status"
  touch "$TMP/$name.headers" "$TMP/$name.body"
}
status_of() { cat "$TMP/$1.status"; }
body_of() { cat "$TMP/$1.body"; }
ctype_of() { grep -i '^content-type:' "$TMP/$1.headers" | tail -n 1 | cut -d: -f2- | tr -d '\r' | sed 's/^ *//'; }
# field NAME KEY prints the JSON value of KEY in NAME's body, encoded as JSON
# (so a string prints quoted and a number bare), or nothing if the body is not
# a JSON object holding KEY.
field() {
  python3 -I -c '
import json, sys
try:
    d = json.load(open(sys.argv[1]))
    print(json.dumps(d[sys.argv[2]]))
except Exception:
    pass
' "$TMP/$1.body" "$2"
}
# str_field NAME KEY prints the value of KEY only if it is a JSON string.
str_field() {
  local v; v=$(field "$1" "$2")
  case "$v" in \"*\") python3 -I -c 'import json,sys; print(json.loads(sys.argv[1]))' "$v" ;; esac
}

echo "Setup: docker compose up -d --build cart, then wait up to 60 seconds for /healthz."
if docker compose up -d --build cart >/dev/null 2>&1; then
  status=000
  deadline=$((SECONDS + 60))
  while [ "$SECONDS" -lt "$deadline" ]; do
    status=$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 "$BASE/healthz" || true)
    [ "$status" = "200" ] && break
    sleep 1
  done
  [ "$status" = "200" ] || fail "setup: no 200 from /healthz within 60 seconds (last status $status)"
else
  fail "setup: docker compose up -d --build cart exited non-zero"
fi

SKU_A="0190a1b2-c3d4-7e5f-8a6b-7c8d9e0f1a2b"
SKU_B="3f2504e0-4f89-41d3-9a0c-0305e82c3301"
UNKNOWN_CART="5b6c7d8e-9f0a-4b1c-8d2e-3f4a5b6c7d8e"

# Requests used by several Outcome lines.
req cart1 POST /v1/carts
req cart2 POST /v1/carts
CART_ID=$(str_field cart1 id)
ITEM_CART=${CART_ID:-$UNKNOWN_CART}
req item1 POST "/v1/carts/$ITEM_CART/items" "{\"sku_id\":\"$SKU_A\",\"quantity\":2}"
req item2 POST "/v1/carts/$ITEM_CART/items" "{\"sku_id\":\"$SKU_B\",\"quantity\":7}"
req item_unknown POST "/v1/carts/$UNKNOWN_CART/items" "{\"sku_id\":\"$SKU_A\",\"quantity\":2}"
req checkout1 POST "/v1/carts/$ITEM_CART/checkout"
req checkout2 POST "/v1/carts/$ITEM_CART/checkout"

# Outcome 1
echo "Outcome 1: POST /v1/carts returns 201 with a JSON body whose id is a lowercase UUIDv7 string."
if [ "$(status_of cart1)" = "201" ]; then
  pass "status 201"
else
  fail "status $(status_of cart1), want 201"
fi
if [[ "$CART_ID" =~ $UUIDV7_RE ]]; then
  pass "id is $CART_ID"
else
  fail "id is '$(field cart1 id)', want a lowercase UUIDv7 string (body: $(body_of cart1))"
fi

# Outcome 2
echo "Outcome 2: two calls to POST /v1/carts return the same id."
id2=$(str_field cart2 id)
if [ -n "$CART_ID" ] && [ "$CART_ID" = "$id2" ]; then
  pass "both calls return $CART_ID"
else
  fail "first id '$CART_ID', second id '$id2', want equal and non-empty"
fi

# Outcome 3
echo "Outcome 3: POST /v1/carts/{id}/items with {\"sku_id\":\"<uuid>\",\"quantity\":2} returns 201 and echoes sku_id and quantity."
check_echo() {
  local name=$1 sku=$2 qty=$3
  if [ "$(status_of "$name")" = "201" ]; then
    pass "$name: status 201"
  else
    fail "$name: status $(status_of "$name"), want 201"
  fi
  local got_sku got_qty
  got_sku=$(field "$name" sku_id)
  got_qty=$(field "$name" quantity)
  if [ "$got_sku" = "\"$sku\"" ]; then
    pass "$name: sku_id is $got_sku"
  else
    fail "$name: sku_id is '$got_sku', want \"$sku\" (body: $(body_of "$name"))"
  fi
  # The quantity must come back as the JSON number sent, not a string.
  if [ "$got_qty" = "$qty" ]; then
    pass "$name: quantity is $got_qty"
  else
    fail "$name: quantity is '$got_qty', want the number $qty (body: $(body_of "$name"))"
  fi
}
check_echo item1 "$SKU_A" 2
# A second request with other values shows the response echoes the request
# rather than returning fixed values.
check_echo item2 "$SKU_B" 7

# Outcome 4
echo "Outcome 4: POST /v1/carts/{id}/items returns the same status and echo for an id that POST /v1/carts never returned."
if [ "$UNKNOWN_CART" = "$CART_ID" ]; then
  fail "the unknown cart id equals the id POST /v1/carts returned; the test needs another id"
fi
check_echo item_unknown "$SKU_A" 2

# Outcome 5
echo "Outcome 5: POST /v1/carts/{id}/checkout returns 202 with a JSON body whose purchase_id is a lowercase UUIDv7 string."
PURCHASE_ID=$(str_field checkout1 purchase_id)
if [ "$(status_of checkout1)" = "202" ]; then
  pass "status 202"
else
  fail "status $(status_of checkout1), want 202"
fi
if [[ "$PURCHASE_ID" =~ $UUIDV7_RE ]]; then
  pass "purchase_id is $PURCHASE_ID"
else
  fail "purchase_id is '$(field checkout1 purchase_id)', want a lowercase UUIDv7 string (body: $(body_of checkout1))"
fi

# Outcome 6
echo "Outcome 6: two calls to POST /v1/carts/{id}/checkout return the same purchase_id, and it differs from the cart id."
pid2=$(str_field checkout2 purchase_id)
if [ -n "$PURCHASE_ID" ] && [ "$PURCHASE_ID" = "$pid2" ]; then
  pass "both calls return $PURCHASE_ID"
else
  fail "first purchase_id '$PURCHASE_ID', second '$pid2', want equal and non-empty"
fi
if [ -n "$PURCHASE_ID" ] && [ -n "$CART_ID" ] && [ "$PURCHASE_ID" != "$CART_ID" ]; then
  pass "purchase_id differs from cart id $CART_ID"
else
  fail "purchase_id '$PURCHASE_ID' and cart id '$CART_ID', want both non-empty and different"
fi

# Outcome 7
echo "Outcome 7: every response from the three routes has Content-Type: application/json."
for name in cart1 cart2 item1 item2 item_unknown checkout1 checkout2; do
  ctype=$(ctype_of "$name")
  # The media type must be application/json; a parameter such as charset is allowed.
  if printf '%s' "$ctype" | grep -Eiq '^application/json( *;.*)?$'; then
    pass "$name: Content-Type is $ctype"
  else
    fail "$name: Content-Type is '$ctype', want application/json"
  fi
done

# Outcome 8
echo "Outcome 8: each of the three routes succeeds with no Authorization header."
# Every request above went out with no Authorization header (req passes
# -H 'Authorization:', which removes one should a curl config add it).
for pair in cart1:201 item1:201 checkout1:202; do
  name=${pair%%:*}; want=${pair#*:}
  if [ "$(status_of "$name")" = "$want" ]; then
    pass "$name: $want without Authorization"
  else
    fail "$name: status $(status_of "$name") without Authorization, want $want"
  fi
done

# Outcome 9
echo "Outcome 9: GET /healthz still returns 200 with {\"status\":\"ok\"}."
req health GET /healthz
if [ "$(status_of health)" = "200" ]; then
  pass "status 200"
else
  fail "status $(status_of health), want 200"
fi
if [ "$(body_of health)" = '{"status":"ok"}' ]; then
  pass "body is $(body_of health)"
else
  fail "body is '$(body_of health)', want {\"status\":\"ok\"}"
fi

# Outcome 10
echo "Outcome 10: contracts/cart.openapi.yaml passes redocly lint with the minimal ruleset."
if [ ! -f contracts/cart.openapi.yaml ]; then
  fail "contracts/cart.openapi.yaml does not exist"
elif out=$(docker run --rm -v "$PWD/contracts:/spec:ro" redocly/cli lint --extends=minimal /spec/cart.openapi.yaml 2>&1); then
  pass "redocly lint exited 0"
else
  fail "redocly lint exited non-zero:"
  printf '%s\n' "$out" | tail -n 20 | sed 's/^/    /'
fi

exit "$failed"
