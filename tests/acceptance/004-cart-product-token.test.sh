#!/usr/bin/env bash
# Acceptance test for task 004-cart-product-token: compose.yaml gives cart a
# bearer token that product accepts and the URL of product, cart reads both
# into its application config at start, and the release refuses to start
# without either of them. It starts what it needs with `docker compose up` and
# never runs `docker compose down` or removes volumes, because the developer's
# own stack may be running. Lines 8 to 11 run the built image with
# `docker run`, so they never start or stop a Compose service.
set -uo pipefail
cd "$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"

BASE="http://127.0.0.1:8080"
WANT_URL="http://product:8080/v1"
IMAGE="fennel-cart"
failed=0
pass() { echo "  ok: $1"; }
fail() { echo "  FAIL: $1"; failed=1; }

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# The resolved config.
if ! docker compose config --format json >"$TMP/config.json" 2>"$TMP/config.err"; then
  fail "setup: docker compose config --format json exited non-zero: $(head -c 500 "$TMP/config.err")"
  echo '{}' >"$TMP/config.json"
fi

# env_of SERVICE VAR prints the value of VAR in SERVICE's resolved environment,
# and exits 1 when the service or the variable is absent or not a string.
env_of() {
  python3 -I -c '
import json, sys
c = json.load(open(sys.argv[1]))
env = (c.get("services", {}).get(sys.argv[2]) or {}).get("environment") or {}
v = env.get(sys.argv[3]) if isinstance(env, dict) else None
if not isinstance(v, str):
    sys.exit(1)
sys.stdout.write(v)
' "$TMP/config.json" "$1" "$2"
}

# Outcome 1
echo "Outcome 1: services.product.environment.FENNEL_PRODUCT_TOKEN_CART is a non-empty string, and services.cart.environment.FENNEL_CART_PRODUCT_TOKEN equals it."
TOKEN=""
if TOKEN=$(env_of product FENNEL_PRODUCT_TOKEN_CART) && [ -n "$TOKEN" ]; then
  pass "product's FENNEL_PRODUCT_TOKEN_CART is '$TOKEN'"
else
  TOKEN=""
  fail "product's FENNEL_PRODUCT_TOKEN_CART is absent, empty or not a string"
fi
if cart_token=$(env_of cart FENNEL_CART_PRODUCT_TOKEN); then
  if [ -n "$TOKEN" ] && [ "$cart_token" = "$TOKEN" ]; then
    pass "cart's FENNEL_CART_PRODUCT_TOKEN equals it"
  else
    fail "cart's FENNEL_CART_PRODUCT_TOKEN is '$cart_token', want '$TOKEN' (non-empty)"
  fi
else
  fail "cart's FENNEL_CART_PRODUCT_TOKEN is absent or not a string"
fi

# Outcome 2
echo "Outcome 2: services.cart.environment.FENNEL_CART_PRODUCT_URL is $WANT_URL."
if url=$(env_of cart FENNEL_CART_PRODUCT_URL) && [ "$url" = "$WANT_URL" ]; then
  pass "FENNEL_CART_PRODUCT_URL is $url"
else
  fail "FENNEL_CART_PRODUCT_URL is '${url-}', want $WANT_URL"
fi

# Outcome 3
echo "Outcome 3: the token's value appears exactly once in compose.yaml."
if [ -z "$TOKEN" ]; then
  fail "no token from line 1 to count"
else
  n=$(python3 -I -c 'import sys; print(open(sys.argv[1]).read().count(sys.argv[2]))' compose.yaml "$TOKEN")
  if [ "$n" = "1" ]; then
    pass "'$TOKEN' appears once"
  else
    fail "'$TOKEN' appears $n times, want 1"
  fi
fi

# Outcome 4
echo "Outcome 4: neither product nor cart has a depends_on entry that names the other."
for pair in "product cart" "cart product"; do
  set -- $pair
  if python3 -I -c '
import json, sys
c = json.load(open(sys.argv[1]))
svc = c.get("services", {}).get(sys.argv[2])
if svc is None:
    print("service %s is absent" % sys.argv[2]); sys.exit(1)
