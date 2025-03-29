package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp" // Added import
	"github.com/pquerna/otp/totp"
)

func GenerateTOTP(secret string) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("totp secret cannot be empty")
	}

	cleanSecret := strings.ToUpper(strings.ReplaceAll(secret, " ", ""))

	opts := totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    6,
		Algorithm: otp.AlgorithmSHA1, // Correct usage of otp package constant
	}

	passcode, err := totp.GenerateCodeCustom(cleanSecret, time.Now().UTC(), opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate totp code: %w", err)
	}

	return passcode, nil
}

func ValidateTOTP(passcode, secret string) (bool, error) {
	if secret == "" {
		return false, fmt.Errorf("totp secret cannot be empty")
	}
	if passcode == "" {
		return false, fmt.Errorf("passcode cannot be empty")
	}

	cleanSecret := strings.ToUpper(strings.ReplaceAll(secret, " ", ""))

	opts := totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    6,
		Algorithm: otp.AlgorithmSHA1, // Correct usage of otp package constant
	}

	valid, err := totp.ValidateCustom(passcode, cleanSecret, time.Now().UTC(), opts)
	if err != nil {
		return false, fmt.Errorf("failed to validate totp code: %w", err)
	}

	return valid, nil
}
