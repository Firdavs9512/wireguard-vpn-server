package config

// Wireguard konfiguratsiya konstantalari
const (
	DNS                 = "1.1.1.1, 8.8.8.8"
	AllowedIPs          = "0.0.0.0/0, ::/0"
	Endpoint            = "192.168.1.151:51820"
	PersistentKeepalive = 25
	InterfaceName       = "wg0"
	ServerPublicKeyPath = "/etc/wireguard/server_public.key"
	ServerIP            = "192.168.1.151"
)
