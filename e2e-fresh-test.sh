#!/bin/bash
# e2e-fresh-test.sh

BASE_URL="http://localhost:8080/api/v1"
APP_KEY="dev_secret_key_123"
PASS=0
FAIL=0

pass() { PASS=$((PASS+1)); echo "✅ $1"; }
fail() { FAIL=$((FAIL+1)); echo "❌ $1"; }

echo "========================================="
echo "  E2E FRESH TEST - League Lifecycle"
echo "========================================="

# Generate unique username
USERNAME="e2e_$(date +%s)"

echo -e "\n=== 1. Register ==="
REGISTER=$(curl -s -X POST $BASE_URL/auth/register \
  -H "Content-Type: application/json" \
  -H "X-App-Key: $APP_KEY" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"password123\",\"gamename\":\"E2E Test\"}")
TOKEN=$(echo $REGISTER | jq -r '.session_id')
USER_ID=$(echo $REGISTER | jq -r '.user_id')
[ "$TOKEN" != "null" ] && [ "$TOKEN" != "" ] && pass "Register ($USERNAME)" || fail "Register: $REGISTER"

echo -e "\n=== 2. Check Wallet ==="
WALLET_BEFORE=$(curl -s $BASE_URL/wallet \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN" | jq -r '.kash')
[ "$WALLET_BEFORE" = "1000" ] && pass "Wallet: $1000" || fail "Wallet: $WALLET_BEFORE"

echo -e "\n=== 3. Create League ==="
CREATE=$(curl -s -X POST $BASE_URL/leagues/create \
  -H "Content-Type: application/json" \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"E2E League","team_count":5,"difficulty":"BEGINNER"}')
LEAGUE_ID=$(echo $CREATE | jq -r '.league_id')
[ "$LEAGUE_ID" != "null" ] && [ "$LEAGUE_ID" != "" ] && pass "Create League: $LEAGUE_ID" || fail "Create League: $CREATE"

echo -e "\n=== 4. Verify League Exists ==="
MY_LEAGUES=$(curl -s $BASE_URL/leagues/my \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN")
LEAGUE_COUNT=$(echo $MY_LEAGUES | jq 'length')
[ "$LEAGUE_COUNT" = "1" ] && pass "League exists in my leagues" || fail "League count: $LEAGUE_COUNT"

echo -e "\n=== 5. Wallet After Create (should be 1000-320=680) ==="
WALLET_AFTER_CREATE=$(curl -s $BASE_URL/wallet \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN" | jq -r '.kash')
[ "$WALLET_AFTER_CREATE" = "680" ] && pass "Wallet after create: $680" || fail "Wallet after create: $WALLET_AFTER_CREATE"

echo -e "\n=== 6. Place Winner Bet ==="
# Get a team ID
TEAM_ID=$(curl -s $BASE_URL/leagues/$LEAGUE_ID/teams \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN" | jq -r '.[0].id')
WINNER_BET=$(curl -s -X POST $BASE_URL/leagues/$LEAGUE_ID/winner-bet \
  -H "Content-Type: application/json" \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{\"team_id\":\"$TEAM_ID\",\"points_range\":\"50-60\"}")
[ "$(echo $WINNER_BET | jq -r '.status')" = "placed" ] && pass "Winner bet placed" || fail "Winner bet: $WINNER_BET"

echo -e "\n=== 7. Forfeit League ==="
FORFEIT=$(curl -s -X POST $BASE_URL/leagues/$LEAGUE_ID/forfeit \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN")
FORFEIT_STATUS=$(echo $FORFEIT | jq -r '.status')
[ "$FORFEIT_STATUS" = "forfeited" ] && pass "Forfeit: $FORFEIT_STATUS" || fail "Forfeit: $FORFEIT"

echo -e "\n=== 8. Wallet After Forfeit (680-100=580) ==="
WALLET_AFTER_FORFEIT=$(curl -s $BASE_URL/wallet \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN" | jq -r '.kash')
[ "$WALLET_AFTER_FORFEIT" = "580" ] && pass "Wallet after forfeit: $580" || fail "Wallet after forfeit: $WALLET_AFTER_FORFEIT"

echo -e "\n=== 9. My Leagues Empty ==="
MY_LEAGUES_AFTER=$(curl -s $BASE_URL/leagues/my \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN")
LEAGUE_COUNT_AFTER=$(echo $MY_LEAGUES_AFTER | jq 'length')
[ "$LEAGUE_COUNT_AFTER" = "0" ] && pass "League deleted" || fail "Leagues still exist: $LEAGUE_COUNT_AFTER"

echo -e "\n=== 10. Forfeit Again (Idempotent - NO double charge) ==="
FORFEIT2=$(curl -s -X POST $BASE_URL/leagues/$LEAGUE_ID/forfeit \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN")
FORFEIT2_STATUS=$(echo $FORFEIT2 | jq -r '.status')
[ "$FORFEIT2_STATUS" = "already_deleted" ] && pass "Idempotent: already_deleted" || fail "Idempotent: $FORFEIT2_STATUS"

echo -e "\n=== 11. Wallet Unchanged (still 580) ==="
WALLET_FINAL=$(curl -s $BASE_URL/wallet \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN" | jq -r '.kash')
[ "$WALLET_FINAL" = "580" ] && pass "No double charge: $580" || fail "Double charged: $WALLET_FINAL"

echo -e "\n=== 12. Create New League ==="
CREATE2=$(curl -s -X POST $BASE_URL/leagues/create \
  -H "Content-Type: application/json" \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"New League","team_count":5,"difficulty":"BEGINNER"}')
LEAGUE_ID2=$(echo $CREATE2 | jq -r '.league_id')
[ "$LEAGUE_ID2" != "null" ] && [ "$LEAGUE_ID2" != "" ] && pass "Create new league: $LEAGUE_ID2" || fail "Create new: $CREATE2"

echo -e "\n=== 13. Test Quick Match ==="
GENERATE=$(curl -s -X POST $BASE_URL/leagues/$LEAGUE_ID2/quick-match/generate \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN")
QUICK_MATCH_ID=$(echo $GENERATE | jq -r '.quick_match_id')
[ "$QUICK_MATCH_ID" != "null" ] && pass "Quick match generated" || fail "Quick match: $GENERATE"

echo -e "\n=== 14. Logout ==="
LOGOUT=$(curl -s -X POST $BASE_URL/auth/logout \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN")
[ "$(echo $LOGOUT | jq -r '.status')" = "logged_out" ] && pass "Logout" || fail "Logout: $LOGOUT"

echo -e "\n=== 15. Token Invalid After Logout ==="
AFTER_LOGOUT=$(curl -s $BASE_URL/wallet \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN")
[ "$(echo $AFTER_LOGOUT | jq -r '.error')" = "Invalid session" ] && pass "Token invalidated" || fail "Token still works: $AFTER_LOGOUT"

echo -e "\n========================================="
echo "  RESULTS: $PASS passed, $FAIL failed"
echo "========================================="
[ $FAIL -eq 0 ] && echo "🎉 ALL TESTS PASSED!" || echo "⚠️ SOME TESTS FAILED"