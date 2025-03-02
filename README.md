# Wireguard VPN Client Yaratuvchi API

Bu loyiha Golang yordamida Wireguard VPN clientlarini yaratish uchun API server taqdim etadi.

## Imkoniyatlar

- `/api/client` endpointiga POST so'rov yuborish orqali yangi Wireguard client yaratish
- Har bir client uchun unikal IP manzil va kalitlar yaratish
- Wireguard konfiguratsiya faylini va uning ma'lumotlarini JSON formatida qaytarish
- Server konfiguratsiyasiga yangi peerlarni avtomatik qo'shish
- Clientlarni SQLite databasega saqlash va boshqarish
- Clientlarni ko'rish, o'chirish va boshqarish uchun API endpointlar

## Talablar

- Go 1.20 yoki undan yuqori versiya
- Wireguard o'rnatilgan bo'lishi kerak
- `/etc/wireguard/` papkasida server konfiguratsiyasi va kalitlar mavjud bo'lishi kerak
- Dasturni root huquqlari bilan ishga tushirish kerak (server konfiguratsiyasini o'zgartirish uchun)
- Test script uchun `jq` o'rnatilgan bo'lishi kerak

## O'rnatish

```bash
# Loyihani klonlash
git clone https://github.com/username/wireguard-vpn-client-creater.git
cd wireguard-vpn-client-creater

# Dependencylarni o'rnatish
go mod tidy

# Dasturni ishga tushirish (Makefile yordamida)
make run
```

## Makefile buyruqlari

Loyihada quyidagi Makefile buyruqlari mavjud:

- `make run` - Dasturni ishga tushirish
- `make build` - Dasturni build qilish
- `make start` - Dasturni build qilib, ishga tushirish
- `make install` - Dasturni build qilib, systemd service sifatida o'rnatish
- `make uninstall` - Systemd serviceni o'chirish
- `make clean` - Build fayllarini tozalash

## API ni test qilish

Loyihada API ni test qilish uchun `test-api.sh` scripti mavjud:

```bash
# Scriptga ishga tushirish huquqini berish
chmod +x test-api.sh

# Scriptni ishga tushirish
./test-api.sh
```

Bu script API ga so'rov yuborib, natijani tekshiradi va client konfiguratsiyasini `client-config.conf` fayliga saqlaydi.

## API Qo'llanmasi

### Yangi client yaratish

**So'rov:**

```
POST /api/client
```

**Request body (ixtiyoriy):**

```json
{
  "description": "Client tavsifi",
  "life_time": 30,
  "type": "normal"
}
```

**Javob:**

```json
{
  "config": "Wireguard konfiguratsiya fayli matni",
  "data": {
    "id": 1,
    "description": "Client tavsifi",
    "private_key": "client_private_key",
    "public_key": "client_public_key",
    "address": "10.0.0.2/32",
    "endpoint": "server_endpoint:51820",
    "type": "normal",
    "life_time": 30,
    "expires_at": "2023-12-31T23:59:59Z"
  }
}
```

### Barcha clientlarni olish

**So'rov:**

```
GET /api/clients
```

**Javob:**

```json
[
  {
    "ID": 1,
    "CreatedAt": "2023-12-01T12:00:00Z",
    "UpdatedAt": "2023-12-01T12:00:00Z",
    "DeletedAt": null,
    "Description": "Normal client",
    "PrivateKey": "client_private_key",
    "PublicKey": "client_public_key",
    "Address": "10.0.0.2/32",
    "Type": "normal",
    "LifeTime": 30,
    "ExpiresAt": "2023-12-31T23:59:59Z"
  },
  {
    "ID": 2,
    "CreatedAt": "2023-12-01T12:30:00Z",
    "UpdatedAt": "2023-12-01T12:30:00Z",
    "DeletedAt": null,
    "Description": "VIP client",
    "PrivateKey": "client_private_key",
    "PublicKey": "client_public_key",
    "Address": "10.0.0.3/32",
    "Type": "vip",
    "LifeTime": 365,
    "ExpiresAt": "2024-12-01T12:30:00Z"
  }
]
```

### Client ma'lumotlarini olish

**So'rov:**

```
GET /api/client/:id
```

**Javob:**

```json
{
  "ID": 1,
  "CreatedAt": "2023-12-01T12:00:00Z",
  "UpdatedAt": "2023-12-01T12:00:00Z",
  "DeletedAt": null,
  "Description": "Normal client",
  "PrivateKey": "client_private_key",
  "PublicKey": "client_public_key",
  "Address": "10.0.0.2/32",
  "Type": "normal",
  "LifeTime": 30,
  "ExpiresAt": "2023-12-31T23:59:59Z"
}
```

### Clientni o'chirish

**So'rov:**

```
DELETE /api/client/:id
```

**Javob:**

```json
{
  "message": "Client muvaffaqiyatli o'chirildi"
}
```

## Texnik tafsilotlar

- Server konfiguratsiyasiga yangi peerlar `wg` va `wg-quick` buyruqlari orqali qo'shiladi
- Har bir client uchun unikal IP manzil 10.7.0.2 - 10.7.0.251 oralig'ida generatsiya qilinadi
- Client ma'lumotlari SQLite databaseda saqlanadi (`./data/wireguard.db`)
- Database GORM ORM orqali boshqariladi
- Clientlar uchun "normal" va "vip" turlari mavjud
- Clientlar uchun amal qilish muddati belgilanishi mumkin (soniyalarda)

## Xavfsizlik eslatmasi

Bu dastur server konfiguratsiyasini o'zgartirish uchun root huquqlariga ega bo'lishi kerak. Ishlab chiqarish muhitida qo'shimcha xavfsizlik choralarini ko'rish tavsiya etiladi. 

Parametrlar:
- `description` - Client tavsifi (majburiy)
- `life_time` - Clientning amal qilish muddati (soniyalarda). 0 qiymati cheksiz muddatni bildiradi (ixtiyoriy, standart qiymati: 0)
- `type` - Client turi: "normal" yoki "vip" (ixtiyoriy, standart qiymati: "normal") 