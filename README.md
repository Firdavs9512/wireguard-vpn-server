# Wireguard VPN Client Yaratuvchi API

Bu loyiha Golang yordamida Wireguard VPN clientlarini yaratish uchun API server taqdim etadi.

## Imkoniyatlar

- `/api/client` endpointiga POST so'rov yuborish orqali yangi Wireguard client yaratish
- Har bir client uchun unikal IP manzil va kalitlar yaratish
- Wireguard konfiguratsiya faylini va uning ma'lumotlarini JSON formatida qaytarish
- Server konfiguratsiyasiga yangi peerlarni avtomatik qo'shish
- Clientlarni SQLite databasega saqlash va boshqarish
- Clientlarni ko'rish, o'chirish va boshqarish uchun API endpointlar
- Client life_time vaqtini olish va yangilash uchun API endpointlar
- Client traffic ma'lumotlarini olish uchun API endpointlar
- Server holati va statistikasini olish uchun API endpoint

## Talablar

- Go 1.20 yoki undan yuqori versiya
- Wireguard o'rnatilgan bo'lishi kerak
- `/etc/wireguard/` papkasida server konfiguratsiyasi va kalitlar mavjud bo'lishi kerak
- Dasturni root huquqlari bilan ishga tushirish kerak (server konfiguratsiyasini o'zgartirish uchun)
- Test script uchun `jq` o'rnatilgan bo'lishi kerak
- `wg-json` buyrug'i o'rnatilgan bo'lishi kerak (traffic ma'lumotlarini olish uchun)

## O'rnatish

```bash
# Loyihani klonlash
git clone https://github.com/username/wireguard-vpn-client-creater.git
cd wireguard-vpn-client-creater

# Dependencylarni o'rnatish
go mod tidy

# Default konfiguratsiya faylini yaratish
sudo ./wireguard-client-api --create-config

# Konfiguratsiya faylini tahrirlash
sudo nano /etc/wireguard/server.yaml

# Dasturni ishga tushirish (Makefile yordamida)
make run
```

## Konfiguratsiya fayli

Dastur `/etc/wireguard/server.yaml` faylidan konfiguratsiya ma'lumotlarini oladi. Bu faylni quyidagi buyruq bilan yaratish mumkin:

```bash
sudo ./wireguard-client-api --create-config
```

Konfiguratsiya fayli formati:

```yaml
server:
  ip: 192.168.1.1 # Server IP manzili
  port: 51820 # Wireguard port
  interface: wg0 # Wireguard interface nomi
  debug: false # Debug rejimi
api:
  port: 8080 # API port
  token: secure-token # API token (xavfsizlik uchun o'zgartiring)
wireguard:
  dns: 1.1.1.1, 8.8.8.8 # DNS serverlari
  allowed_ips: 0.0.0.0/0, ::/0 # Ruxsat berilgan IP manzillar
  persistent_keepalive: 25 # Persistent keepalive vaqti
  server_public_key_path: /etc/wireguard/server_public.key # Server public key fayli
database:
  path: ./data/wireguard.db # Database fayli yo'li
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
sudo ./test-api.sh
```

Bu script API ga so'rov yuborib, natijani tekshiradi va client konfiguratsiyasini `normal-client.conf` va `vip-client.conf` fayllariga saqlaydi.

## API Qo'llanmasi

Barcha API so'rovlari `Authorization` headerida token bilan yuborilishi kerak:

```
Authorization: Bearer <token>
```

Token konfiguratsiya faylida `api.token` maydonida ko'rsatilgan.

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

### Client life_time vaqtini olish

**So'rov:**

```
GET /api/client/:id/lifetime
```

**Javob:**

```json
{
  "id": 1,
  "description": "Normal client",
  "type": "normal",
  "life_time": 2592000,
  "expires_at": "2023-12-31T23:59:59Z",
  "remaining_time": 2591000
}
```

