package handlers

import (
	"faucet-backend/database"
	"faucet-backend/models"
	"faucet-backend/services"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gofiber/fiber/v2"
)

type DripRequest struct {
	Address      string `json:"address"`
	TokenID      string `json:"tokenId"`
	CaptchaToken string `json:"captchaToken"`
	Fingerprint  string `json:"fingerprint"`
}

func RequestDrip(c *fiber.Ctx) error {
	var req DripRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate address
	if !common.IsHexAddress(req.Address) {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid Ethereum address",
		})
	}

	address := common.HexToAddress(req.Address).Hex()
	ip := c.IP()

	// Verify token exists and is active
	var token models.Token
	if err := database.DB.Where("id = ? AND is_active = true", req.TokenID).First(&token).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid or inactive token",
		})
	}

	// Verify CAPTCHA
	if err := services.VerifyCaptcha(req.CaptchaToken, ip); err != nil {
		return c.Status(403).JSON(fiber.Map{
			"error": "CAPTCHA verification failed",
		})
	}

	// Check rate limits
	rateLimitCheck, err := services.CheckRateLimit(address, req.TokenID, ip, req.Fingerprint)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Rate limit check failed",
		})
	}

	if !rateLimitCheck.Allowed {
		return c.Status(429).JSON(fiber.Map{
			"error":      rateLimitCheck.Reason,
			"retryAfter": rateLimitCheck.RetryAfter,
		})
	}

	// Create drip record
	drip := models.Drip{
		Recipient:   address,
		TokenID:     req.TokenID,
		Amount:      token.DripAmount,
		IPAddress:   ip,
		Fingerprint: req.Fingerprint,
		Status:      "pending",
	}

	if err := database.DB.Create(&drip).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create drip record",
		})
	}

	// Execute transaction (async)
	go func() {
		services.ExecuteDrip(address, req.TokenID, drip.ID)
	}()

	// Record in rate limiter
	services.RecordDrip(address, req.TokenID, ip, req.Fingerprint)

	return c.JSON(fiber.Map{
		"success": true,
		"txHash":  "",
		"amount":  token.DripAmount,
		"token":   token.Symbol,
		"message": fmt.Sprintf("Sending %s %s to your address", token.DripAmount, token.Symbol),
		"dripId":  drip.ID,
	})
}

func GetStatus(c *fiber.Ctx) error {
	address := c.Params("address")
	ip := c.IP()

	if !common.IsHexAddress(address) {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid address",
		})
	}

	address = strings.ToLower(common.HexToAddress(address).Hex())

	// Get all tokens
	var tokens []models.Token
	database.DB.Where("is_active = true").Find(&tokens)

	// Get drip status for each token
	type TokenStatus struct {
		TokenID       string      `json:"tokenId"`
		TokenSymbol   string      `json:"tokenSymbol"`
		LastDrip      interface{} `json:"lastDrip"`
		CanRequest    bool        `json:"canRequest"`
		NextRequestIn int64       `json:"nextRequestIn"`
	}

	var drips []TokenStatus

	for _, token := range tokens {
		var drip models.Drip
		result := database.DB.Where("recipient = ? AND token_id = ?", address, token.ID).
			Order("created_at DESC").
			First(&drip)

		var lastDrip interface{} = nil
		canRequest := true
		var nextRequestIn int64 = 0

		if result.Error == nil {
			lastDrip = fiber.Map{
				"txHash":    drip.TxHash,
				"amount":    drip.Amount,
				"timestamp": drip.CreatedAt,
				"status":    drip.Status,
			}

			// Calculate cooldown
			cooldownDuration := time.Duration(token.CooldownHours) * time.Hour
			elapsed := time.Since(drip.CreatedAt)
			canRequest = elapsed >= cooldownDuration

			if !canRequest {
				nextRequestIn = int64((cooldownDuration - elapsed).Seconds())
			}
		}

		drips = append(drips, TokenStatus{
			TokenID:       token.ID,
			TokenSymbol:   token.Symbol,
			LastDrip:      lastDrip,
			CanRequest:    canRequest,
			NextRequestIn: nextRequestIn,
		})
	}

	// Get IP rate limit
	ipRateLimit, _ := services.GetIPRateLimit(ip)

	return c.JSON(fiber.Map{
		"drips": drips,
		"ipRateLimit": fiber.Map{
			"used":       ipRateLimit.Used,
			"limit":      ipRateLimit.Limit,
			"canRequest": ipRateLimit.CanRequest,
		},
	})
}

func GetTokens(c *fiber.Ctx) error {
	var tokens []models.Token
	database.DB.Where("is_active = true").Find(&tokens)

	// Get stats for each token
	type TokenResponse struct {
		models.Token
		TotalDrips int64  `json:"totalDrips"`
		Balance    string `json:"balance"`
	}

	var response []TokenResponse

	for _, token := range tokens {
		var count int64
		database.DB.Model(&models.Drip{}).
			Where("token_id = ? AND status = ?", token.ID, "completed").
			Count(&count)

		// Get balance
		var balance string
		if token.Address == "" {
			// Native ETH
			bal, _ := services.GetFaucetBalance()
			balance = bal
		} else {
			// ERC20
			tokenAddr := common.HexToAddress(token.Address)
			bal, err := services.GetERC20Balance(tokenAddr)
			if err == nil {
				decimals := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(token.Decimals)), nil)
				balFloat := new(big.Float).Quo(new(big.Float).SetInt(bal), new(big.Float).SetInt(decimals))
				balance = balFloat.Text('f', 2)
			} else {
				balance = "0"
			}
		}

		response = append(response, TokenResponse{
			Token:      token,
			TotalDrips: count,
			Balance:    balance,
		})
	}

	return c.JSON(fiber.Map{
		"tokens": response,
	})
}

func GetStats(c *fiber.Ctx) error {
	// Total drips across all tokens
	var totalDrips int64
	database.DB.Model(&models.Drip{}).Where("status = ?", "completed").Count(&totalDrips)

	// Unique users (distinct recipients)
	var totalUsers int64
	database.DB.Model(&models.Drip{}).
		Where("status = ?", "completed").
		Distinct("recipient").
		Count(&totalUsers)

	// Tokens distributed per token type
	type TokenDistribution struct {
		TokenID string
		Symbol  string
		Count   int64
	}

	var distributions []TokenDistribution
	database.DB.Table("drips").
		Select("drips.token_id, tokens.symbol, COUNT(*) as count").
		Joins("JOIN tokens ON tokens.id = drips.token_id").
		Where("drips.status = ?", "completed").
		Group("drips.token_id, tokens.symbol").
		Scan(&distributions)

	tokensDistributed := make(map[string]int64)
	for _, dist := range distributions {
		tokensDistributed[dist.Symbol] = dist.Count
	}

	return c.JSON(fiber.Map{
		"totalDrips":        totalDrips,
		"totalUsers":        totalUsers,
		"tokensDistributed": tokensDistributed,
	})
}

func GetFaucetAddress() string {
	return services.GetWalletAddress()
}
