package services

import (
	"context"
	"fmt"
	"faucet-backend/database"
	"faucet-backend/models"
	"strconv"
	"strings"
	"time"
)

type RateLimitCheck struct {
	Allowed    bool
	RetryAfter int64
	Reason     string
}

type IPRateLimit struct {
	Used       int
	Limit      int
	CanRequest bool
}

func CheckRateLimit(wallet, tokenID, ip, fingerprint string) (*RateLimitCheck, error) {
	ctx := context.Background()

	// Get token cooldown
	var cooldownHours int
	if err := database.DB.Model(&models.Token{}).
		Where("id = ?", tokenID).
		Pluck("cooldown_hours", &cooldownHours).Error; err != nil {
		cooldownHours = 24
	}

	wallet = strings.ToLower(wallet)

	// Check wallet cooldown for specific token
	walletKey := fmt.Sprintf("faucet:wallet:%s:%s", wallet, tokenID)
	lastDrip, err := database.Redis.Get(ctx, walletKey).Result()

	if err == nil && lastDrip != "" {
		timestamp, _ := strconv.ParseInt(lastDrip, 10, 64)
		elapsed := time.Now().Unix() - timestamp
		cooldownSeconds := int64(cooldownHours * 3600)

		if elapsed < cooldownSeconds {
			retryAfter := cooldownSeconds - elapsed
			return &RateLimitCheck{
				Allowed:    false,
				RetryAfter: retryAfter,
				Reason:     fmt.Sprintf("Wallet cooldown for this token: %d hours remaining", retryAfter/3600),
			}, nil
		}
	}

	// Check IP rate limit (3 total requests per day across all tokens)
	ipKey := fmt.Sprintf("faucet:ip:%s", ip)
	ipCount, err := database.Redis.Get(ctx, ipKey).Result()

	if err == nil && ipCount != "" {
		count, _ := strconv.Atoi(ipCount)
		if count >= 3 {
			ttl, _ := database.Redis.TTL(ctx, ipKey).Result()
			return &RateLimitCheck{
				Allowed:    false,
				RetryAfter: int64(ttl.Seconds()),
				Reason:     "IP daily limit reached (3 requests per day)",
			}, nil
		}
	}

	// Check fingerprint (2 per day)
	if fingerprint != "" {
		fpKey := fmt.Sprintf("faucet:fp:%s", fingerprint)
		fpCount, err := database.Redis.Get(ctx, fpKey).Result()

		if err == nil && fpCount != "" {
			count, _ := strconv.Atoi(fpCount)
			if count >= 2 {
				ttl, _ := database.Redis.TTL(ctx, fpKey).Result()
				return &RateLimitCheck{
					Allowed:    false,
					RetryAfter: int64(ttl.Seconds()),
					Reason:     "Device limit exceeded (2 per day)",
				}, nil
			}
		}
	}

	return &RateLimitCheck{Allowed: true}, nil
}

func RecordDrip(wallet, tokenID, ip, fingerprint string) error {
	ctx := context.Background()

	// Get token cooldown
	var cooldownHours int
	if err := database.DB.Model(&models.Token{}).
		Where("id = ?", tokenID).
		Pluck("cooldown_hours", &cooldownHours).Error; err != nil {
		cooldownHours = 24
	}

	wallet = strings.ToLower(wallet)

	// Set wallet cooldown for specific token
	walletKey := fmt.Sprintf("faucet:wallet:%s:%s", wallet, tokenID)
	database.Redis.Set(ctx, walletKey, time.Now().Unix(), time.Duration(cooldownHours)*time.Hour)

	// Increment IP counter (across all tokens)
	ipKey := fmt.Sprintf("faucet:ip:%s", ip)
	count, _ := database.Redis.Incr(ctx, ipKey).Result()
	if count == 1 {
		database.Redis.Expire(ctx, ipKey, 24*time.Hour)
	}

	// Increment fingerprint counter
	if fingerprint != "" {
		fpKey := fmt.Sprintf("faucet:fp:%s", fingerprint)
		count, _ := database.Redis.Incr(ctx, fpKey).Result()
		if count == 1 {
			database.Redis.Expire(ctx, fpKey, 24*time.Hour)
		}
	}

	return nil
}

func GetIPRateLimit(ip string) (*IPRateLimit, error) {
	ctx := context.Background()
	ipKey := fmt.Sprintf("faucet:ip:%s", ip)

	ipCount, err := database.Redis.Get(ctx, ipKey).Result()
	used := 0
	if err == nil && ipCount != "" {
		used, _ = strconv.Atoi(ipCount)
	}

	return &IPRateLimit{
		Used:       used,
		Limit:      3,
		CanRequest: used < 3,
	}, nil
}
