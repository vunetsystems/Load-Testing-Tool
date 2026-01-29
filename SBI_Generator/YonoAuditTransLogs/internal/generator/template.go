package generator

import (
	"crypto/rand"
	"encoding/base64"
	"math/big"

	"kafka-message-generator/internal/config"
)

// TemplateSelector handles random selection from templates
type TemplateSelector struct {
	config *config.Config
}

// NewTemplateSelector creates a new template selector
func NewTemplateSelector(cfg *config.Config) *TemplateSelector {
	return &TemplateSelector{config: cfg}
}

// SelectErrorCode randomly selects an error code
func (ts *TemplateSelector) SelectErrorCode() string {
	return selectRandom(ts.config.Templates.Error.ErrorCodes)
}

// SelectErrorType randomly selects an error type
func (ts *TemplateSelector) SelectErrorType() string {
	return selectRandom(ts.config.Templates.Error.ErrorTypes)
}

// SelectErrorDescription randomly selects an error description
func (ts *TemplateSelector) SelectErrorDescription() string {
	return selectRandom(ts.config.Templates.Error.ErrorDescriptions)
}

// SelectErrorDetails randomly selects error details
func (ts *TemplateSelector) SelectErrorDetails() string {
	return selectRandom(ts.config.Templates.Error.ErrorDetails)
}

// SelectTransactionStatus randomly selects a transaction status with weighting
func (ts *TemplateSelector) SelectTransactionStatus() string {
	statuses := ts.config.Templates.Transaction.Statuses
	if len(statuses) == 0 {
		return "success"
	}

	// Calculate total weight
	totalWeight := 0
	for _, status := range statuses {
		totalWeight += status.Weight
	}

	// Generate random number in range [0, totalWeight)
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(totalWeight)))
	randomWeight := int(n.Int64())

	// Select based on weight
	currentWeight := 0
	for _, status := range statuses {
		currentWeight += status.Weight
		if randomWeight < currentWeight {
			return status.Value
		}
	}

	return statuses[0].Value
}

// SelectUserType randomly selects a user type
func (ts *TemplateSelector) SelectUserType() string {
	return selectRandom(ts.config.Templates.Transaction.UserTypes)
}

// SelectCommandID randomly selects a command ID
func (ts *TemplateSelector) SelectCommandID() string {
	return selectRandom(ts.config.Templates.Transaction.CommandIDs)
}

// SelectChannelID randomly selects a channel ID
func (ts *TemplateSelector) SelectChannelID() int {
	return selectRandomInt(ts.config.Templates.Transaction.ChannelIDs)
}

// SelectUserRelationshipNumber randomly selects a user relationship number
func (ts *TemplateSelector) SelectUserRelationshipNumber() string {
	return selectRandom(ts.config.Templates.Transaction.UserRelationshipNumbers)
}

// SelectBizReqInput randomly selects and encodes a business request input
func (ts *TemplateSelector) SelectBizReqInput() string {
	input := selectRandom(ts.config.Templates.Transaction.BizReqInputs)
	return base64.StdEncoding.EncodeToString([]byte(input))
}

// SelectBizRespOutput randomly selects and encodes a business response output
func (ts *TemplateSelector) SelectBizRespOutput() string {
	output := selectRandom(ts.config.Templates.Transaction.BizRespOutputs)
	return base64.StdEncoding.EncodeToString([]byte(output))
}

// SelectUserID randomly selects or generates a user ID
func (ts *TemplateSelector) SelectUserID() int64 {
	if ts.config.UserIDs.Mode == "fixed" {
		return selectRandomInt64(ts.config.UserIDs.FixedList)
	}

	// Generate random in range
	rangeSize := ts.config.UserIDs.RangeMax - ts.config.UserIDs.RangeMin
	n, _ := rand.Int(rand.Reader, big.NewInt(rangeSize))
	return ts.config.UserIDs.RangeMin + n.Int64()
}

// SelectPodName randomly selects a pod name
func (ts *TemplateSelector) SelectPodName() string {
	return selectRandom(ts.config.AccessLog.PodNames)
}

// SelectLogPath randomly selects a log path
func (ts *TemplateSelector) SelectLogPath() string {
	return selectRandom(ts.config.AccessLog.LogPaths)
}

// SelectLogName randomly selects a log name
func (ts *TemplateSelector) SelectLogName() string {
	return selectRandom(ts.config.AccessLog.LogNames)
}

// SelectAccessLogChannelID randomly selects a channel ID for access logs
func (ts *TemplateSelector) SelectAccessLogChannelID() int {
	return selectRandomInt(ts.config.AccessLog.ChannelIDs)
}

// SelectAPIUrl randomly selects an API URL
func (ts *TemplateSelector) SelectAPIUrl() string {
	return selectRandom(ts.config.AccessLog.APIUrls)
}

// SelectHttpStatus randomly selects an HTTP status with weighting
func (ts *TemplateSelector) SelectHttpStatus() string {
	statuses := ts.config.AccessLog.HttpStatuses
	if len(statuses) == 0 {
		return "200"
	}

	totalWeight := 0
	for _, status := range statuses {
		totalWeight += status.Weight
	}

	n, _ := rand.Int(rand.Reader, big.NewInt(int64(totalWeight)))
	randomWeight := int(n.Int64())

	currentWeight := 0
	for _, status := range statuses {
		currentWeight += status.Weight
		if randomWeight < currentWeight {
			return status.Value
		}
	}

	return statuses[0].Value
}

// selectRandom selects a random string from a slice
func selectRandom(items []string) string {
	if len(items) == 0 {
		return ""
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(items))))
	return items[n.Int64()]
}

// selectRandomInt selects a random int from a slice
func selectRandomInt(items []int) int {
	if len(items) == 0 {
		return 0
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(items))))
	return items[n.Int64()]
}

// selectRandomInt64 selects a random int64 from a slice
func selectRandomInt64(items []int64) int64 {
	if len(items) == 0 {
		return 0
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(items))))
	return items[n.Int64()]
}
