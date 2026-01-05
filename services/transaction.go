
package services

import (
	"context"
	"faucet-backend/database"
	"faucet-backend/models"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func ExecuteDrip(recipient, tokenID string, dripID uint) {
	// Get token info
	var token models.Token
	if err := database.DB.Where("id = ? AND is_active = true", tokenID).First(&token).Error; err != nil {
		log.Printf("❌ Token %s not found or inactive", tokenID)
		database.DB.Model(&models.Drip{}).Where("id = ?", dripID).Updates(map[string]interface{}{
			"status": "failed",
			"error":  "Token not found or inactive",
		})
		return
	}

	// Parse amount
	amount := new(big.Float)
	amount.SetString(token.DripAmount)

	// Convert to smallest unit based on decimals
	decimals := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(token.Decimals)), nil)
	amountFloat := new(big.Float).Mul(amount, new(big.Float).SetInt(decimals))
	amountInt := new(big.Int)
	amountFloat.Int(amountInt)

	recipientAddr := common.HexToAddress(recipient)
	var txHash string
	var err error

	// Send transaction based on token type
	if token.Address == "" {
		// Native ETH transfer
		txHash, err = SendTransaction(recipientAddr, amountInt)
	} else {
		// ERC20 transfer
		tokenAddr := common.HexToAddress(token.Address)
		txHash, err = SendERC20(tokenAddr, recipientAddr, amountInt)
	}

	if err != nil {
		log.Printf("❌ Failed to send %s to %s: %v", token.Symbol, recipient, err)
		database.DB.Model(&models.Drip{}).Where("id = ?", dripID).Updates(map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		})
		return
	}

	log.Printf("💧 %s drip sent: %s to %s", token.Symbol, txHash, recipient)

	// Update with tx hash
	database.DB.Model(&models.Drip{}).Where("id = ?", dripID).Updates(map[string]interface{}{
		"tx_hash": txHash,
		"status":  "pending",
	})

	// Wait for confirmation in background
	go func() {
		ctx := context.Background()

		for i := 0; i < 60; i++ {
			time.Sleep(5 * time.Second)

			receipt, err := client.TransactionReceipt(ctx, common.HexToHash(txHash))
			if err != nil {
				continue
			}

			status := "completed"
			if receipt.Status == 0 {
				status = "failed"
			}

			database.DB.Model(&models.Drip{}).Where("id = ?", dripID).Updates(map[string]interface{}{
				"status":       status,
				"completed_at": time.Now(),
			})

			if status == "completed" {
				log.Printf("✅ %s drip confirmed: %s", token.Symbol, txHash)
			} else {
				log.Printf("❌ %s drip failed: %s", token.Symbol, txHash)
			}

			return
		}

		log.Printf("⚠️  Transaction %s not confirmed after 5 minutes", txHash)
	}()
}