deps = svc.get("depends_on") or {}
if sys.argv[3] in deps:
    print("%s depends_on %s" % (sys.argv[2], sys.argv[3])); sys.exit(1)
' "$TMP/config.json" "$1" "$2" >"$TMP/dep.out" 2>&1; then
    pass "$1 has no depends_on naming $2"
  else
    fail "$(cat "$TMP/dep.out")"
  fi
done

echo "Setup: docker compose build cart, so the image $IMAGE is current."
if ! docker compose build cart >"$TMP/build.out" 2>&1; then
  fail "setup: docker compose build cart exited non-zero:"
  tail -n 20 "$TMP/build.out" | sed 's/^/    /'
fi

# run_image NAME VAR_NAME ENV_ARGS... runs the cart image with only the given
# -e arguments, under a 30-second timeout, and checks that it exits non-zero
# within the time and that its output names VAR_NAME. The container gets a
# unique name and is removed afterwards, in case the timeout left it running.
run_image() {
  local name=$1 want=$2; shift 2
  local cname="fennel-acc-004-$name-$$" rc
  timeout 30 docker run --rm --name "$cname" "$@" "$IMAGE" >"$TMP/$name.out" 2>&1
  rc=$?
  docker rm -f "$cname" >/dev/null 2>&1 || true
  case "$rc" in
    0) fail "$name: the image exited 0, want non-zero" ;;
    124) fail "$name: the image did not exit within 30 seconds" ;;
    125|126|127) fail "$name: docker run could not run the image (exit $rc): $(tail -n 5 "$TMP/$name.out")" ;;
    *) pass "$name: the image exited $rc within 30 seconds" ;;
  esac
  if grep -qF "$want" "$TMP/$name.out"; then
    pass "$name: output contains $want"
  else
    fail "$name: output does not contain $want; last lines:"
    tail -n 10 "$TMP/$name.out" | sed 's/^/    /'
  fi
}

# Outcome 8
echo "Outcome 8: the image, with FENNEL_CART_PRODUCT_URL set and FENNEL_CART_PRODUCT_TOKEN unset, exits non-zero within 30 seconds and its output contains FENNEL_CART_PRODUCT_TOKEN."
run_image token_unset FENNEL_CART_PRODUCT_TOKEN -e "FENNEL_CART_PRODUCT_URL=$WANT_URL"

# Outcome 9
echo "Outcome 9: the image, with FENNEL_CART_PRODUCT_URL set and FENNEL_CART_PRODUCT_TOKEN empty, exits non-zero within 30 seconds and its output contains FENNEL_CART_PRODUCT_TOKEN."
run_image token_empty FENNEL_CART_PRODUCT_TOKEN -e "FENNEL_CART_PRODUCT_URL=$WANT_URL" -e "FENNEL_CART_PRODUCT_TOKEN="

# Outcome 10
echo "Outcome 10: the image, with FENNEL_CART_PRODUCT_TOKEN set and FENNEL_CART_PRODUCT_URL unset, exits non-zero within 30 seconds and its output contains FENNEL_CART_PRODUCT_URL."
run_image url_unset FENNEL_CART_PRODUCT_URL -e "FENNEL_CART_PRODUCT_TOKEN=acceptance-token"

# Outcome 11
echo "Outcome 11: the image, with FENNEL_CART_PRODUCT_TOKEN set and FENNEL_CART_PRODUCT_URL empty, exits non-zero within 30 seconds and its output contains FENNEL_CART_PRODUCT_URL."
run_image url_empty FENNEL_CART_PRODUCT_URL -e "FENNEL_CART_PRODUCT_TOKEN=acceptance-token" -e "FENNEL_CART_PRODUCT_URL="

# Outcome 12 (token). done.md numbers two lines 12; this is the token one.
echo "Outcome 12 (token): a token consisting of empty spaces is treated as empty."
# The token from line 1 counts as empty if it holds only spaces.
if [ -n "$TOKEN" ] && [ -n "${TOKEN//[[:space:]]/}" ]; then
  pass "the token from line 1 is not only spaces"
