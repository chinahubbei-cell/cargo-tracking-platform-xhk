#!/bin/bash

# 测试设备购买订单创建 API
# 需要先登录获取 token

echo "=== 测试设备购买订单 API ==="
echo ""

# 1. 先登录获取 token
echo "1. 登录获取 token..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:5052/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@trackcard.com",
    "password": "admin123"
  }')

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ] || [ "$TOKEN" == "null" ]; then
  echo "❌ 登录失败，无法获取 token"
  echo "响应: $LOGIN_RESPONSE"
  exit 1
fi

echo "✅ 登录成功"
echo ""

# 2. 测试创建设备购买订单
echo "2. 测试创建设备购买订单..."
PURCHASE_RESPONSE=$(curl -s -X POST http://localhost:5052/api/trade/orders/purchase \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "contact_name": "测试联系人",
    "phone": "13800138000",
    "service_years": 1,
    "items": [
      {
        "product_name": "X3 追货版",
        "qty": 1,
        "unit_price": 299,
        "service_price": 100
      }
    ]
  }')

echo "响应: $PURCHASE_RESPONSE"
echo ""

# 3. 检查是否成功
if echo "$PURCHASE_RESPONSE" | grep -q '"success":true'; then
  echo "✅ 订单创建成功"

  # 提取订单ID
  ORDER_ID=$(echo $PURCHASE_RESPONSE | grep -o '"id":"[^"]*' | cut -d'"' -f4 | head -1)
  echo "订单ID: $ORDER_ID"
else
  echo "❌ 订单创建失败"
fi

echo ""
echo "=== 测试完成 ==="
