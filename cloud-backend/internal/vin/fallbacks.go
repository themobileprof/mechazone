package vin

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Fallbacks struct {
	CarAPIToken  string
	CarAPISecret string
	VincarioKey  string
	VincarioSec  string
	HTTP         *http.Client
}

func NewFallbacks(carToken, carSecret, vincarioKey, vincarioSecret string) *Fallbacks {
	return &Fallbacks{
		CarAPIToken:  strings.TrimSpace(carToken),
		CarAPISecret: strings.TrimSpace(carSecret),
		VincarioKey:  strings.TrimSpace(vincarioKey),
		VincarioSec:  strings.TrimSpace(vincarioSecret),
		HTTP:         &http.Client{Timeout: 8 * time.Second},
	}
}

func (f *Fallbacks) Enabled() bool {
	if f == nil {
		return false
	}
	return (f.CarAPIToken != "" && f.CarAPISecret != "") || (f.VincarioKey != "" && f.VincarioSec != "")
}

func (f *Fallbacks) Decode(ctx context.Context, vin string) (Decode, error) {
	if f.CarAPIToken != "" && f.CarAPISecret != "" {
		dec, err := f.carAPI(ctx, vin)
		if err == nil && !dec.Empty {
			return dec, nil
		}
	}
	if f.VincarioKey != "" && f.VincarioSec != "" {
		return f.vincario(ctx, vin)
	}
	return Decode{}, fmt.Errorf("no VIN fallback credentials configured")
}

func (f *Fallbacks) carAPI(ctx context.Context, vin string) (Decode, error) {
	loginBody, _ := json.Marshal(map[string]string{
		"api_token":  f.CarAPIToken,
		"api_secret": f.CarAPISecret,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://carapi.app/api/auth/login", bytes.NewReader(loginBody))
	if err != nil {
		return Decode{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.HTTP.Do(req)
	if err != nil {
		return Decode{}, fmt.Errorf("carapi login: %w", err)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Decode{}, fmt.Errorf("carapi login status %d", resp.StatusCode)
	}
	var token struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &token); err != nil || token.Token == "" {
		return Decode{}, fmt.Errorf("carapi login token missing")
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, "https://carapi.app/api/vin/"+vin, nil)
	if err != nil {
		return Decode{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token.Token)
	resp, err = f.HTTP.Do(req)
	if err != nil {
		return Decode{}, fmt.Errorf("carapi vin: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Decode{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Decode{}, fmt.Errorf("carapi vin status %d", resp.StatusCode)
	}
	var parsed struct {
		Make  string `json:"make"`
		Model string `json:"model"`
		Year  int    `json:"year"`
		Specs struct {
			Make  string `json:"make"`
			Model string `json:"model"`
			Year  int    `json:"year"`
		} `json:"specs"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Decode{}, err
	}
	makeName := firstNonEmpty(parsed.Make, parsed.Specs.Make)
	model := firstNonEmpty(parsed.Model, parsed.Specs.Model)
	year := parsed.Year
	if year == 0 {
		year = parsed.Specs.Year
	}
	return Decode{
		VIN: vin, Make: makeName, Model: model, Year: year,
		Source: "carapi", Raw: json.RawMessage(raw),
		Empty: makeName == "" && model == "" && year == 0,
	}, nil
}

func (f *Fallbacks) vincario(ctx context.Context, vin string) (Decode, error) {
	vin = strings.ToUpper(vin)
	sum := sha1.Sum([]byte(vin + "|decode|" + f.VincarioKey + "|" + f.VincarioSec))
	control := hex.EncodeToString(sum[:])[:10]
	url := fmt.Sprintf("https://api.vincario.com/3.2/%s/%s/decode/%s.json", f.VincarioKey, control, vin)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Decode{}, err
	}
	resp, err := f.HTTP.Do(req)
	if err != nil {
		return Decode{}, fmt.Errorf("vincario: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Decode{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Decode{}, fmt.Errorf("vincario status %d", resp.StatusCode)
	}
	var parsed struct {
		Decode []struct {
			Label string `json:"label"`
			Value string `json:"value"`
		} `json:"decode"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Decode{}, err
	}
	fields := map[string]string{}
	for _, row := range parsed.Decode {
		fields[strings.ToLower(row.Label)] = row.Value
	}
	year := 0
	fmt.Sscanf(fields["model year"], "%d", &year)
	makeName := firstNonEmpty(fields["make"], fields["manufacturer"])
	model := fields["model"]
	return Decode{
		VIN: vin, Make: makeName, Model: model, Year: year,
		Source: "vincario", Raw: json.RawMessage(raw),
		Empty: makeName == "" && model == "" && year == 0,
	}, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
