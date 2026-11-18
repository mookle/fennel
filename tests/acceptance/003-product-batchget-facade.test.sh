#!/usr/bin/env bash
# Acceptance test for task 003-product-batchget-facade: product serves
# GET /v1/skus:batchGet with the ids as a repeated sku_ids query parameter,
# validates them, and returns the contract's SkuBatch example whatever ids it
# was given. It starts product with `docker compose up` and never runs
# `docker compose down` or removes volumes, because the developer's own stack
# may be running.
set -uo pipefail
cd "$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"

BASE="http://127.0.0.1:8080"
TOKEN="dev-token"
ROUTE="/v1/skus:batchGet"
failed=0
pass() { echo "  ok: $1"; }
fail() { echo "  FAIL: $1"; failed=1; }

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# req NAME METHOD PATH_AND_QUERY [TOKEN]
# Sends a request with no body. With TOKEN it carries
# Authorization: Bearer TOKEN; without it, no Authorization header at all.
# Records the status in $TMP/NAME.status, the headers in $TMP/NAME.headers and
# the body in $TMP/NAME.body.
req() {
  local name=$1 method=$2 path=$3 token=${4-}
  local args=(-s --max-time 10 -X "$method" -D "$TMP/$name.headers" -o "$TMP/$name.body" -w '%{http_code}')
  if [ -n "$token" ]; then
    args+=(-H "Authorization: Bearer $token")
  else
    args+=(-H 'Authorization:')
  fi
  curl "${args[@]}" "$BASE$path" >"$TMP/$name.status" 2>/dev/null || echo 000 >"$TMP/$name.status"
  touch "$TMP/$name.headers" "$TMP/$name.body"
}
status_of() { cat "$TMP/$1.status"; }
body_of() { head -c 500 "$TMP/$1.body"; }
header_of() { grep -i "^$2:" "$TMP/$1.headers" | tail -n 1 | cut -d: -f2- | tr -d '\r' | sed 's/^ *//'; }

# check_status NAME WANT
check_status() {
  if [ "$(status_of "$1")" = "$2" ]; then
    pass "$1: status $2"
  else
    fail "$1: status $(status_of "$1"), want $2 (body: $(body_of "$1"))"
  fi
}

# check_example_body NAME compares NAME's body to the value of the SkuBatch
# example in contracts/product.openapi.yaml, as parsed JSON.
check_example_body() {
  local out
  if out=$(python3 -I -c '
import json, sys, yaml
want = yaml.safe_load(open(sys.argv[1]))["components"]["examples"]["SkuBatch"]["value"]
try:
    got = json.load(open(sys.argv[2]))
except Exception as e:
    print("body is not JSON: %s" % e); sys.exit(1)
if got != want:
    print("body differs from the SkuBatch example\n    got:  %s\n    want: %s" % (json.dumps(got, sort_keys=True), json.dumps(want, sort_keys=True)))
    sys.exit(1)
' contracts/product.openapi.yaml "$TMP/$1.body" 2>&1); then
    pass "$1: body is JSON-equal to the SkuBatch example"
  else
    fail "$1: $out"
  fi
}

# query N [REPEAT_ID] prints a query string of N sku_ids values. With
# REPEAT_ID every value is that id; without it the values are distinct
# lowercase hyphenated UUIDs, none of them an id in the SkuBatch example.
query() {
  python3 -I -c '
import sys, uuid, yaml
n = int(sys.argv[1])
repeat = sys.argv[3] if len(sys.argv) > 3 else ""
ex = yaml.safe_load(open(sys.argv[2]))["components"]["examples"]["SkuBatch"]["value"]
taken = {i["sku_id"] for i in ex["items"]} | set(ex["missing"])
ids = []
while len(ids) < n:
    u = repeat or str(uuid.uuid4())
    if repeat or (u not in taken and u not in ids):
        ids.append(u)
print("&".join("sku_ids=" + u for u in ids))
' "$1" contracts/product.openapi.yaml "${2-}"
}

echo "Setup: docker compose up -d --build product, then wait up to 90 seconds for product to answer."
if docker compose up -d --build product >/dev/null 2>&1; then
  status=000
  deadline=$((SECONDS + 90))
  while [ "$SECONDS" -lt "$deadline" ]; do
    status=$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 -H "Authorization: Bearer $TOKEN" "$BASE/v1/products" || true)
    [ "$status" != "000" ] && break
    sleep 1
  done
  [ "$status" != "000" ] || fail "setup: product did not answer within 90 seconds"
