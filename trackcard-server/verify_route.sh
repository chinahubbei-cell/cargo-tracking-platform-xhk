#!/bin/bash
# Login to get token
curl -s -X POST http://localhost:5051/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@trackcard.com", "password":"admin123","type":"email"}' > login_response.json

TOKEN=$(grep -o '"token":"[^"]*"' login_response.json | cut -d'"' -f4)

# Get Route
curl -s -X GET http://localhost:5051/api/shipments/260201000002/route \
  -H "Authorization: Bearer $TOKEN" > route_response.json

# Count points
POINTS=$(grep -o '"latitude":' route_response.json | wc -l)
echo "Found points: $POINTS"
