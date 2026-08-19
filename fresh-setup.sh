#!/bin/bash
# fresh-setup.sh

BASE_URL="http://localhost:8080/api/v1"
APP_KEY="dev_secret_key_123"

# Clear old files
> /tmp/load_tokens.txt
> /tmp/load_leagues.txt

echo "=== Creating Fresh Users ==="

for i in 1 2 3 4 5 6 7 8 9 10; do
  # Unique username with timestamp
  USERNAME="lt_$(date +%s)_$i"
  
  # Register
  REGISTER=$(curl -s -X POST $BASE_URL/auth/register \
    -H "Content-Type: application/json" \
    -H "X-App-Key: $APP_KEY" \
    -d "{\"username\":\"$USERNAME\",\"password\":\"password123\",\"gamename\":\"LoadUser$i\"}")
  
  TOKEN=$(echo $REGISTER | jq -r '.session_id')
  USER_ID=$(echo $REGISTER | jq -r '.user_id')
  
  if [ "$TOKEN" != "null" ] && [ "$TOKEN" != "" ] && [ "$TOKEN" != "null" ]; then
    echo "export TOKEN_$i=\"$TOKEN\"" >> /tmp/load_tokens.txt
    echo "export USER_$i=\"$USER_ID\"" >> /tmp/load_users.txt
    
    # Verify token works
    WALLET=$(curl -s $BASE_URL/wallet \
      -H "X-App-Key: $APP_KEY" \
      -H "Authorization: Bearer $TOKEN")
    
    if echo "$WALLET" | jq -e '.kash' > /dev/null 2>&1; then
      echo "User $i: $USERNAME - TOKEN OK"
      
      # Create league
      CREATE=$(curl -s -X POST $BASE_URL/leagues/create \
        -H "Content-Type: application/json" \
        -H "X-App-Key: $APP_KEY" \
        -H "Authorization: Bearer $TOKEN" \
        -d "{\"name\":\"Load League $i\",\"team_count\":5,\"difficulty\":\"BEGINNER\"}")
      
      LEAGUE_ID=$(echo $CREATE | jq -r '.league_id')
      if [ "$LEAGUE_ID" != "null" ] && [ "$LEAGUE_ID" != "" ]; then
        echo "export LEAGUE_$i=\"$LEAGUE_ID\"" >> /tmp/load_leagues.txt
        echo "  League: $LEAGUE_ID - OK"
      else
        echo "  League creation failed: $CREATE"
      fi
    else
      echo "User $i: TOKEN INVALID: $WALLET"
    fi
  else
    echo "User $i: REGISTER FAILED: $REGISTER"
  fi
done

echo -e "\n=== RESULTS ==="
echo "Tokens saved:"
cat /tmp/load_tokens.txt
echo -e "\nLeagues saved:"
cat /tmp/load_leagues.txt