### Client life_time vaqtini yangilash

**So'rov:**

```
PUT /api/client/:id/lifetime
```

**Request body:**

```json
{
  "life_time": 604800
}
```

**Javob:**

```json
{
  "id": 1,
  "description": "Normal client",
  "type": "normal",
  "life_time": 604800,
  "expires_at": "2023-12-08T12:00:00Z",
  "remaining_time": 604800,
  "message": "Client life_time vaqti muvaffaqiyatli yangilandi"
}
```

### Client traffic ma'lumotlarini olish

**So'rov:**

```
GET /api/client/:id/traffic
```

**Javob:**

```json
{
  "id": 1,
  "description": "Normal client",
  "public_key": "client_public_key",
  "address": "10.0.0.2/32",
  "type": "normal",
  "latest_handshake": "2023-12-01T12:30:45Z",
  "bytes_received": 1048576,
  "bytes_sent": 524288,
  "allowed_ips": "10.0.0.2/32",
  "endpoint": "192.168.1.100:48220",
  "bytes_received_formatted": "1.00 MB",
  "bytes_sent_formatted": "512.00 KB",
  "total_traffic": "1.50 MB"
}
```

### Barcha clientlar traffic ma'lumotlarini olish

**So'rov:**

```
GET /api/clients/traffic
```

**Javob:**

```json
[
  {
    "id": 1,
    "description": "Normal client",
    "public_key": "client_public_key",
    "address": "10.0.0.2/32",
    "type": "normal",
    "latest_handshake": "2023-12-01T12:30:45Z",
    "bytes_received": 1048576,
    "bytes_sent": 524288,
    "allowed_ips": "10.0.0.2/32",
    "endpoint": "192.168.1.100:48220",
    "bytes_received_formatted": "1.00 MB",
    "bytes_sent_formatted": "512.00 KB",
    "total_traffic": "1.50 MB"
  },
  {
    "id": 2,
    "description": "VIP client",
    "public_key": "client_public_key",
    "address": "10.0.0.3/32",
    "type": "vip",
    "latest_handshake": "2023-12-01T13:15:30Z",
    "bytes_received": 2097152,
    "bytes_sent": 1048576,
    "allowed_ips": "10.0.0.3/32",
    "endpoint": "192.168.1.100:41244",
    "bytes_received_formatted": "2.00 MB",
    "bytes_sent_formatted": "1.00 MB",
    "total_traffic": "3.00 MB"
  }
]
```

### Server holatini olish

**So'rov:**

```
GET /api/server/status
```

**Javob:**

```json
{
  "server": {
    "interface_name": "wg0",
    "listen_port": 51820,
    "public_key": "server_public_key",
    "active_clients": 2,
    "total_clients": 5,
    "total_bytes_received": 3145728,
    "total_bytes_sent": 1572864,
    "total_traffic": 4718592,
    "last_handshake": "2023-12-01T13:15:30Z",
    "uptime": "up 10 days, 5 hours, 30 minutes",
    "total_bytes_received_formatted": "3.00 MB",
    "total_bytes_sent_formatted": "1.50 MB",
    "total_traffic_formatted": "4.50 MB"
  },
  "database": {
    "total_clients": 5,
    "active_clients": 2,
    "inactive_clients": 3
  },
  "system": {
    "uptime": "up 10 days, 5 hours, 30 minutes"
  },
  "traffic": {
    "total_bytes_received": 3145728,
    "total_bytes_sent": 1572864,
    "total_traffic": 4718592,
    "total_bytes_received_formatted": "3.00 MB",
    "total_bytes_sent_formatted": "1.50 MB",
    "total_traffic_formatted": "4.50 MB"
  }
}
```

### Server health holatini olish

**So'rov:**

```
GET /api/health
```

**Javob:**

```json
{
  "status": "ok"
}
```

## Texnik tafsilotlar

