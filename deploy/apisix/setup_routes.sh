#!/usr/bin/env bash
# Registers APISIX routes that match the frontend client (api.ts) and k6 scripts.
set -euo pipefail

APISIX_ADMIN_URL="${APISIX_ADMIN_URL:-http://localhost:9180/apisix/admin}"
ADMIN_KEY="${APISIX_ADMIN_KEY:-edd1c9f034335f136f87ad84b625c8f1}"
UPSTREAM="${UPSTREAM_HOST:-host.docker.internal}"
JWT_SECRET="${JWT_SECRET:-super_secret_jwt_signing_key_change_in_production}"

put() {
  local path="$1"
  local body="$2"
  curl -sS -o /tmp/apisix_route.json -w "%{http_code}" -X PUT "${APISIX_ADMIN_URL}${path}" \
    -H "X-API-KEY: ${ADMIN_KEY}" \
    -H "Content-Type: application/json" \
    -d "${body}"
}

echo "Configuring APISIX routes via ${APISIX_ADMIN_URL} (upstream ${UPSTREAM})..."

put "/consumers/urban-prime-user" "{
  \"username\": \"urban-prime-user\",
  \"plugins\": {
    \"jwt-auth\": {
      \"key\": \"urban-prime-jwt\",
      \"secret\": \"${JWT_SECRET}\"
    }
  }
}" >/dev/null || true

put "/routes/1" "{
  \"uri\": \"/auth/*\",
  \"name\": \"auth-service-public\",
  \"methods\": [\"GET\", \"POST\", \"OPTIONS\", \"HEAD\"],
  \"upstream\": { \"type\": \"roundrobin\", \"nodes\": { \"${UPSTREAM}:8080\": 1 } },
  \"plugins\": { \"cors\": {} }
}" >/dev/null

put "/routes/5" "{
  \"uri\": \"/health\",
  \"name\": \"auth-health\",
  \"methods\": [\"GET\", \"HEAD\"],
  \"upstream\": { \"type\": \"roundrobin\", \"nodes\": { \"${UPSTREAM}:8080\": 1 } },
  \"plugins\": { \"cors\": {} }
}" >/dev/null

put "/routes/2" "{
  \"uris\": [\"/api/v1/trips\", \"/api/v1/trips/*\", \"/v1/trips*\"],
  \"name\": \"trip-service-rest\",
  \"methods\": [\"GET\", \"POST\", \"PUT\", \"DELETE\", \"OPTIONS\"],
  \"upstream\": { \"type\": \"roundrobin\", \"nodes\": { \"${UPSTREAM}:8051\": 1 } },
  \"plugins\": { \"cors\": {} }
}" >/dev/null

put "/routes/3" "{
  \"uris\": [\"/api/v1/drivers\", \"/api/v1/drivers/*\", \"/v1/drivers*\"],
  \"name\": \"driver-service-rest\",
  \"methods\": [\"GET\", \"POST\", \"PUT\", \"OPTIONS\"],
  \"upstream\": { \"type\": \"roundrobin\", \"nodes\": { \"${UPSTREAM}:8052\": 1 } },
  \"plugins\": { \"cors\": {} }
}" >/dev/null

put "/routes/4" "{
  \"uris\": [\"/api/v1/location\", \"/api/v1/location/*\", \"/v1/location*\"],
  \"name\": \"location-service-rest\",
  \"methods\": [\"GET\", \"POST\", \"PUT\", \"OPTIONS\"],
  \"upstream\": { \"type\": \"roundrobin\", \"nodes\": { \"${UPSTREAM}:8053\": 1 } },
  \"plugins\": { \"cors\": {} }
}" >/dev/null

put "/routes/6" "{
  \"uri\": \"/api/v1/auth/*\",
  \"name\": \"auth-service-api-prefix\",
  \"methods\": [\"GET\", \"POST\", \"OPTIONS\"],
  \"upstream\": { \"type\": \"roundrobin\", \"nodes\": { \"${UPSTREAM}:8080\": 1 } },
  \"plugins\": {
    \"cors\": {},
    \"proxy-rewrite\": { \"regex_uri\": [\"^/api/v1/auth/(.*)\", \"/auth/\$1\"] }
  }
}" >/dev/null

echo "APISIX routes configured."
