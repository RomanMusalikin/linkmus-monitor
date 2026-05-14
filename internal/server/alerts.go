package server

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"sync"
	"time"
)

type AlertSettings struct {
	SMTPHost     string `json:"smtpHost"`
	SMTPPort     int    `json:"smtpPort"`
	SMTPUser     string `json:"smtpUser"`
	SMTPPass     string `json:"smtpPass"`
	FromEmail    string `json:"fromEmail"`
	ToEmail      string `json:"toEmail"`
	CPUThreshold int    `json:"cpuThreshold"`
	RAMThreshold int    `json:"ramThreshold"`
	CooldownMin  int    `json:"cooldownMin"`
	Enabled      bool   `json:"enabled"`

	TGBotToken string `json:"tgBotToken"`
	TGChatID   string `json:"tgChatID"`
	TGTopicID  int    `json:"tgTopicID"`
	TGEnabled  bool   `json:"tgEnabled"`

	UTCOffset int `json:"utcOffset"` // смещение от UTC в часах, например 3 для Москвы
}

func GetAlertSettings(db *sql.DB) AlertSettings {
	var s AlertSettings
	s.SMTPPort = 587
	s.CooldownMin = 30
	db.QueryRow(`SELECT smtp_host,smtp_port,smtp_user,smtp_pass,from_email,to_email,
		cpu_threshold,ram_threshold,cooldown_min,enabled,
		COALESCE(tg_bot_token,''),COALESCE(tg_chat_id,''),COALESCE(tg_topic_id,0),COALESCE(tg_enabled,0),
		COALESCE(utc_offset,0)
		FROM alert_settings WHERE id=1`).
		Scan(&s.SMTPHost, &s.SMTPPort, &s.SMTPUser, &s.SMTPPass,
			&s.FromEmail, &s.ToEmail, &s.CPUThreshold, &s.RAMThreshold, &s.CooldownMin, &s.Enabled,
			&s.TGBotToken, &s.TGChatID, &s.TGTopicID, &s.TGEnabled,
			&s.UTCOffset)
	return s
}

func SaveAlertSettings(db *sql.DB, s AlertSettings) error {
	s.TGBotToken = strings.TrimSpace(s.TGBotToken)
	s.TGChatID = strings.TrimSpace(s.TGChatID)
	_, err := db.Exec(`INSERT INTO alert_settings
		(id,smtp_host,smtp_port,smtp_user,smtp_pass,from_email,to_email,
		 cpu_threshold,ram_threshold,cooldown_min,enabled,
		 tg_bot_token,tg_chat_id,tg_topic_id,tg_enabled,utc_offset)
		VALUES (1,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
		  smtp_host=excluded.smtp_host, smtp_port=excluded.smtp_port,
		  smtp_user=excluded.smtp_user, smtp_pass=excluded.smtp_pass,
		  from_email=excluded.from_email, to_email=excluded.to_email,
		  cpu_threshold=excluded.cpu_threshold, ram_threshold=excluded.ram_threshold,
		  cooldown_min=excluded.cooldown_min, enabled=excluded.enabled,
		  tg_bot_token=excluded.tg_bot_token, tg_chat_id=excluded.tg_chat_id,
		  tg_topic_id=excluded.tg_topic_id, tg_enabled=excluded.tg_enabled,
		  utc_offset=excluded.utc_offset`,
		s.SMTPHost, s.SMTPPort, s.SMTPUser, s.SMTPPass, s.FromEmail, s.ToEmail,
		s.CPUThreshold, s.RAMThreshold, s.CooldownMin, s.Enabled,
		s.TGBotToken, s.TGChatID, s.TGTopicID, s.TGEnabled, s.UTCOffset)
	return err
}

// SendTestEmail отправляет тестовое письмо с текущими настройками SMTP.
func SendTestEmail(s AlertSettings) error {
	return sendMail(s, "✅ LinkMus Monitor — тест уведомлений",
		"Если вы получили это письмо, настройка SMTP работает корректно.")
}

// SendTestTelegram отправляет тестовое сообщение в Telegram.
func SendTestTelegram(s AlertSettings) error {
	return sendTelegram(s, "✅ *LinkMus Monitor* — тест уведомлений\n\nЕсли вы получили это сообщение, настройка Telegram работает корректно.")
}

// ── Трекер кулдаунов (в памяти) ─────────────────────────────────────────────

var (
	cooldownMu    sync.Mutex
	lastAlertSent = map[string]time.Time{} // "nodeName:cpu" / "nodeName:ram"

	nodeOnlineStateMu sync.Mutex
	nodeOnlineState   = map[string]bool{} // node name → последнее известное состояние
)

