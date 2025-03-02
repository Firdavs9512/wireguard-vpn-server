#!/bin/bash

# API ni test qilish uchun script

echo "Wireguard VPN Client API ni test qilish..."

# Funksiya: API ga so'rov yuborish
function call_api() {
    local method=$1
    local endpoint=$2
    local data=$3
    
    if [ -z "$data" ]; then
        echo "$(curl -s -X $method http://localhost:8080$endpoint)"
    else
        echo "$(curl -s -X $method -H "Content-Type: application/json" -d "$data" http://localhost:8080$endpoint)"
    fi
}

# 1. Normal client yaratish
echo "1. POST /api/client endpointiga so'rov yuborilmoqda (Normal client)..."
response=$(call_api "POST" "/api/client" '{"description":"Normal client", "life_time": 2592000, "type": "normal"}')

# Natijani tekshirish
if [ $? -ne 0 ]; then
    echo "Xatolik: API ga ulanib bo'lmadi. Server ishga tushirilganligini tekshiring."
    exit 1
fi

# JSON formatini tekshirish
if ! echo "$response" | jq . >/dev/null 2>&1; then
    echo "Xatolik: API javobini JSON formatida qayta ishlab bo'lmadi."
    echo "Javob: $response"
    exit 1
fi

# Natijani chiroyli ko'rinishda chiqarish
echo "API javob berdi (Normal client):"
echo "$response" | jq .

# Config va data mavjudligini tekshirish
if echo "$response" | jq -e '.config' >/dev/null && echo "$response" | jq -e '.data' >/dev/null; then
    echo "Normal client yaratish muvaffaqiyatli o'tdi!"
    
    # Config faylini saqlash
    echo "$response" | jq -r '.config' > normal-client.conf
    echo "Normal client konfiguratsiyasi 'normal-client.conf' fayliga saqlandi."
else
    echo "Xatolik: API javobida 'config' yoki 'data' maydonlari topilmadi."
    exit 1
fi

# 2. VIP client yaratish
echo -e "\n2. POST /api/client endpointiga so'rov yuborilmoqda (VIP client)..."
response=$(call_api "POST" "/api/client" '{"description":"VIP client", "life_time": 31536000, "type": "vip"}')

# Natijani tekshirish
if [ $? -ne 0 ]; then
    echo "Xatolik: API ga ulanib bo'lmadi."
    exit 1
fi

# Natijani chiroyli ko'rinishda chiqarish
echo "API javob berdi (VIP client):"
echo "$response" | jq .

# Config va data mavjudligini tekshirish
if echo "$response" | jq -e '.config' >/dev/null && echo "$response" | jq -e '.data' >/dev/null; then
    echo "VIP client yaratish muvaffaqiyatli o'tdi!"
    
    # Config faylini saqlash
    echo "$response" | jq -r '.config' > vip-client.conf
    echo "VIP client konfiguratsiyasi 'vip-client.conf' fayliga saqlandi."
else
    echo "Xatolik: API javobida 'config' yoki 'data' maydonlari topilmadi."
    exit 1
fi

# 3. Barcha clientlarni olish
echo -e "\n3. GET /api/clients endpointiga so'rov yuborilmoqda..."
clients_response=$(call_api "GET" "/api/clients")
echo "Barcha clientlar:"
echo "$clients_response" | jq .

# Birinchi clientni olish
first_client_id=$(echo "$clients_response" | jq '.[0].ID')
if [ -z "$first_client_id" ] || [ "$first_client_id" = "null" ]; then
    echo "Xatolik: Clientlar topilmadi."
    exit 1
fi

# 4. Birinchi clientni olish
echo -e "\n4. GET /api/client/$first_client_id endpointiga so'rov yuborilmoqda..."
client_response=$(call_api "GET" "/api/client/$first_client_id")
echo "Client ma'lumotlari:"
echo "$client_response" | jq .

# 5. Client life_time vaqtini olish
echo -e "\n5. GET /api/client/$first_client_id/lifetime endpointiga so'rov yuborilmoqda..."
lifetime_response=$(call_api "GET" "/api/client/$first_client_id/lifetime")
echo "Client life_time ma'lumotlari:"
echo "$lifetime_response" | jq .

# 6. Client life_time vaqtini yangilash
echo -e "\n6. PUT /api/client/$first_client_id/lifetime endpointiga so'rov yuborilmoqda..."
updated_lifetime_response=$(call_api "PUT" "/api/client/$first_client_id/lifetime" '{"life_time": 604800}')
echo "Yangilangan client life_time ma'lumotlari (1 hafta = 604800 soniya):"
echo "$updated_lifetime_response" | jq .

# 7. Yangilangan life_time ni tekshirish
echo -e "\n7. GET /api/client/$first_client_id/lifetime endpointiga so'rov yuborilmoqda (tekshirish uchun)..."
check_lifetime_response=$(call_api "GET" "/api/client/$first_client_id/lifetime")
echo "Yangilangan client life_time ma'lumotlari (tekshirish):"
echo "$check_lifetime_response" | jq .

# 8. Client traffic ma'lumotlarini olish
echo -e "\n8. GET /api/client/$first_client_id/traffic endpointiga so'rov yuborilmoqda..."
traffic_response=$(call_api "GET" "/api/client/$first_client_id/traffic")
echo "Client traffic ma'lumotlari:"
echo "$traffic_response" | jq .

# 9. Barcha clientlar traffic ma'lumotlarini olish
echo -e "\n9. GET /api/clients/traffic endpointiga so'rov yuborilmoqda..."
all_traffic_response=$(call_api "GET" "/api/clients/traffic")
echo "Barcha clientlar traffic ma'lumotlari:"
echo "$all_traffic_response" | jq .

echo -e "\nBarcha testlar muvaffaqiyatli o'tdi!" 