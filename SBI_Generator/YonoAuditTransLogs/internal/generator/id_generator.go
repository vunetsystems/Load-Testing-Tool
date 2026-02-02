package generator

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
)

// GenerateTransactionID generates a unique transaction ID
// Format: <epoch_millis><pattern><random_suffix>
func GenerateTransactionID(pattern string) string {
	epochMillis := time.Now().UnixMilli()
	randomSuffix := generateRandomNumericString(6)
	return fmt.Sprintf("%d%s%s", epochMillis, pattern, randomSuffix)
}

// GenerateRequestNo generates a random numeric string of specified length
func GenerateRequestNo(length int) string {
	return generateRandomNumericString(length)
}

// GenerateTraceID generates a trace ID based on format
func GenerateTraceID(format string, length int) string {
	if format == "uuid" {
		return uuid.New().String()
	}
	// hex format
	return generateRandomHexString(length)
}

// generateRandomNumericString generates a random numeric string
func generateRandomNumericString(length int) string {
	const digits = "0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		result[i] = digits[n.Int64()]
	}
	return string(result)
}

// generateRandomHexString generates a random hex string
func generateRandomHexString(length int) string {
	bytes := make([]byte, (length+1)/2)
	rand.Read(bytes)
	hexStr := hex.EncodeToString(bytes)
	if len(hexStr) > length {
		return hexStr[:length]
	}
	return hexStr
}

// generateRandomAlphanumeric generates a random alphanumeric string
func generateRandomAlphanumeric(length int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		result[i] = chars[n.Int64()]
	}
	return string(result)
}
