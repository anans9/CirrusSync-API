package config

import "github.com/pquerna/otp"

type TOTPConfig struct {
	TOTPIssuer    string        // The issuer name in TOTP apps
	TOTPDigits    otp.Digits    // Number of digits in TOTP code
	TOTPPeriod    uint          // TOTP period in seconds
	TOTPSkew      uint          // Accepted time skew for validation
	TOTPAlgorithm otp.Algorithm // Algorithm used for TOTP
}

func LoadTOTPConfig() *TOTPConfig {
	config := &TOTPConfig{
		TOTPIssuer:    "CirrusSync",
		TOTPDigits:    6,
		TOTPPeriod:    30,
		TOTPSkew:      1,
		TOTPAlgorithm: otp.AlgorithmSHA512,
	}

	return config
}