- Server konfiguratsiyasiga yangi peerlar `wg` va `wg-quick` buyruqlari orqali qo'shiladi
- Client turiga qarab IP manzil generatsiya qilinadi:
  - Normal clientlar uchun: 10.7.x.x subnet
  - VIP clientlar uchun: 10.77.x.x subnet
- IP manzillar databasedagi mavjud manzillarni tekshirib, bo'sh manzilni topish orqali belgilanadi
- O'chirilgan clientlarning IP manzillari yangi clientlar uchun qayta ishlatilishi mumkin
- Client ma'lumotlari SQLite databaseda saqlanadi (`./data/wireguard.db`)
- Database GORM ORM orqali boshqariladi
- Clientlar uchun "normal" va "vip" turlari mavjud
- Clientlar uchun amal qilish muddati belgilanishi mumkin (soniyalarda)
- Muddati o'tgan clientlar avtomatik ravishda o'chiriladi va Wireguard konfiguratsiyasidan olib tashlanadi
- Muddati o'tgan clientlarni tekshirish har 15 daqiqada amalga oshiriladi
- Client life_time vaqtini olish va yangilash uchun maxsus API endpointlar mavjud
- Client traffic ma'lumotlarini olish uchun maxsus API endpointlar mavjud
- Traffic ma'lumotlari `wg-json` buyrug'i orqali olinadi va odam o'qiy oladigan formatda qaytariladi
- Traffic ma'lumotlari quyidagilarni o'z ichiga oladi:
  - Oxirgi handshake vaqti
  - Qabul qilingan baytlar (bytes_received)
  - Yuborilgan baytlar (bytes_sent)
  - Ruxsat berilgan IP manzillar (allowed_ips)
  - Client endpoint (IP manzil va port)
  - Odam o'qiy oladigan formatdagi ma'lumotlar (MB, GB kabi)
  - Umumiy traffic (qabul qilingan + yuborilgan)
- Server holati API endpointi quyidagi ma'lumotlarni taqdim etadi:
  - Server interfeysi nomi va port raqami
  - Server public key
  - Aktiv va umumiy clientlar soni
  - Umumiy qabul qilingan va yuborilgan traffic
  - Oxirgi handshake vaqti
  - Server uptime (ishga tushirilgan vaqtdan beri o'tgan vaqt)
  - Databasedagi clientlar statistikasi

## Xavfsizlik

Dastur quyidagi xavfsizlik mexanizmlarini taqdim etadi:

### API Token autentifikatsiyasi

Barcha API so'rovlari `Authorization` headerida token bilan yuborilishi kerak:

```
Authorization: Bearer <token>
```

Token konfiguratsiya faylida `api.token` maydonida ko'rsatilgan.

### IP bloklash tizimi

Dastur xato token bilan API so'rovlarini yuborgan IP manzillarni bloklash imkoniyatini taqdim etadi. Bu mexanizm quyidagicha ishlaydi:

1. Agar foydalanuvchi belgilangan maksimal urinishlar sonidan ko'p marta xato token bilan so'rov yuborsa, uning IP manzili ma'lum vaqtga bloklanadi.
2. Bloklangan IP manzildan kelgan barcha so'rovlar rad etiladi va bloklash muddati tugaguncha 429 (Too Many Requests) xatolik kodi qaytariladi.
3. Barcha muvaffaqiyatsiz urinishlar va bloklashlar log fayliga yoziladi.

IP bloklash tizimi konfiguratsiyasi:

```yaml
security:
  ip_blocker:
    enabled: true # IP bloklash tizimini yoqish/o'chirish
    max_attempts: 3 # Maksimal urinishlar soni
    block_duration: 60 # Bloklash muddati (minutlarda)
    log_file_path: "./logs/auth_failures.log" # Log fayli yo'li
```

Standart konfiguratsiyada, 3 marta xato token bilan so'rov yuborgan IP manzil 1 soatga bloklanadi.
