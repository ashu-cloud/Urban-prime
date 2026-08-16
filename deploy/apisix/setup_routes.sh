#!/bin/bash
# APISIX Route Setup Script
# Registers all microservice routes and configures rate limits & JWT validation in Apache APISIX

APISIX_ADMIN_URL="http://localhost:9090/apisix/admin"
ADMIN_KEY="edd1c9f034335f136f87ad84b625c8f1"

echo "Configuring Apache APISIX Routes..."

# 1. Unauthenticated Auth Routes (Login, Register, Refresh)
curl -i -X PUT "${APISIX_ADMIN_URL}/routes/1" \
  -H "X-API-KEY: ${ADMIN_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "uri": "/auth/*",
    "name": "auth-service-public",
    "methods": ["POST", "OPTIONS"],
    "upstream": {
      "type": "roundrobin",
      "nodes": {
        "host.docker.internal:8080": 1
      }
    },
    "plugins": {
      "cors": {},
      "limit-req": {
        "rate": 10,
        "burst": 5,
        "key_type": "var",
        "key": "remote_addr",
        "rejected_code": 429
      }
    }
  }'

# 2. Authenticated Trip Service Routes
curl -i -X PUT "${APISIX_ADMIN_URL}/routes/2" \
  -H "X-API-KEY: ${ADMIN_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "uri": "/v1/trips*",
    "name": "trip-service-protected",
    "methods": ["GET", "POST", "PUT", "DELETE"],
    "upstream": {
      "type": "roundrobin",
      "nodes": {
        "host.docker.internal:50051": 1
      }
    },
    "plugins": {
      "cors": {},
      "jwt-auth": {},
      "limit-req": {
        "rate": 30,
        "burst": 10,
        "key_type": "var",
        "key": "remote_addr",
        "rejected_code": 429
      }
    }
  }'

# 3. Authenticated Driver Service Routes
curl -i -X PUT "${APISIX_ADMIN_URL}/routes/3" \
  -H "X-API-KEY: ${ADMIN_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "uri": "/v1/drivers*",
    "name": "driver-service-protected",
    "methods": ["GET", "POST", "PUT"],
    "upstream": {
      "type": "roundrobin",
      "nodes": {
        "host.docker.internal:50052": 1
      }
    },
    "plugins": {
      "cors": {},
      "jwt-auth": {},
      "limit-req": {
        "rate": 60,
        "burst": 20,
        "key_type": "var",
        "key": "remote_addr",
        "rejected_code": 429
      }
    }
  }'

# 4. Authenticated Location Service GPS Firehose Route (High Throughput Rate Limit)
curl -i -X PUT "${APISIX_ADMIN_URL}/routes/4" \
  -H "X-API-KEY: ${ADMIN_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "uri": "/v1/location*",
    "name": "location-service-protected",
    "methods": ["POST", "PUT"],
    "upstream": {
      "type": "roundrobin",
      "nodes": {
        "host.docker.internal:50053": 1
      }
    },
    "plugins": {
      "cors": {},
      "jwt-auth": {},
      "limit-req": {
        "rate": 200,
        "burst": 50,
        "key_type": "var",
        "key": "remote_addr",
        "rejected_code": 429
      }
    }
  }'

echo "APISIX Routes configured successfully!"
