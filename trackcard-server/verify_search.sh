#!/bin/bash
# Login to get token
curl -s -X POST http://localhost:5051/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@trackcard.com", "password":"admin123","type":"email"}' > login_response.json

TOKEN=$(grep -o '"token":"[^"]*"' login_response.json | cut -d'"' -f4)

# Search for precise ID
echo "Searching for 260128000004..."
time curl -s -X GET "http://localhost:5051/api/shipments?search=260128000004" \
  -H "Authorization: Bearer $TOKEN" > search_response.json

# Check if ID is in response
if grep -q "260128000004" search_response.json; then
  echo "SUCCESS: Found shipment"
else
  echo "FAILURE: Shipment not found"
  cat search_response.json
fi
