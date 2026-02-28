#!/bin/bash

# Login
token=$(curl -s -X POST http://127.0.0.1:5050/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@trackcard.com", "password": "admin123"}' | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)

echo "Token: $token"

# 1. Check User Info (to see org)
echo "Checking User Info..."
user_info=$(curl -s -X GET http://127.0.0.1:5050/api/auth/me \
  -H "Authorization: Bearer $token")
echo "User Info: $user_info"

# 2. Create Shipment (Land)
echo "Creating Shipment..."
create_resp=$(curl -s -X POST http://127.0.0.1:5050/api/shipments \
  -H "Authorization: Bearer $token" \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "贵州省铜仁市",
    "destination": "湖北省襄阳市",
    "transport_type": "land",
    "cargo_name": "Visible Test Cargo",
    "org_id": "org-hq"
  }')
echo "Create Resp: $create_resp"

shipment_id=$(echo $create_resp | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Created ID: $shipment_id"

# 3. Get Shipment Details
echo "Getting Shipment Details..."
detail_resp=$(curl -s -X GET http://127.0.0.1:5050/api/shipments/$shipment_id \
  -H "Authorization: Bearer $token")
echo "Detail Resp: $detail_resp"

# 4. List Shipments with org-hq
echo "Listing Shipments (org_id=org-hq)..."
list_resp=$(curl -s -X GET "http://127.0.0.1:5050/api/shipments?org_id=org-hq&limit=10" \
  -H "Authorization: Bearer $token")

# Check if shipment_id is in list
if [[ $list_resp == *"$shipment_id"* ]]; then
  echo "SUCCESS: Shipment found in list."
else
  echo "FAILURE: Shipment NOT found in list."
fi
