// Package claude turns numeric backtest results into a plain-language summary
// using the Anthropic Messages API.
package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"MFBasketBacktester/internal/models"
)

const (
	apiURL     = "https://api.anthropic.com/v1/messages"
	apiVersion = "2023-06-01"
	model      = "claude-haiku-4-5"
	maxTokens  = 400
)

type messageRequest struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	Messages  []messageContent `json:"messages"`
}

type messageContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messageResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Summarize asks Claude to describe the backtest result in a few plain
// sentences an everyday investor can understand.
func Summarize(ctx context.Context, apiKey string, basketName string, result models.BacktestResult) (string, error) {
	if apiKey == "" {
		return "", errors.New("claude api key is not configured")
	}

	investment := "lump-sum"
	if result.Mode == "sip" {
		investment = "monthly SIP"
	}

	prompt := fmt.Sprintf(
		"A user ran a %s backtest on a mutual fund basket named %q. Results:\n"+
			"- CAGR: %.2f%%\n"+
			"- XIRR: %.2f%%\n"+
			"- Maximum drawdown: %.2f%%\n"+
			"- Total invested: %.2f\n"+
			"- Final portfolio value: %.2f\n\n"+
			"Write a concise, plain-language summary (3-4 sentences) for a non-expert "+
			"investor. Explain what these numbers mean for the basket's performance "+
			"and risk. For a SIP, note that XIRR is the more meaningful return "+
			"measure. Do not give financial advice.",
		investment, basketName, result.CAGR, result.XIRR, result.Drawdown,
		result.TotalInvested, result.FinalValue,
	)

	body, err := json.Marshal(messageRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  []messageContent{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", apiVersion)
	req.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var parsed messageResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("unexpected response from claude api: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("claude api error: %s", parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("claude api returned status %d", resp.StatusCode)
	}

	for _, c := range parsed.Content {
		if c.Type == "text" {
			return c.Text, nil
		}
	}
	return "", errors.New("claude api returned no text content")
}