func canAlert(key string, cooldownMin int) bool {
	cooldownMu.Lock()
	defer cooldownMu.Unlock()
	last, ok := lastAlertSent[key]
	if !ok || time.Since(last) >= time.Duration(cooldownMin)*time.Minute {
		lastAlertSent[key] = time.Now()
		return true
	}
	return false
}

// ── Горутина-чекер ───────────────────────────────────────────────────────────

func StartAlertChecker(db *sql.DB) {
	go func() {
		for {
			time.Sleep(60 * time.Second)
			checkAlerts(db)
		}
	}()
}

func checkAlerts(db *sql.DB) {
	s := GetAlertSettings(db)
	emailOK := s.Enabled && s.ToEmail != "" && s.SMTPHost != ""
	tgOK := s.TGEnabled && s.TGBotToken != "" && s.TGChatID != ""
	if !emailOK && !tgOK {
		return
	}

	cachedNodes, _ := getCachedNodes()
	nodes := make([]NodeSummary, len(cachedNodes))
	copy(nodes, cachedNodes)

	// ── Алерты на смену online/offline ──────────────────────────────────────
	type stateChange struct {
		displayName string
		wentOffline bool
	}
	var changes []stateChange

	nodeOnlineStateMu.Lock()
	for _, n := range nodes {
		dn := n.DisplayName
		if dn == "" {
			dn = n.Name
		}
		prev, known := nodeOnlineState[n.Name]
		nodeOnlineState[n.Name] = n.Online
		if known && prev != n.Online {
			changes = append(changes, stateChange{dn, !n.Online})
		}
	}
	nodeOnlineStateMu.Unlock()

	if emailOK || tgOK {
		for _, ch := range changes {
			ts := time.Now().In(time.FixedZone("Moscow", 3*60*60)).Format("02.01.2006 15:04:05 MSK")
			if ch.wentOffline {
				if emailOK {
					subj := fmt.Sprintf("🔴 Узел недоступен: %s", ch.displayName)
					body := fmt.Sprintf("Узел: %s\nСтатус: недоступен\nВремя: %s", ch.displayName, ts)
					if err := sendMail(s, subj, body); err != nil {
						log.Printf("⚠️  alert email offline %s: %v", ch.displayName, err)
					} else {
						log.Printf("📧 Алерт offline: %s", ch.displayName)
					}
				}
				if tgOK {
					msg := fmt.Sprintf("🔴 Узел *%s* недоступен\nВремя: %s", ch.displayName, ts)
					if err := sendTelegram(s, msg); err != nil {
						log.Printf("⚠️  alert tg offline %s: %v", ch.displayName, err)
					} else {
						log.Printf("📨 Алерт TG offline: %s", ch.displayName)
					}
				}
			} else {
				if emailOK {
					subj := fmt.Sprintf("🟢 Узел снова доступен: %s", ch.displayName)
					body := fmt.Sprintf("Узел: %s\nСтатус: доступен\nВремя: %s", ch.displayName, ts)
					if err := sendMail(s, subj, body); err != nil {
						log.Printf("⚠️  alert email online %s: %v", ch.displayName, err)
					} else {
						log.Printf("📧 Алерт online: %s", ch.displayName)
					}
				}
				if tgOK {
					msg := fmt.Sprintf("🟢 Узел *%s* снова доступен\nВремя: %s", ch.displayName, ts)
					if err := sendTelegram(s, msg); err != nil {
						log.Printf("⚠️  alert tg online %s: %v", ch.displayName, err)
					} else {
						log.Printf("📨 Алерт TG online: %s", ch.displayName)
					}
				}
			}
		}
	}
	// ────────────────────────────────────────────────────────────────────────

	for _, n := range nodes {
		if !n.Online {
			continue
		}
		displayName := n.DisplayName
		if displayName == "" {
			displayName = n.Name
		}

		if s.CPUThreshold > 0 && int(n.CPU) >= s.CPUThreshold {
			key := n.Name + ":cpu"
			if canAlert(key, s.CooldownMin) {
				ts := time.Now().In(time.FixedZone("Moscow", 3*60*60)).Format("02.01.2006 15:04:05 MSK")
				if emailOK {
					subject := fmt.Sprintf("⚠️ CPU %d%% на узле %s", int(n.CPU), displayName)
					body := fmt.Sprintf("Узел: %s\nМетрика: CPU\nЗначение: %d%%\nПорог: %d%%\nВремя: %s",
						displayName, int(n.CPU), s.CPUThreshold, ts)
					if err := sendMail(s, subject, body); err != nil {
						log.Printf("⚠️  alert email CPU %s: %v", n.Name, err)
					} else {
						log.Printf("📧 Алерт CPU отправлен: %s = %d%%", displayName, int(n.CPU))
					}
				}
				if tgOK {
					msg := fmt.Sprintf("⚠️ *CPU %d%%* на узле *%s*\nПорог: %d%%\nВремя: %s",
						int(n.CPU), displayName, s.CPUThreshold, ts)
					if err := sendTelegram(s, msg); err != nil {
						log.Printf("⚠️  alert tg CPU %s: %v", n.Name, err)
					} else {
						log.Printf("📨 Алерт TG CPU: %s = %d%%", displayName, int(n.CPU))
					}
				}
			}
		}

		if s.RAMThreshold > 0 && n.RAMTotal > 0 {
			ramPct := int(n.RAMUsed / n.RAMTotal * 100)
			if ramPct >= s.RAMThreshold {
				key := n.Name + ":ram"
				if canAlert(key, s.CooldownMin) {
					ts := time.Now().In(time.FixedZone("Moscow", 3*60*60)).Format("02.01.2006 15:04:05 MSK")
					if emailOK {
						subject := fmt.Sprintf("⚠️ RAM %d%% на узле %s", ramPct, displayName)
						body := fmt.Sprintf("Узел: %s\nМетрика: RAM\nЗначение: %d%% (%.1f / %.1f GB)\nПорог: %d%%\nВремя: %s",
							displayName, ramPct, n.RAMUsed, n.RAMTotal, s.RAMThreshold, ts)
						if err := sendMail(s, subject, body); err != nil {
							log.Printf("⚠️  alert email RAM %s: %v", n.Name, err)
						} else {
							log.Printf("📧 Алерт RAM отправлен: %s = %d%%", displayName, ramPct)
						}
					}
					if tgOK {
						msg := fmt.Sprintf("⚠️ *RAM %d%%* на узле *%s*\n%.1f / %.1f GB\nПорог: %d%%\nВремя: %s",
							ramPct, displayName, n.RAMUsed, n.RAMTotal, s.RAMThreshold, ts)
						if err := sendTelegram(s, msg); err != nil {
							log.Printf("⚠️  alert tg RAM %s: %v", n.Name, err)
						} else {
							log.Printf("📨 Алерт TG RAM: %s = %d%%", displayName, ramPct)
						}
					}
				}
			}
		}
	}
}

