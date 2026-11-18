#!/usr/bin/env bash
# Acceptance test for task 001-cart-container: cart runs as a container in
# Compose and answers GET /healthz. It starts cart with `docker compose up`
# and never runs `docker compose down` or removes volumes, because the
# developer's own stack may be running.
set -uo pipefail
cd "$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"

URL="http://127.0.0.1:4000/healthz"
failed=0
pass() { echo "  ok: $1"; }
fail() { echo "  FAIL: $1"; failed=1; }

# Outcome 1
echo "Outcome 1: docker compose build cart exits 0 from the repository root."
if docker compose build cart >/dev/null 2>&1; then
  pass "build exited 0"
else
  fail "docker compose build cart exited non-zero"
fi

# Outcome 7 is checked first among the runtime lines, so the set of running
# services is captured before this test starts anything.
echo "Outcome 7: docker compose up -d cart succeeds with no other service running, because cart depends on no other service."
before=$(docker compose ps --services --status running 2>/dev/null | grep -vx cart | sort)
if docker compose config cart 2>/dev/null | grep -q 'depends_on:'; then
  fail "the cart service (or a service it pulls in) declares depends_on"
fi
if docker compose up -d cart >/dev/null 2>&1; then
  after=$(docker compose ps --services --status running 2>/dev/null | grep -vx cart | sort)
  if [ "$before" = "$after" ]; then
    pass "up -d cart exited 0 and started no other service"
  else
    fail "up -d cart started other services: $(comm -13 <(echo "$before") <(echo "$after") | tr '\n' ' ')"
  fi
else
  fail "docker compose up -d cart exited non-zero"
fi

# Outcome 2
echo "Outcome 2: after docker compose up -d --build cart, GET $URL returns 200 within 60 seconds."
if docker compose up -d --build cart >/dev/null 2>&1; then
  status=000
  deadline=$((SECONDS + 60))
  while [ "$SECONDS" -lt "$deadline" ]; do
    status=$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 "$URL" || true)
    [ "$status" = "200" ] && break
    sleep 1
  done
  if [ "$status" = "200" ]; then
    pass "200 within 60 seconds"
  else
    fail "no 200 within 60 seconds (last status $status)"
  fi
else
  fail "docker compose up -d --build cart exited non-zero"
fi

# Outcome 3
echo "Outcome 3: /healthz has Content-Type: application/json and the body {\"status\":\"ok\"}."
hdrs=$(mktemp)
body=$(curl -s --max-time 5 -D "$hdrs" "$URL" || true)
# The media type must be application/json; a parameter such as charset is allowed.
ctype=$(grep -i '^content-type:' "$hdrs" | head -n 1 | cut -d: -f2- | tr -d '\r' | sed 's/^ *//')
rm -f "$hdrs"
if printf '%s' "$ctype" | grep -Eiq '^application/json( *;.*)?$'; then
  pass "Content-Type is $ctype"
else
  fail "Content-Type is '$ctype', want application/json"
fi
if [ "$body" = '{"status":"ok"}' ]; then
  pass "body is $body"
else
  fail "body is '$body', want {\"status\":\"ok\"}"
fi

# Outcome 4
echo "Outcome 4: /healthz returns 200 to a request with no Authorization header."
# curl sends no Authorization header unless told to; -H 'Authorization:'
# removes one should a config file add it.
status=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 -H 'Authorization:' "$URL" || true)
if [ "$status" = "200" ]; then
  pass "200 without Authorization"
else
  fail "status $status without Authorization, want 200"
fi

# Outcome 5
echo "Outcome 5: docker compose port cart 4000 prints 127.0.0.1:4000."
port=$(docker compose port cart 4000 2>/dev/null | tr -d '\r')
if [ "$port" = "127.0.0.1:4000" ]; then
  pass "port prints $port"
else
  fail "port prints '$port', want 127.0.0.1:4000"
fi

# Outcome 6
echo "Outcome 6: docker compose exec cart id -u prints a user id other than 0."
uid=$(docker compose exec -T cart id -u 2>/dev/null | tr -d '\r')
if [[ "$uid" =~ ^[0-9]+$ ]] && [ "$uid" != "0" ]; then
  pass "id -u prints $uid"
else
  fail "id -u prints '$uid', want a non-zero user id"
fi

exit "$failed"
