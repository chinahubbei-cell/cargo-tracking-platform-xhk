#!/bin/bash

# Login to get token
login_resp=$(curl -v -X POST http://127.0.0.1:5050/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@trackcard.com", "password": "admin123"}')

token=$(echo $login_resp | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$token" ]; then
    echo "Login failed. Response: $login_resp"
    # Try default password if the above fails (commented out for now, assuming ChangeMe123! based on common practices or previous sessions)
    # exit 1
fi

echo "Got token: $token"

# Create a test shipment
response=$(curl -s -X POST http://localhost:5050/api/shipments \
  -H "Authorization: Bearer $token" \
  -H "Content-Type: application/json" \
  -d '{
    "transport_type": "fcl",
    "origin": "Shenzhen, China",
    "destination": "Los Angeles, USA",
    "origin_address": "Shenzhen Port",
    "dest_address": "Port of Los Angeles",
    "weight": 1000,
    "volume": 20,
    "cargo_name": "Electronics",
    "sender_name": "Tech Corp",
    "receiver_name": "US Retailer"
  }')

echo "Create Response: $response"

shipment_id=$(echo $response | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$shipment_id" ]; then
    echo "Failed to create shipment"
    exit 1
fi

echo "Created Shipment ID: $shipment_id"

# Wait a moment for async processing (though stage creation is synchronous in the handler)
sleep 2

# Check shipment stages
echo "Checking shipment stages..."
# We can use sqlite3 to query the database directly as it is a local file
sqlite3 trackcard.db "SELECT stage_code, status, port_code, cost FROM shipment_stages WHERE shipment_id = '$shipment_id' ORDER BY stage_order;"
