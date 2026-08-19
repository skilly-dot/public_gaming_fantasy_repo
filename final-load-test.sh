#!/bin/bash
# fixed-load-test.sh

BASE_URL="http://localhost:8080/api/v1"
APP_KEY="dev_secret_key_123"

# Read tokens directly from file
TOKEN_1=$(grep "TOKEN_1=" /tmp/load_tokens.txt | cut -d'"' -f2)
TOKEN_2=$(grep "TOKEN_2=" /tmp/load_tokens.txt | cut -d'"' -f2)
TOKEN_3=$(grep "TOKEN_3=" /tmp/load_tokens.txt | cut -d'"' -f2)
TOKEN_4=$(grep "TOKEN_4=" /tmp/load_tokens.txt | cut -d'"' -f2)
TOKEN_5=$(grep "TOKEN_5=" /tmp/load_tokens.txt | cut -d'"' -f2)

LEAGUE_1=$(grep "LEAGUE_1=" /tmp/load_leagues.txt | cut -d'"' -f2)
LEAGUE_2=$(grep "LEAGUE_2=" /tmp/load_leagues.txt | cut -d'"' -f2)
LEAGUE_3=$(grep "LEAGUE_3=" /tmp/load_leagues.txt | cut -d'"' -f2)
LEAGUE_4=$(grep "LEAGUE_4=" /tmp/load_leagues.txt | cut -d'"' -f2)
LEAGUE_5=$(grep "LEAGUE_5=" /tmp/load_leagues.txt | cut -d'"' -f2)

echo "Token 1: $TOKEN_1"
echo "League 1: $LEAGUE_1"

# Verify token works FIRST
echo -e "\n=== Verify Token ==="
curl -s "$BASE_URL/wallet" \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN_1" | jq .

# Single user test
echo -e "\n=== Single User: Wallet ==="
hey -z 10s -c 1 -q 0.125 "$BASE_URL/wallet" \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN_1" 2>&1 | tail -5

# Results test
echo -e "\n=== Single User: Results ==="
hey -z 10s -c 1 -q 0.033 "$BASE_URL/leagues/$LEAGUE_1/results" \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN_1" 2>&1 | tail -5

# Daily data
echo -e "\n=== Single User: Daily ==="
hey -z 10s -c 1 -q 0.05 "$BASE_URL/leagues/$LEAGUE_1/daily" \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN_1" 2>&1 | tail -5

# Admin bets
echo -e "\n=== Single User: Admin Bets ==="
hey -z 10s -c 1 -q 0.016 "$BASE_URL/bets/admin-matches" \
  -H "X-App-Key: $APP_KEY" \
  -H "Authorization: Bearer $TOKEN_1" 2>&1 | tail -5