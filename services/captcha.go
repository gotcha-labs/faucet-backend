package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

type CaptchaResponse struct {
	Success bool `json:"success"`
}

func VerifyCaptcha(token, remoteIP string) error {
	secretKey := os.Getenv("GOTCHA_SECRET_KEY")
	if secretKey == "" {
		return errors.New("GOTCHA_SECRET_KEY not configured")
	}

	// Use your custom CAPTCHA URL
	verifyURL := os.Getenv("GOTCHA_VERIFY_URL")
	if verifyURL == "" {
		verifyURL = "http://api.gotcha.land/api/siteverify"
	}

	resp, err := http.PostForm(verifyURL, url.Values{
		"secret":   {secretKey},
		"response": {token},
		"remoteip": {remoteIP},
	})

	if err != nil {
		return fmt.Errorf("captcha verification request failed: %w", err)
	}
	defer resp.Body.Close()

	var captchaResp CaptchaResponse
	if err := json.NewDecoder(resp.Body).Decode(&captchaResp); err != nil {
		return fmt.Errorf("failed to parse captcha response: %w", err)
	}

	if !captchaResp.Success {
		return errors.New("captcha verification failed")
	}

	return nil
}
