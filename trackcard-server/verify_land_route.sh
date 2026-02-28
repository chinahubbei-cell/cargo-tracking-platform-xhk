#!/bin/bash

# Login and get token
echo "Logging in..."
login_resp=$(curl -s -X POST http://127.0.0.1:8000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@trackcard.com", "password": "admin123"}')

token=$(echo $login_resp | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$token" ]; then
  echo "Login failed"
  echo $login_resp
  exit 1
fi
echo "Token: $token"

# Create Land Shipment
echo "Creating Land Shipment..."
shipment_resp=$(curl -s -X POST http://127.0.0.1:8000/api/shipments \
  -H "Authorization: Bearer $token" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "贵州省铜仁市松桃苗族自治县大兴街道白岩社区火连坳",
    "destination": "湖北省襄阳市樊城区新华北路27-9号附近",
    "transport_type": "land",
    "cargo_name": "Test Cargo",
    "sender_name": "Zhang San",
    "sender_phone": "123456",
    "receiver_name": "Li Si",
    "receiver_phone": "654321",
    "pieces": 100,
    "weight": 5000,
    "volume": 20
  }')

echo "Shipment Response: $shipment_resp"

shipment_id=$(echo $shipment_resp | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$shipment_id" ]; then
  echo "Shipment creation failed"
  exit 1
fi

echo "Created Shipment ID: $shipment_id"

# Wait for stage generation (async?)
# Actually stage generation is synchronous in Create handler, but we can verify via GET stages endpoint.
echo "Checking Stages..."
stages_resp=$(curl -s -X GET http://127.0.0.1:8000/api/shipments/$shipment_id/stages \
  -H "Authorization: Bearer $token")

echo "Stages Response: $stages_resp"

# Check if stages exist and have correct info
count=$(echo $stages_resp | grep -o "stage_code" | wc -l)
echo "Stage Count: $count"

if [ "$count" -gt 0 ]; then
    echo "SUCCESS: Land shipment created with stages."
else
    echo "FAILURE: Stages not created."
fi
