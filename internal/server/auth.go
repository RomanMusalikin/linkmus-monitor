package server

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// HasUsers возвращает true если в системе уже есть хотя бы один пользователь
func HasUsers(db *sql.DB) (bool, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count > 0, err
}

// RegisterUser создаёт пользователя с bcrypt-хэшем пароля, возвращает ID нового пользователя.
func RegisterUser(db *sql.DB, login, password string) (int64, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	res, err := db.Exec(`INSERT INTO users (login, password) VALUES (?, ?)`, login, string(hash))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// AuthenticateUser проверяет логин/пароль, возвращает ID пользователя
func AuthenticateUser(db *sql.DB, login, password string) (int64, error) {
	var id int64
	var hash string
	err := db.QueryRow(`SELECT id, password FROM users WHERE login = ?`, login).Scan(&id, &hash)
	if err != nil {
		return 0, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return 0, err
	}
	return id, nil
}

// CreateSession генерирует токен сессии и сохраняет его в БД (срок — 30 дней)
func CreateSession(db *sql.DB, userID int64) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	expires := time.Now().Add(30 * 24 * time.Hour)
	_, err := db.Exec(
		`INSERT INTO sessions (token, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		token, userID, time.Now().UTC(), expires.UTC(),
	)
	return token, err
}

// ValidateSession проверяет что токен существует и не истёк
func ValidateSession(db *sql.DB, token string) bool {
	if token == "" {
		return false
	}
	var expires time.Time
	err := db.QueryRow(`SELECT expires_at FROM sessions WHERE token = ?`, token).Scan(&expires)
	if err != nil {
		return false
	}
	return time.Now().Before(expires)
}

// DeleteSession удаляет сессию (logout)
func DeleteSession(db *sql.DB, token string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}
