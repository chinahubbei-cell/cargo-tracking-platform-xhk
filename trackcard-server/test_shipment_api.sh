#!/bin/bash

# 测试运单详情API
echo "=== 测试运单详情 API ===="
echo "URL: http://127.0.0.1:5051/api/shipments/260211000001"
echo ""

# 先尝试获取登录token
echo "1. 获取登录token..."
LOGIN_RESPONSE=$(curl -s -X POST http://127.0.0.1:5051/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@trackcard.com","password":"$2a$10$fUco6LKkJbeMhW0lVPc3feyybyzqPH/KpzJqIRpiSDrFn.7FM5UZu"}')

echo "$LOGIN_RESPONSE" | python3 -m json.tool | head -5

# 提取token
TOKEN=$(echo "$LOGIN_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('data', {}).get('token', ''))" 2>/dev/null)

if [ -z "$TOKEN" ]; then
    echo "❌ 获取token失败"
    exit 1
fi

echo "Token: ${TOKEN:0:50}..."
echo ""

# 调用运单详情API
echo "2. 调用运单详情API..."
API_RESPONSE=$(curl -s -X GET "http://127.0.0.1:5051/api/shipments/260211000001" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json")

echo "$API_RESPONSE" | python3 -m json.tool
echo ""
echo "=== 检查关键字段 ===="
echo "$API_RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
shipment = data.get('data', {})
print('运单ID:', shipment.get('id'))
print('device_id:', shipment.get('device_id'))
print('device对象:', 'device' in shipment)
if 'device' in shipment:
    device = shipment['device']
    print('  device.id:', device.get('id'))
    print('  device.external_device_id:', device.get('external_device_id'))
    print('  device.name:', device.get('name'))
else:
    print('  device对象不存在')
"
