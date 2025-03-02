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

# Yangi client yaratish
echo "1. POST /api/client endpointiga so'rov yuborilmoqda..."
response=$(call_api "POST" "/api/client" '{"description":"Test client"}')

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
echo "API javob berdi:"
echo "$response" | jq .

# Config va data mavjudligini tekshirish
if echo "$response" | jq -e '.config' >/dev/null && echo "$response" | jq -e '.data' >/dev/null; then
    echo "Yangi client yaratish muvaffaqiyatli o'tdi!"
    
    # Config faylini saqlash
    echo "$response" | jq -r '.config' > client-config.conf
    echo "Client konfiguratsiyasi 'client-config.conf' fayliga saqlandi."
else
    echo "Xatolik: API javobida 'config' yoki 'data' maydonlari topilmadi."
    exit 1
fi

# Barcha clientlarni olish
echo -e "\n2. GET /api/clients endpointiga so'rov yuborilmoqda..."
clients_response=$(call_api "GET" "/api/clients")
echo "Barcha clientlar:"
echo "$clients_response" | jq .

# Birinchi clientni olish
first_client_id=$(echo "$clients_response" | jq '.[0].ID')
if [ -z "$first_client_id" ] || [ "$first_client_id" = "null" ]; then
    echo "Xatolik: Clientlar topilmadi."
    exit 1
fi

echo -e "\n3. GET /api/client/$first_client_id endpointiga so'rov yuborilmoqda..."
client_response=$(call_api "GET" "/api/client/$first_client_id")
echo "Client ma'lumotlari:"
echo "$client_response" | jq .

echo -e "\nBarcha testlar muvaffaqiyatli o'tdi!" 