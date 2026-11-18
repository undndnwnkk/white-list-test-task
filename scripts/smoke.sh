#!/usr/bin/env bash
# End-to-end check of the business rules with plain curl.
# Usage: ./scripts/smoke.sh            (API at http://localhost:8080)
#        BASE_URL=http://localhost:9090 ./scripts/smoke.sh
#
# Every run uses a new company BIN and a random future Monday, and cancels its
# meetings at the end, so the script can be re-run against the same database.

BASE_URL=${BASE_URL:-http://localhost:8080}
MANAGER_A=11111111-1111-1111-1111-111111111111
MANAGER_B=22222222-2222-2222-2222-222222222222

# Mondays in the future; times below are UTC, Asia/Almaty = UTC+5.
MONDAYS=(2030-01-07 2030-04-08 2030-07-08 2030-10-07 2031-01-06 2031-04-07 2031-07-07 2031-10-06
         2032-01-05 2032-04-05 2032-07-05 2032-10-04 2033-01-03 2033-04-04 2033-07-04 2033-10-03
         2034-01-02 2034-04-03 2034-07-03 2034-10-02 2036-03-31 2036-06-30 2036-09-29 2036-12-29
         2037-03-30 2037-06-29 2037-09-28 2037-12-28 2038-03-29 2038-06-28 2038-09-27 2038-12-27)
DAY=${MONDAYS[$((RANDOM % ${#MONDAYS[@]}))]}

TMP=$(mktemp)
FAILURES=0
CLEANUP_IDS=()

cleanup() {
  for id in "${CLEANUP_IDS[@]}"; do
    curl -s -o /dev/null -X PATCH "$BASE_URL/meetings/$id" \
      -H 'Content-Type: application/json' -d '{"status":"cancelled"}'
  done
  rm -f "$TMP"
}
trap cleanup EXIT

# req NAME EXPECTED_STATUS METHOD PATH [JSON_BODY]; response body is left in $RESP.
req() {
  local name=$1 expected=$2 method=$3 path=$4 body=${5:-}
  local code
  if [ -n "$body" ]; then
    code=$(curl -s -o "$TMP" -w '%{http_code}' -X "$method" "$BASE_URL$path" \
      -H 'Content-Type: application/json' -d "$body")
  else
    code=$(curl -s -o "$TMP" -w '%{http_code}' -X "$method" "$BASE_URL$path")
  fi
  if [ "$code" = "000" ]; then
    echo "Cannot reach the API at $BASE_URL. Start it first: docker compose up --build -d --wait"
    exit 1
  fi
  RESP=$(cat "$TMP")
  if [ "$code" = "$expected" ]; then
    printf 'PASS  %-62s %s\n' "$name" "$code"
  else
    printf 'FAIL  %-62s got %s, want %s\n      %s\n' "$name" "$code" "$expected" "$RESP"
    FAILURES=$((FAILURES + 1))
  fi
}

id_of() { printf '%s' "$RESP" | sed -n 's/^{"id":"\([^"]*\)".*/\1/p'; }

new_bin() { printf '%012d' "$(( ((RANDOM * 32768 + RANDOM) * 1000 + RANDOM % 1000) % 1000000000000 ))"; }

echo "API: $BASE_URL, meeting day: $DAY (Monday)"
echo

BIN1=$(new_bin)
req "create company" 201 POST /companies \
  "{\"name\":\"ТОО Smoke 1\",\"bin\":\"$BIN1\",\"phone\":\"+77011234567\"}"
C1=$(id_of)
req "duplicate BIN is rejected" 409 POST /companies \
  "{\"name\":\"ТОО Smoke dup\",\"bin\":\"$BIN1\",\"phone\":\"+77011234567\"}"
req "invalid BIN is rejected" 400 POST /companies \
  '{"name":"ТОО Bad","bin":"12345","phone":"+77011234567"}'
req "second company" 201 POST /companies \
  "{\"name\":\"ТОО Smoke 2\",\"bin\":\"$(new_bin)\",\"phone\":\"+77017654321\"}"
C2=$(id_of)

req "list companies with status filter" 200 GET "/companies?status=new&limit=5&offset=0"
req "unknown status filter is rejected" 400 GET "/companies?status=bogus"

req "call: no_answer" 201 POST /calls \
  "{\"company_id\":\"$C1\",\"manager_id\":\"$MANAGER_A\",\"result\":\"no_answer\"}"
req "call: meeting_set without meeting is rejected" 400 POST /calls \
  "{\"company_id\":\"$C1\",\"manager_id\":\"$MANAGER_A\",\"result\":\"meeting_set\"}"

req "meeting at 17:30 Almaty does not fit the day" 422 POST /meetings \
  "{\"company_id\":\"$C1\",\"manager_id\":\"$MANAGER_A\",\"start_at\":\"${DAY}T12:30:00Z\"}"
req "meeting at 08:00 Almaty is before the day" 422 POST /meetings \
  "{\"company_id\":\"$C1\",\"manager_id\":\"$MANAGER_A\",\"start_at\":\"${DAY}T03:00:00Z\"}"
req "meeting at 10:00 Almaty" 201 POST /meetings \
  "{\"company_id\":\"$C1\",\"manager_id\":\"$MANAGER_A\",\"start_at\":\"${DAY}T05:00:00Z\"}"
M1=$(id_of); CLEANUP_IDS+=("$M1")

req "call: meeting_set with the meeting" 201 POST /calls \
  "{\"company_id\":\"$C1\",\"manager_id\":\"$MANAGER_A\",\"result\":\"meeting_set\",\"meeting_id\":\"$M1\"}"
req "same meeting cannot be linked to a second call" 409 POST /calls \
  "{\"company_id\":\"$C1\",\"manager_id\":\"$MANAGER_A\",\"result\":\"meeting_set\",\"meeting_id\":\"$M1\"}"

req "manager overlap 10:30 is rejected" 409 POST /meetings \
  "{\"company_id\":\"$C2\",\"manager_id\":\"$MANAGER_A\",\"start_at\":\"${DAY}T05:30:00Z\"}"
req "back-to-back 11:00 is not a conflict" 201 POST /meetings \
  "{\"company_id\":\"$C2\",\"manager_id\":\"$MANAGER_A\",\"start_at\":\"${DAY}T06:00:00Z\"}"
M2=$(id_of); CLEANUP_IDS+=("$M2")
req "company with active meeting, other manager, is rejected" 409 POST /meetings \
  "{\"company_id\":\"$C1\",\"manager_id\":\"$MANAGER_B\",\"start_at\":\"${DAY}T09:00:00Z\"}"

req "reschedule meeting to 14:00 Almaty" 200 PATCH "/meetings/$M1" \
  "{\"start_at\":\"${DAY}T09:00:00Z\"}"
case "$RESP" in
  *"\"original_start_at\":\"${DAY}T05:00:00Z\""*'"type":"rescheduled"'*) echo "PASS  original start and history are kept" ;;
  *) echo "FAIL  original start and history are kept"; echo "      $RESP"; FAILURES=$((FAILURES + 1)) ;;
esac
req "reschedule into a manager's busy slot is rejected" 409 PATCH "/meetings/$M1" \
  "{\"start_at\":\"${DAY}T06:00:00Z\"}"

req "manager schedule for the day" 200 GET "/managers/$MANAGER_A/schedule?date=$DAY"
req "schedule with a bad date is rejected" 400 GET "/managers/$MANAGER_A/schedule?date=07.01.2030"
req "schedule of unknown manager" 404 GET "/managers/99999999-9999-9999-9999-999999999999/schedule?date=$DAY"

req "cancel meeting" 200 PATCH "/meetings/$M1" '{"status":"cancelled"}'
req "cancelled meeting cannot be cancelled again" 409 PATCH "/meetings/$M1" '{"status":"cancelled"}'
req "meeting history" 200 GET "/meetings/$M1"
req "unknown meeting" 404 GET "/meetings/99999999-9999-9999-9999-999999999999"

echo
if [ "$FAILURES" -eq 0 ]; then
  echo "All checks passed."
else
  echo "$FAILURES check(s) failed."
  exit 1
fi
