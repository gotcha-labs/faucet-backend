package config

import (
	"faucet-backend/database"
	"faucet-backend/models"
	"log"
)

// SeedTokens initializes default tokens in the database
func SeedTokens() {
	tokens := []models.Token{
		{
			ID:            "eth",
			Name:          "Ethereum",
			Symbol:        "ETH",
			Address:       "", // Native token
			DripAmount:    "0.5",
			CooldownHours: 24,
			Decimals:      18,
			LogoURL:       "https://cryptologos.cc/logos/ethereum-eth-logo.svg",
			IsActive:      true,
		},
		{
			ID:            "usdc",
			Name:          "USD Coin",
			Symbol:        "USDC",
			Address:       "0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238", // Sepolia USDC
			DripAmount:    "100",
			CooldownHours: 24,
			Decimals:      6,
			LogoURL:       "https://cryptologos.cc/logos/usd-coin-usdc-logo.svg",
			IsActive:      true,
		},
		{
			ID:            "usdt",
			Name:          "Tether USD",
			Symbol:        "USDT",
			Address:       "0xaA8E23Fb1079EA71e0a56F48a2aA51851D8433D0", // Sepolia USDT (example)
			DripAmount:    "100",
			CooldownHours: 24,
			Decimals:      6,
			LogoURL:       "https://cryptologos.cc/logos/tether-usdt-logo.svg",
			IsActive:      true,
		},
		{
			ID:            "dai",
			Name:          "Dai Stablecoin",
			Symbol:        "DAI",
			Address:       "0x68194a729C2450ad26072b3D33ADaCbcef39D574", // Sepolia DAI (example)
			DripAmount:    "100",
			CooldownHours: 24,
			Decimals:      18,
			LogoURL:       "https://cryptologos.cc/logos/multi-collateral-dai-dai-logo.svg",
			IsActive:      true,
		},
		{
			ID:            "link",
			Name:          "Chainlink",
			Symbol:        "LINK",
			Address:       "0x779877A7B0D9E8603169DdbD7836e478b4624789", // Sepolia LINK
			DripAmount:    "10",
			CooldownHours: 24,
			Decimals:      18,
			LogoURL:       "https://cryptologos.cc/logos/chainlink-link-logo.svg",
			IsActive:      true,
		},
	}

	for _, token := range tokens {
		var existing models.Token
		result := database.DB.Where("id = ?", token.ID).First(&existing)

		if result.Error != nil {
			// Token doesn't exist, create it
			if err := database.DB.Create(&token).Error; err != nil {
				log.Printf("Failed to seed token %s: %v", token.ID, err)
			} else {
				log.Printf("✅ Seeded token: %s", token.Symbol)
			}
		}
	}
}
