package server

import (
	"bytes"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// GigachatSettings — учётные данные GigaChat API
type GigachatSettings struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Scope        string `json:"scope"`
}

func GetGigachatSettings(db *sql.DB) GigachatSettings {
	var s GigachatSettings
	db.QueryRow(`SELECT client_id, client_secret, scope FROM gigachat_settings WHERE id=1`).
		Scan(&s.ClientID, &s.ClientSecret, &s.Scope)
	if s.Scope == "" {
		s.Scope = "GIGACHAT_API_PERS"
	}
	return s
}

func SaveGigachatSettings(db *sql.DB, s GigachatSettings) error {
	if s.Scope == "" {
		s.Scope = "GIGACHAT_API_PERS"
	}
	_, err := db.Exec(`
		INSERT INTO gigachat_settings(id, client_id, client_secret, scope) VALUES(1,?,?,?)
		ON CONFLICT(id) DO UPDATE SET client_id=excluded.client_id, client_secret=excluded.client_secret, scope=excluded.scope`,
		s.ClientID, s.ClientSecret, s.Scope)
	return err
}

var (
	gcTokenMu  sync.Mutex
	gcToken    string
	gcTokenExp time.Time
)

// gigachatHTTPClient — клиент без проверки TLS (GigaChat использует российский УЦ)
var gigachatHTTPClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	},
	Timeout: 90 * time.Second,
}

// GigachatGetToken получает OAuth-токен, кеширует до истечения.
func GigachatGetToken(s GigachatSettings) (string, error) {
	gcTokenMu.Lock()
	defer gcTokenMu.Unlock()

	if gcToken != "" && time.Now().Before(gcTokenExp) {
		return gcToken, nil
	}

	form := url.Values{"scope": {s.Scope}}

	req, err := http.NewRequest("POST",
		"https://ngw.devices.sberbank.ru:9443/api/v2/oauth",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", err
	}
	// ClientSecret здесь — это Authorization Key из личного кабинета Sber,
	// уже готовый base64-токен, вставляется напрямую
	req.Header.Set("Authorization", "Basic "+s.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("RqUID", uuid.NewString())

	resp, err := gigachatHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresAt   int64  `json:"expires_at"` // milliseconds
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("auth decode: %w", err)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("empty access_token from GigaChat auth")
	}

	gcToken = result.AccessToken
	gcTokenExp = time.Unix(result.ExpiresAt/1000, 0).Add(-60 * time.Second)
	return gcToken, nil
}

// GigachatChat отправляет текст в GigaChat и возвращает ответ.
func GigachatChat(token, prompt string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"model": "GigaChat",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST",
		"https://gigachat.devices.sberbank.ru/api/v1/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := gigachatHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("chat request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("chat returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("chat decode: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in GigaChat response")
	}
	return result.Choices[0].Message.Content, nil
}