else
  fail "setup: docker compose up -d --build product exited non-zero"
fi

# A lowercase hyphenated UUID that is not an id in the SkuBatch example.
ONE_ID="5b6c7d8e-9f0a-4b1c-8d2e-3f4a5b6c7d8e"

# Outcome 1
echo "Outcome 1: GET $ROUTE?sku_ids=<uuid> with the token returns 200, Content-Type: application/json, and the SkuBatch example."
req one GET "$ROUTE?sku_ids=$ONE_ID" "$TOKEN"
check_status one 200
ctype=$(header_of one content-type)
# The media type must be application/json; a parameter such as charset is allowed.
if printf '%s' "$ctype" | grep -Eiq '^application/json( *;.*)?$'; then
  pass "one: Content-Type is $ctype"
else
  fail "one: Content-Type is '$ctype', want application/json"
fi
check_example_body one

# Outcome 2
echo "Outcome 2: GET $ROUTE with 100 distinct sku_ids, none in the example, with the token returns 200 and the same body as line 1."
q100=$(query 100)
n=$(printf '%s' "$q100" | tr '&' '\n' | sort -u | wc -l)
[ "$n" = "100" ] || fail "test setup: built $n distinct values, want 100"
req hundred GET "$ROUTE?$q100" "$TOKEN"
check_status hundred 200
check_example_body hundred

# Outcome 3
echo "Outcome 3: GET $ROUTE with 101 distinct sku_ids, with the token returns 400."
q101=$(query 101)
n=$(printf '%s' "$q101" | tr '&' '\n' | sort -u | wc -l)
[ "$n" = "101" ] || fail "test setup: built $n distinct values, want 101"
req hundred_one GET "$ROUTE?$q101" "$TOKEN"
check_status hundred_one 400

# Outcome 4
echo "Outcome 4: GET $ROUTE with 101 sku_ids values that are all the same id, with the token returns 400."
req hundred_one_same GET "$ROUTE?$(query 101 "$ONE_ID")" "$TOKEN"
check_status hundred_one_same 400
# Boundary: 100 copies of the same id are within the limit.
req hundred_same GET "$ROUTE?$(query 100 "$ONE_ID")" "$TOKEN"
check_status hundred_same 200

# Outcome 5
echo "Outcome 5: GET $ROUTE with no sku_ids parameter, with the token returns 400."
req none GET "$ROUTE" "$TOKEN"
check_status none 400

# Outcome 6
echo "Outcome 6: GET $ROUTE?sku_ids=<uuid>&sku_ids=not-a-uuid with the token returns 400."
req bad_id GET "$ROUTE?sku_ids=$ONE_ID&sku_ids=not-a-uuid" "$TOKEN"
check_status bad_id 400

# Outcome 7
echo "Outcome 7: GET $ROUTE?sku_ids=<uuid> with no Authorization header returns 401 with WWW-Authenticate: Bearer."
req no_auth GET "$ROUTE?sku_ids=$ONE_ID"
check_status no_auth 401
www=$(header_of no_auth www-authenticate)
# The scheme must be Bearer; parameters such as realm are allowed.
if printf '%s' "$www" | grep -Eq '^Bearer( .*)?$'; then
  pass "no_auth: WWW-Authenticate is $www"
else
  fail "no_auth: WWW-Authenticate is '$www', want Bearer"
fi

# Outcome 8
echo "Outcome 8: GET $ROUTE?sku_ids=<uuid> with a bearer token compose.yaml does not configure returns 401."
req bad_token GET "$ROUTE?sku_ids=$ONE_ID" "not-a-configured-token"
check_status bad_token 401

# Outcome 9
echo "Outcome 9: POST $ROUTE with the token returns 405."
req post POST "$ROUTE" "$TOKEN"
check_status post 405

# Outcome 10
echo "Outcome 10: GET /v1/products with the token still returns 200."
req products GET /v1/products "$TOKEN"
check_status products 200

# Outcome 11
echo "Outcome 11: redocly lint --extends=minimal on contracts/product.openapi.yaml reports no errors."
if out=$(docker run --rm -v "$PWD/contracts:/spec:ro" redocly/cli lint --extends=minimal /spec/product.openapi.yaml 2>&1); then
  pass "redocly lint exited 0"
else
  fail "redocly lint exited non-zero:"
  printf '%s\n' "$out" | tail -n 20 | sed 's/^/    /'
fi

exit "$failed"
