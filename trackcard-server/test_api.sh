#!/bin/bash
# 测试API - 先登录获取token
echo "=== Testing Login ==="
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:5051/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}')
echo $LOGIN_RESPONSE | head -100

# 提取token
TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
echo ""
echo "Token: $TOKEN"

echo ""
echo "=== Testing Ports API ==="
curl -s "http://localhost:5051/api/ports?page=1&page_size=5" \
  -H "Authorization: Bearer $TOKEN" | head -200

echo ""
echo "=== Testing Airports API ==="
curl -s "http://localhost:5051/api/airports?page=1&page_size=5" \
  -H "Authorization: Bearer $TOKEN" | head -200