else
  fail "the token from line 1 is '$TOKEN', which is empty or only spaces"
fi
# The release treats a token of spaces as it treats an empty one (line 9).
run_image token_one_space FENNEL_CART_PRODUCT_TOKEN -e "FENNEL_CART_PRODUCT_URL=$WANT_URL" -e "FENNEL_CART_PRODUCT_TOKEN= "
run_image token_spaces FENNEL_CART_PRODUCT_TOKEN -e "FENNEL_CART_PRODUCT_URL=$WANT_URL" -e "FENNEL_CART_PRODUCT_TOKEN=   "

# Outcome 12 (URL). done.md numbers two lines 12; this is the URL one.
echo "Outcome 12 (URL): a URL consisting of empty spaces is treated as empty."
# The URL from line 2 is checked exactly there, so it cannot be only spaces.
# The release treats a URL of spaces as it treats an empty one (line 11).
run_image url_one_space FENNEL_CART_PRODUCT_URL -e "FENNEL_CART_PRODUCT_TOKEN=acceptance-token" -e "FENNEL_CART_PRODUCT_URL= "
run_image url_spaces FENNEL_CART_PRODUCT_URL -e "FENNEL_CART_PRODUCT_TOKEN=acceptance-token" -e "FENNEL_CART_PRODUCT_URL=   "

# Outcome 5
echo "Outcome 5: GET /v1/products on product with Authorization: Bearer <the token from line 1> returns 200."
if [ -z "$TOKEN" ]; then
  fail "no token from line 1 to send"
elif docker compose up -d --build product >/dev/null 2>&1; then
  status=000
  deadline=$((SECONDS + 90))
  while [ "$SECONDS" -lt "$deadline" ]; do
    status=$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 -H "Authorization: Bearer $TOKEN" "$BASE/v1/products" || true)
    [ "$status" = "200" ] && break
    sleep 1
  done
  if [ "$status" = "200" ]; then
    pass "200 with the cart token"
  else
    fail "status $status with the cart token within 90 seconds, want 200"
  fi
else
  fail "setup: docker compose up -d --build product exited non-zero"
fi

echo "Setup: docker compose up -d --wait cart."
cart_up=0
if docker compose up -d --wait cart >"$TMP/up.out" 2>&1; then
  cart_up=1
else
  fail "setup: docker compose up -d --wait cart exited non-zero:"
  tail -n 20 "$TMP/up.out" | sed 's/^/    /'
fi

# rpc EXPR prints what bin/cart rpc EXPR prints in the running cart, with
# carriage returns and the final newline removed.
rpc() {
  docker compose exec -T cart /app/bin/cart rpc "$1" 2>&1 | tr -d '\r'
}

# Outcome 6
echo "Outcome 6: bin/cart rpc 'IO.puts(Application.fetch_env!(:cart, :product_url))' prints $WANT_URL."
if [ "$cart_up" = 1 ]; then
  got=$(rpc 'IO.puts(Application.fetch_env!(:cart, :product_url))')
  if [ "$got" = "$WANT_URL" ]; then
    pass "prints $got"
  else
    fail "prints '$(printf '%s' "$got" | head -c 500)', want $WANT_URL"
  fi
else
  fail "cart is not up"
fi

# Outcome 7
echo "Outcome 7: bin/cart rpc 'IO.puts(Application.fetch_env!(:cart, :product_token))' prints the token from line 1."
if [ "$cart_up" = 1 ] && [ -n "$TOKEN" ]; then
  got=$(rpc 'IO.puts(Application.fetch_env!(:cart, :product_token))')
  if [ "$got" = "$TOKEN" ]; then
    pass "prints $got"
  else
    fail "prints '$(printf '%s' "$got" | head -c 500)', want '$TOKEN'"
  fi
else
  fail "cart is not up, or there is no token from line 1"
fi

exit "$failed"
