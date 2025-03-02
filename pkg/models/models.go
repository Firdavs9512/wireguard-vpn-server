package models

// WireguardConfig - Wireguard konfiguratsiya ma'lumotlari
type WireguardConfig struct {
	Endpoint   string `json:"endpoint"`
	Address    string `json:"address"`
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	DNS        string `json:"dns"`
	AllowedIPs string `json:"allowed_ips"`
}

// ClientResponse - API response strukturasi
type ClientResponse struct {
	Config string          `json:"config"`
	Data   WireguardConfig `json:"data"`
}
