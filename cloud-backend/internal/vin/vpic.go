package vin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Decode is a VIN decode result. Persist it in vin_decode_cache; do not re-query a hit.
type Decode struct {
	VIN     string
	Make    string
	Model   string
	Year    int
	Source  string
	Raw     json.RawMessage
	Empty   bool
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func Normalize(vin string) (string, error) {
	v := strings.ToUpper(strings.TrimSpace(vin))
	if len(v) != 17 {
		return "", fmt.Errorf("VIN must be 17 characters")
	}
	for _, r := range v {
		if !unicode.IsDigit(r) && !unicode.IsLetter(r) {
			return "", fmt.Errorf("VIN contains invalid characters")
		}
		if r == 'I' || r == 'O' || r == 'Q' {
			return "", fmt.Errorf("VIN contains forbidden character %q", r)
		}
	}
	return v, nil
}

func (c *Client) Decode(ctx context.Context, vin string) (Decode, error) {
	url := fmt.Sprintf("%s/vehicles/DecodeVinValues/%s?format=json", c.BaseURL, vin)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Decode{}, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return Decode{}, fmt.Errorf("vpic request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Decode{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Decode{}, fmt.Errorf("vpic status %d", resp.StatusCode)
	}

	var parsed struct {
		Results []map[string]any `json:"Results"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return Decode{}, fmt.Errorf("vpic decode: %w", err)
	}
	if len(parsed.Results) == 0 {
		return Decode{VIN: vin, Source: "vpic", Raw: body, Empty: true}, nil
	}
	row := parsed.Results[0]
	makeName := stringField(row, "Make")
	model := stringField(row, "Model")
	year := intField(row, "ModelYear")
	empty := makeName == "" && model == "" && year == 0
	return Decode{
		VIN:    vin,
		Make:   makeName,
		Model:  model,
		Year:   year,
		Source: "vpic",
		Raw:    json.RawMessage(body),
		Empty:  empty,
	}, nil
}

func stringField(row map[string]any, key string) string {
	v, ok := row[key]
	if !ok || v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" || strings.EqualFold(s, "null") {
		return ""
	}
	return s
}

func intField(row map[string]any, key string) int {
	s := stringField(row, key)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
