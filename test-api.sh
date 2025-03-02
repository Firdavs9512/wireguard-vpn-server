#!/bin/bash

# API ni test qilish uchun script

echo "Wireguard VPN Client API ni test qilish..."
echo "POST /api/client endpointiga so'rov yuborilmoqda..."

# API ga so'rov yuborish
response=$(curl -s -X POST http://localhost:8080/api/client)

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
    echo "Test muvaffaqiyatli o'tdi!"
    
    # Config faylini saqlash
    echo "$response" | jq -r '.config' > client-config.conf
    echo "Client konfiguratsiyasi 'client-config.conf' fayliga saqlandi."
else
    echo "Xatolik: API javobida 'config' yoki 'data' maydonlari topilmadi."
    exit 1
fi 