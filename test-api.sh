#!/bin/bash

# API ni test qilish uchun script

echo "Wireguard VPN Client API ni test qilish..."

# Konfiguratsiya faylini o'qish
CONFIG_FILE="/etc/wireguard/server.yaml"

# Konfiguratsiya fayli mavjudligini tekshirish
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Xato: Konfiguratsiya fayli topilmadi: $CONFIG_FILE"
    echo "Iltimos, avval konfiguratsiya faylini yarating:"
    echo "sudo ./wireguard-client-api --create-config"
    exit 1
fi

# API token va portni konfiguratsiya faylidan olish
API_TOKEN=$(grep -A1 "token:" "$CONFIG_FILE" | tail -n1 | awk '{print $2}')
API_PORT=$(grep -A1 "port:" "$CONFIG_FILE" | head -n1 | awk '{print $2}')

# Agar port topilmagan bo'lsa, default qiymatni ishlatish
if [ -z "$API_PORT" ]; then
    API_PORT=8080
fi

# Funksiya: API ga so'rov yuborish
function call_api() {
    local method=$1
    local endpoint=$2
    local data=$3

    if [ -z "$data" ]; then
        curl -s -X "$method" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $API_TOKEN" \
            "http://localhost:$API_PORT$endpoint"
    else
        curl -s -X "$method" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $API_TOKEN" \
            -d "$data" \
            "http://localhost:$API_PORT$endpoint"
    fi
}

echo "=== Wireguard VPN Client API Test ==="
echo "API port: $API_PORT"
echo

# Normal client yaratish
echo "1. Normal client yaratish..."
NORMAL_CLIENT=$(call_api "POST" "/api/client" '{"description": "Normal client", "life_time": 2592000, "type": "normal"}')
echo "$NORMAL_CLIENT" | jq .
echo

# Normal client konfiguratsiyasini saqlash
echo "$NORMAL_CLIENT" | jq -r .config >normal-client.conf
echo "Normal client konfiguratsiyasi 'normal-client.conf' fayliga saqlandi"
echo

# VIP client yaratish
echo "2. VIP client yaratish..."
VIP_CLIENT=$(call_api "POST" "/api/client" '{"description": "VIP client", "life_time": 31536000, "type": "vip"}')
echo "$VIP_CLIENT" | jq .
echo

# VIP client konfiguratsiyasini saqlash
echo "$VIP_CLIENT" | jq -r .config >vip-client.conf
echo "VIP client konfiguratsiyasi 'vip-client.conf' fayliga saqlandi"
echo

# Barcha clientlarni olish
echo "3. Barcha clientlarni olish..."
CLIENTS=$(call_api "GET" "/api/clients")
echo "$CLIENTS" | jq .
echo

# Birinchi clientni olish
FIRST_CLIENT_ID=$(echo "$CLIENTS" | jq -r '.[0].ID')
echo "4. Client ma'lumotlarini olish (ID: $FIRST_CLIENT_ID)..."
CLIENT=$(call_api "GET" "/api/client/$FIRST_CLIENT_ID")
echo "$CLIENT" | jq .
echo

# Client life_time vaqtini olish
echo "5. Client life_time vaqtini olish (ID: $FIRST_CLIENT_ID)..."
CLIENT_LIFETIME=$(call_api "GET" "/api/client/$FIRST_CLIENT_ID/lifetime")
echo "$CLIENT_LIFETIME" | jq .
echo

# Client life_time vaqtini yangilash
echo "6. Client life_time vaqtini yangilash (ID: $FIRST_CLIENT_ID)..."
UPDATED_LIFETIME=$(call_api "PUT" "/api/client/$FIRST_CLIENT_ID/lifetime" '{"life_time": 604800}')
echo "$UPDATED_LIFETIME" | jq .
echo

# Client traffic ma'lumotlarini olish
echo "7. Client traffic ma'lumotlarini olish (ID: $FIRST_CLIENT_ID)..."
CLIENT_TRAFFIC=$(call_api "GET" "/api/client/$FIRST_CLIENT_ID/traffic")
echo "$CLIENT_TRAFFIC" | jq .
echo

# Barcha clientlar traffic ma'lumotlarini olish
echo "8. Barcha clientlar traffic ma'lumotlarini olish..."
ALL_CLIENTS_TRAFFIC=$(call_api "GET" "/api/clients/traffic")
echo "$ALL_CLIENTS_TRAFFIC" | jq .
echo

# Server holatini olish
echo "9. Server holatini olish..."
SERVER_STATUS=$(call_api "GET" "/api/server/status")
echo "$SERVER_STATUS" | jq .
echo

echo "10. Health holatini olish..."
HEALTH=$(call_api "GET" "/api/health")
echo "$HEALTH" | jq .
echo

echo "=== Test yakunlandi ==="
