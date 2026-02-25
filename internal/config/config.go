package config

import (
	"crypto/rand"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration
type Config struct {
	ServerPort              int
	TR069Port               int
	TR069Secure             bool
	DatabaseURL             string
	JWTSecret               string
	LogLevel                string
	AuthEnabled             bool
	AdminUser               string
	AdminPass               string
	MikrotikHost            string
	MikrotikUser            string
	MikrotikPass            string
	MikrotikPort            int
	TripayAPIKey            string
	TripayPrivateKey        string
	TripayMerchantCode      string
	TripayMode              string // sandbox or production
	WAProviderURL           string
	WAApiKey                string
	FirebaseCredentialsFile string
	TelegramToken           string
	TelegramChatID          string
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		// Generate a random JWT secret if not provided
		jwtSecret = generateRandomSecret(32)
		fmt.Printf("⚠️  WARNING: JWT_SECRET not set, generated random secret: %s\n", jwtSecret)
		fmt.Printf("   Please set JWT_SECRET environment variable for production use!\n")
	}
	
	return &Config{
		ServerPort:              getEnvAsInt("SERVER_PORT", 8080),
		TR069Port:               getEnvAsInt("TR069_PORT", 7547),
		TR069Secure:             getEnvAsBool("TR069_SECURE", false),
		DatabaseURL:             getEnv("DATABASE_URL", "./data/goacs.db"),
		JWTSecret:               jwtSecret,
		LogLevel:                getEnv("LOG_LEVEL", "info"),
		AuthEnabled:             getEnvAsBool("AUTH_ENABLED", true),
		AdminUser:               getEnv("ADMIN_USER", "admin"),
		AdminPass:               getEnv("ADMIN_PASS", "admin123"),
		MikrotikHost:            getEnv("MIKROTIK_HOST", "192.168.88.1"),
		MikrotikUser:            getEnv("MIKROTIK_USER", "admin"),
		MikrotikPass:            getEnv("MIKROTIK_PASS", ""),
		MikrotikPort:            getEnvAsInt("MIKROTIK_PORT", 8728),
		TripayAPIKey:            getEnv("TRIPAY_API_KEY", "DEV-YOUR-API-KEY"),
		TripayPrivateKey:        getEnv("TRIPAY_PRIVATE_KEY", "DEV-YOUR-PRIVATE-KEY"),
		TripayMerchantCode:      getEnv("TRIPAY_MERCHANT_CODE", "T12345"),
		TripayMode:              getEnv("TRIPAY_MODE", "sandbox"),
		WAProviderURL:           getEnv("WA_PROVIDER_URL", "https://api.fonnte.com/send"),
		WAApiKey:                getEnv("WA_API_KEY", ""),
		FirebaseCredentialsFile: getEnv("FIREBASE_CREDENTIALS_FILE", "firebase-service-account.json"),
		TelegramToken:           getEnv("TELEGRAM_TOKEN", "1981178828:AAEld2oOK1rkvSOlHuyx7HGd8kYsVzzdZGk"),
		TelegramChatID:          getEnv("TELEGRAM_CHAT_ID", "567858628"),
	}
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// generateRandomSecret generates a cryptographically secure random string
func generateRandomSecret(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		// Fallback to time-based seed if crypto/rand fails
		return fmt.Sprintf("fallback-secret-%d", time.Now().UnixNano())
	}
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		switch value {
		case "1", "t", "T", "true", "TRUE", "True", "yes", "YES":
			return true
		case "0", "f", "F", "false", "FALSE", "False", "no", "NO":
			return false
		}
	}
	return defaultValue
}
