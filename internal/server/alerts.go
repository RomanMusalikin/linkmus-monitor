package server

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"sync"
	"time"
)

type AlertSettings struct {
	SMTPHost      string `json:"smtpHost"`
	SMTPPort      int    `json:"smtpPort"`
	SMTPUser      string `json:"smtpUser"`
	SMTPPass      string `json:"smtpPass"`
	FromEmail     string `json:"fromEmail"`
	ToEmail       string `json:"toEmail"`
	CPUThreshold  int    `json:"cpuThreshold"`
	RAMThreshold  int    `json:"ramThreshold"`
	CooldownMin   int    `json:"cooldownMin"`
	Enabled       bool   `json:"enabled"`
}

func GetAlertSettings(db *sql.DB) AlertSettings {
	var s AlertSettings
	s.SMTPPort = 587
	s.CooldownMin = 30
	db.QueryRow(`SELECT smtp_host,smtp_port,smtp_user,smtp_pass,from_email,to_email,
		cpu_threshold,ram_threshold,cooldown_min,enabled FROM alert_settings WHERE id=1`).
		Scan(&s.SMTPHost, &s.SMTPPort, &s.SMTPUser, &s.SMTPPass,
			&s.FromEmail, &s.ToEmail, &s.CPUThreshold, &s.RAMThreshold, &s.CooldownMin, &s.Enabled)
	return s
}

func SaveAlertSettings(db *sql.DB, s AlertSettings) error {
	_, err := db.Exec(`INSERT INTO alert_settings
		(id,smtp_host,smtp_port,smtp_user,smtp_pass,from_email,to_email,
		 cpu_threshold,ram_threshold,cooldown_min,enabled)
		VALUES (1,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
		  smtp_host=excluded.smtp_host, smtp_port=excluded.smtp_port,
		  smtp_user=excluded.smtp_user, smtp_pass=excluded.smtp_pass,
		  from_email=excluded.from_email, to_email=excluded.to_email,
		  cpu_threshold=excluded.cpu_threshold, ram_threshold=excluded.ram_threshold,
		  cooldown_min=excluded.cooldown_min, enabled=excluded.enabled`,
		s.SMTPHost, s.SMTPPort, s.SMTPUser, s.SMTPPass, s.FromEmail, s.ToEmail,
		s.CPUThreshold, s.RAMThreshold, s.CooldownMin, s.Enabled)
	return err
}

// SendTestEmail отправляет тестовое письмо с текущими настройками SMTP.
func SendTestEmail(s AlertSettings) error {
	return sendMail(s, "✅ LinkMus Monitor — тест уведомлений",
		"Если вы получили это письмо, настройка SMTP работает корректно.")
}

// ── Трекер кулдаунов (в памяти) ─────────────────────────────────────────────

var (
	cooldownMu   sync.Mutex
	lastAlertSent = map[string]time.Time{} // "nodeName:cpu" / "nodeName:ram"
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
	if !s.Enabled || s.ToEmail == "" || s.SMTPHost == "" {
		return
	}

	cachedNodes, _ := getCachedNodes()
	nodes := make([]NodeSummary, len(cachedNodes))
	copy(nodes, cachedNodes)

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
				subject := fmt.Sprintf("⚠️ CPU %d%% на узле %s", int(n.CPU), displayName)
				body := fmt.Sprintf(
					"Узел: %s\nМетрика: CPU\nЗначение: %d%%\nПорог: %d%%\nВремя: %s",
					displayName, int(n.CPU), s.CPUThreshold,
					time.Now().Format("02.01.2006 15:04:05"),
				)
				if err := sendMail(s, subject, body); err != nil {
					log.Printf("⚠️  alert email CPU %s: %v", n.Name, err)
				} else {
					log.Printf("📧 Алерт CPU отправлен: %s = %d%%", displayName, int(n.CPU))
				}
			}
		}

		if s.RAMThreshold > 0 && n.RAMTotal > 0 {
			ramPct := int(n.RAMUsed / n.RAMTotal * 100)
			if ramPct >= s.RAMThreshold {
				key := n.Name + ":ram"
				if canAlert(key, s.CooldownMin) {
					subject := fmt.Sprintf("⚠️ RAM %d%% на узле %s", ramPct, displayName)
					body := fmt.Sprintf(
						"Узел: %s\nМетрика: RAM\nЗначение: %d%% (%.1f / %.1f GB)\nПорог: %d%%\nВремя: %s",
						displayName, ramPct, n.RAMUsed, n.RAMTotal, s.RAMThreshold,
						time.Now().Format("02.01.2006 15:04:05"),
					)
					if err := sendMail(s, subject, body); err != nil {
						log.Printf("⚠️  alert email RAM %s: %v", n.Name, err)
					} else {
						log.Printf("📧 Алерт RAM отправлен: %s = %d%%", displayName, ramPct)
					}
				}
			}
		}
	}
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