// ── Telegram-отправка ────────────────────────────────────────────────────────

func sendTelegram(s AlertSettings, text string) error {
	token := strings.TrimSpace(s.TGBotToken)
	chatID := strings.TrimSpace(s.TGChatID)
	if token == "" || chatID == "" {
		return fmt.Errorf("не заполнены токен бота или Chat ID")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	params := url.Values{
		"chat_id":    {chatID},
		"text":       {text},
		"parse_mode": {"Markdown"},
	}
	if s.TGTopicID > 0 {
		params.Set("message_thread_id", fmt.Sprintf("%d", s.TGTopicID))
	}

	resp, err := http.PostForm(apiURL, params)
	if err != nil {
		return fmt.Errorf("сетевая ошибка: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
	}
	json.Unmarshal(body, &result)
	if !result.OK {
		desc := result.Description
		if desc == "Not Found" {
			desc = "Not Found — проверьте токен бота (возможно, скопирован с лишними символами или бот удалён)"
		}
		return fmt.Errorf("Telegram: %s", desc)
	}
	return nil
}

// ── SMTP-отправка ────────────────────────────────────────────────────────────

func sendMail(s AlertSettings, subject, body string) error {
	from := s.FromEmail
	if from == "" {
		from = s.SMTPUser
	}
	addr := fmt.Sprintf("%s:%d", s.SMTPHost, s.SMTPPort)

	header := strings.Join([]string{
		"From: LinkMus Monitor <" + from + ">",
		"To: " + s.ToEmail,
		"Subject: " + subject,
		"Content-Type: text/plain; charset=UTF-8",
		"MIME-Version: 1.0",
		"",
		body,
	}, "\r\n")

	auth := smtp.PlainAuth("", s.SMTPUser, s.SMTPPass, s.SMTPHost)

	// Порт 465 — implicit TLS; остальные — STARTTLS
	if s.SMTPPort == 465 {
		tlsCfg := &tls.Config{ServerName: s.SMTPHost}
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return err
		}
		client, err := smtp.NewClient(conn, s.SMTPHost)
		if err != nil {
			return err
		}
		defer client.Close()
		if err = client.Auth(auth); err != nil {
			return err
		}
		if err = client.Mail(from); err != nil {
			return err
		}
		if err = client.Rcpt(s.ToEmail); err != nil {
			return err
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(w, header)
		if err != nil {
			return err
		}
		return w.Close()
	}

	return smtp.SendMail(addr, auth, from, []string{s.ToEmail}, []byte(header))
}
