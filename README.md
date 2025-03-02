# Wireguard VPN Client Yaratuvchi API

Bu loyiha Golang yordamida Wireguard VPN clientlarini yaratish uchun API server taqdim etadi.

## Imkoniyatlar

- `/api/client` endpointiga POST so'rov yuborish orqali yangi Wireguard client yaratish
- Har bir client uchun unikal IP manzil va kalitlar yaratish
- Wireguard konfiguratsiya faylini va uning ma'lumotlarini JSON formatida qaytarish
- Server konfiguratsiyasiga yangi peerlarni avtomatik qo'shish

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

**Javob:**

```json
{
  "config": "[Interface]\nPrivateKey = CLIENT_PRIVATE_KEY\nAddress = 10.0.0.X/32\nDNS = 1.1.1.1\n\n[Peer]\nPublicKey = SERVER_PUBLIC_KEY\nAllowedIPs = 0.0.0.0/0, ::/0\nEndpoint = SERVER_IP:51820\nPersistentKeepalive = 25\n",
  "data": {
    "endpoint": "SERVER_IP:51820",
    "address": "10.0.0.X/32",
    "private_key": "CLIENT_PRIVATE_KEY",
    "public_key": "SERVER_PUBLIC_KEY",
    "dns": "1.1.1.1",
    "allowed_ips": "0.0.0.0/0, ::/0"
  }
}
```

## Texnik tafsilotlar

- Server konfiguratsiyasiga yangi peerlar `wg` va `wg-quick` buyruqlari orqali qo'shiladi
- Har bir client uchun unikal IP manzil 10.0.0.2 - 10.0.0.251 oralig'ida generatsiya qilinadi
- Server interfeysi nomi `/etc/wireguard/` papkasidagi birinchi `.conf` faylidan olinadi

## Xavfsizlik eslatmasi

Bu dastur server konfiguratsiyasini o'zgartirish uchun root huquqlariga ega bo'lishi kerak. Ishlab chiqarish muhitida qo'shimcha xavfsizlik choralarini ko'rish tavsiya etiladi. 