.PHONY: run build clean

# Dasturni ishga tushirish
run:
	sudo go run cmd/server/main.go

# Dasturni build qilish
build:
	go build -o wireguard-client-api cmd/server/main.go

# Dasturni build qilib, ishga tushirish
start: build
	sudo ./wireguard-client-api

# Dasturni build qilib, systemd service sifatida o'rnatish
install: build
	sudo cp wireguard-client-api /usr/local/bin/
	@echo "[Unit]" > wireguard-client-api.service
	@echo "Description=Wireguard VPN Client API" >> wireguard-client-api.service
	@echo "After=network.target" >> wireguard-client-api.service
	@echo "" >> wireguard-client-api.service
	@echo "[Service]" >> wireguard-client-api.service
	@echo "ExecStart=/usr/local/bin/wireguard-client-api" >> wireguard-client-api.service
	@echo "Restart=on-failure" >> wireguard-client-api.service
	@echo "" >> wireguard-client-api.service
	@echo "[Install]" >> wireguard-client-api.service
	@echo "WantedBy=multi-user.target" >> wireguard-client-api.service
	sudo mv wireguard-client-api.service /etc/systemd/system/
	sudo systemctl daemon-reload
	sudo systemctl enable wireguard-client-api.service
	sudo systemctl start wireguard-client-api.service
	@echo "Service installed and started"

# Systemd serviceni o'chirish
uninstall:
	sudo systemctl stop wireguard-client-api.service || true
	sudo systemctl disable wireguard-client-api.service || true
	sudo rm -f /etc/systemd/system/wireguard-client-api.service
	sudo rm -f /usr/local/bin/wireguard-client-api
	sudo systemctl daemon-reload
	@echo "Service uninstalled"

# Tozalash
clean:
	rm -f wireguard-client-api
	rm -f wireguard-client-api.service 