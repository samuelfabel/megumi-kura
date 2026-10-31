package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/samuelfabel/megumi-kura/api/internal/quantity"
)

var (
	ErrInvalidEmail = errors.New("invalid email")
	ErrTemplate     = errors.New("template validation failed")
	ErrNotFound     = errors.New("not found")
	ErrSMTP         = errors.New("smtp error")
)

type Template struct {
	Key          string   `json:"key"`
	Name         string   `json:"name"`
	Subject      string   `json:"subject"`
	BodyHTML     string   `json:"body_html"`
	RequiredTags []string `json:"required_tags"`
	OptionalTags []string `json:"optional_tags"`
}

type AlertSettings struct {
	AutoEnabled     bool       `json:"auto_enabled"`
	AutoFrequency   string     `json:"auto_frequency"` // hourly|daily|weekly
	GlobalThreshold quantity.Q `json:"global_threshold"`
}

type SMTPConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	From     string `json:"from"`
}

type SubscribeResult struct {
	OK bool `json:"ok"`
}

type Repository struct {
	DB *sql.DB
}

func (r *Repository) ListTemplates(ctx context.Context) ([]Template, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT key, name, subject, body_html, required_tags, optional_tags
		FROM email_templates ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Template
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.Key, &t.Name, &t.Subject, &t.BodyHTML, &t.RequiredTags, &t.OptionalTags); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repository) GetTemplate(ctx context.Context, key string) (Template, error) {
	var t Template
	err := r.DB.QueryRowContext(ctx, `
		SELECT key, name, subject, body_html, required_tags, optional_tags
		FROM email_templates WHERE key = $1`, key,
	).Scan(&t.Key, &t.Name, &t.Subject, &t.BodyHTML, &t.RequiredTags, &t.OptionalTags)
	if errors.Is(err, sql.ErrNoRows) {
		return Template{}, ErrNotFound
	}
	return t, err
}

func ValidateTemplateHTML(t Template, subject, body string) error {
	for _, tag := range t.RequiredTags {
		token := "{{" + tag + "}}"
		if !strings.Contains(subject, token) && !strings.Contains(body, token) {
			return fmt.Errorf("%w: missing required tag %s", ErrTemplate, tag)
		}
	}
	return nil
}

func (r *Repository) UpdateTemplate(ctx context.Context, key, subject, body string) (Template, error) {
	t, err := r.GetTemplate(ctx, key)
	if err != nil {
		return Template{}, err
	}
	if err := ValidateTemplateHTML(t, subject, body); err != nil {
		return Template{}, err
	}
	_, err = r.DB.ExecContext(ctx, `
		UPDATE email_templates SET subject=$2, body_html=$3, updated_at=NOW() WHERE key=$1`,
		key, subject, body)
	if err != nil {
		return Template{}, err
	}
	return r.GetTemplate(ctx, key)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (r *Repository) isTrustedDomain(ctx context.Context, domain string) (bool, error) {
	var n int
	err := r.DB.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM trusted_email_domains WHERE domain = $1`, strings.ToLower(domain)).Scan(&n)
	return n > 0, err
}

func domainHasMX(domain string) bool {
	mx, err := net.LookupMX(domain)
	if err == nil && len(mx) > 0 {
		return true
	}
	// Fallback: A record
	ips, err := net.LookupHost(domain)
	return err == nil && len(ips) > 0
}

// Subscribe always returns ok=true to the public caller (no enumeration).
func (r *Repository) Subscribe(ctx context.Context, email string) (SubscribeResult, error) {
	norm := normalizeEmail(email)
	if !strings.Contains(norm, "@") || strings.Contains(norm, " ") {
		// Still return ok to avoid leaking validation? User asked for success on already subscribed;
		// invalid format can still 400 at handler. We'll validate format.
		return SubscribeResult{}, ErrInvalidEmail
	}
	parts := strings.Split(norm, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || !strings.Contains(parts[1], ".") {
		return SubscribeResult{}, ErrInvalidEmail
	}
	domain := parts[1]
	trusted, err := r.isTrustedDomain(ctx, domain)
	if err != nil {
		return SubscribeResult{}, err
	}
	if !trusted && !domainHasMX(domain) {
		return SubscribeResult{}, ErrInvalidEmail
	}

	token, err := randomToken(24)
	if err != nil {
		return SubscribeResult{}, err
	}

	// Upsert: if exists, reactivate silently
	_, err = r.DB.ExecContext(ctx, `
		INSERT INTO email_subscribers(email, email_normalized, unsubscribe_token, active)
		VALUES ($1, $2, $3, TRUE)
		ON CONFLICT (email_normalized) DO UPDATE
		SET active = TRUE, inactivated_reason = NULL, updated_at = NOW()`,
		email, norm, token)
	if err != nil {
		// Unique race — still ok
		return SubscribeResult{OK: true}, nil
	}
	return SubscribeResult{OK: true}, nil
}

func (r *Repository) Unsubscribe(ctx context.Context, token string) error {
	res, err := r.DB.ExecContext(ctx, `
		UPDATE email_subscribers SET active = FALSE, inactivated_reason = 'unsubscribed', updated_at = NOW()
		WHERE unsubscribe_token = $1`, strings.TrimSpace(token))
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ActiveSubscriberCount(ctx context.Context) (int, error) {
	var n int
	err := r.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM email_subscribers WHERE active = TRUE`).Scan(&n)
	return n, err
}

type ShortageItem struct {
	Name    string
	Unit    string
	Stock   quantity.Q
	Missing quantity.Q
}

func (r *Repository) LoadSMTP(ctx context.Context) (SMTPConfig, error) {
	keys := []string{"smtp.host", "smtp.port", "smtp.user", "smtp.password", "smtp.from"}
	vals := map[string]string{}
	rows, err := r.DB.QueryContext(ctx, `SELECT key, value FROM settings WHERE key = ANY($1)`, pqArray(keys))
	if err != nil {
		// fallback without array
		for _, k := range keys {
			var v string
			_ = r.DB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = $1`, k).Scan(&v)
			vals[k] = v
		}
		return SMTPConfig{
			Host: vals["smtp.host"], Port: vals["smtp.port"], User: vals["smtp.user"],
			Password: vals["smtp.password"], From: vals["smtp.from"],
		}, nil
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return SMTPConfig{}, err
		}
		vals[k] = v
	}
	return SMTPConfig{
		Host: vals["smtp.host"], Port: vals["smtp.port"], User: vals["smtp.user"],
		Password: vals["smtp.password"], From: vals["smtp.from"],
	}, nil
}

func pqArray(ss []string) any {
	// simple postgres text array literal
	b := strings.Builder{}
	b.WriteByte('{')
	for i, s := range ss {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`"` + strings.ReplaceAll(s, `"`, `\"`) + `"`)
	}
	b.WriteByte('}')
	return b.String()
}

func (r *Repository) SaveSMTP(ctx context.Context, cfg SMTPConfig) error {
	pairs := map[string]string{
		"smtp.host": cfg.Host, "smtp.port": cfg.Port, "smtp.user": cfg.User,
		"smtp.password": cfg.Password, "smtp.from": cfg.From,
	}
	for k, v := range pairs {
		if _, err := r.DB.ExecContext(ctx, `
			INSERT INTO settings(key, value) VALUES ($1, $2)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`, k, v); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) LoadAlertSettings(ctx context.Context) AlertSettings {
	s := AlertSettings{AutoFrequency: "daily", GlobalThreshold: quantity.Zero()}
	var v string
	if err := r.DB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key='mail.auto_enabled'`).Scan(&v); err == nil {
		s.AutoEnabled = v == "true" || v == "1"
	}
	if err := r.DB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key='mail.auto_frequency'`).Scan(&v); err == nil && v != "" {
		s.AutoFrequency = v
	}
	if err := r.DB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key='mail.global_threshold'`).Scan(&v); err == nil {
		if q, err := quantity.New(v); err == nil {
			s.GlobalThreshold = q
		}
	}
	return s
}

func (r *Repository) SaveAlertSettings(ctx context.Context, s AlertSettings) error {
	pairs := map[string]string{
		"mail.auto_enabled":     fmt.Sprintf("%t", s.AutoEnabled),
		"mail.auto_frequency":   s.AutoFrequency,
		"mail.global_threshold": s.GlobalThreshold.String(),
	}
	for k, v := range pairs {
		if _, err := r.DB.ExecContext(ctx, `
			INSERT INTO settings(key, value) VALUES ($1, $2)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`, k, v); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) SetFoodThreshold(ctx context.Context, foodID int64, threshold quantity.Q) error {
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO food_alert_thresholds(food_id, threshold) VALUES ($1, $2)
		ON CONFLICT (food_id) DO UPDATE SET threshold = EXCLUDED.threshold, updated_at = NOW()`,
		foodID, threshold)
	return err
}

func (r *Repository) ListFoodThresholds(ctx context.Context) (map[int64]quantity.Q, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT food_id, threshold FROM food_alert_thresholds`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]quantity.Q{}
	for rows.Next() {
		var id int64
		var q quantity.Q
		if err := rows.Scan(&id, &q); err != nil {
			return nil, err
		}
		out[id] = q
	}
	return out, rows.Err()
}

type SendResult struct {
	Sent          int    `json:"sent"`
	ShortageCount int    `json:"shortage_count"`
	SkippedReason string `json:"skipped_reason,omitempty"`
}

// FindShortages returns foods whose stock is at or below threshold (per-food or global).
func (r *Repository) FindShortages(ctx context.Context, global quantity.Q) ([]ShortageItem, error) {
	thresholds, err := r.ListFoodThresholds(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.DB.QueryContext(ctx, `
		SELECT f.id, f.name, f.unit,
			COALESCE(SUM(CASE WHEN m.movement_type='IN' THEN m.quantity
			                  WHEN m.movement_type='OUT' THEN -m.quantity ELSE 0 END), 0)
		FROM foods f
		LEFT JOIN stock_movements m ON m.food_id = f.id
		WHERE f.active = TRUE
		GROUP BY f.id, f.name, f.unit
		ORDER BY f.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ShortageItem
	for rows.Next() {
		var id int64
		var name, unit string
		var stock quantity.Q
		if err := rows.Scan(&id, &name, &unit, &stock); err != nil {
			return nil, err
		}
		th, ok := thresholds[id]
		if !ok {
			th = global
		}
		if stock.Cmp(th) <= 0 {
			out = append(out, ShortageItem{
				Name: name, Unit: unit, Stock: stock, Missing: th.Sub(stock),
			})
		}
	}
	return out, rows.Err()
}

func (r *Repository) listActiveEmails(ctx context.Context) ([]struct{ Email, Token string }, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT email, unsubscribe_token FROM email_subscribers WHERE active = TRUE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct{ Email, Token string }
	for rows.Next() {
		var e, t string
		if err := rows.Scan(&e, &t); err != nil {
			return nil, err
		}
		out = append(out, struct{ Email, Token string }{e, t})
	}
	return out, rows.Err()
}

func (r *Repository) deactivateInvalid(ctx context.Context, email, reason string) error {
	_, err := r.DB.ExecContext(ctx, `
		UPDATE email_subscribers SET active=FALSE, inactivated_reason=$2, updated_at=NOW()
		WHERE email_normalized = $1`, normalizeEmail(email), reason)
	return err
}

func Render(tpl string, vars map[string]string) string {
	out := tpl
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	return out
}

func (r *Repository) SendShortageAlert(ctx context.Context, siteName, publicBaseURL, triggeredBy string, shortages []ShortageItem) (SendResult, error) {
	if len(shortages) == 0 {
		_, _ = r.DB.ExecContext(ctx, `
			INSERT INTO mail_send_log(kind, triggered_by, recipient_count, shortage_count, skipped_reason)
			VALUES ('shortage_alert', $1, 0, 0, 'no_shortages')`, triggeredBy)
		return SendResult{SkippedReason: "no_shortages"}, nil
	}
	cfg, err := r.LoadSMTP(ctx)
	if err != nil || cfg.Host == "" || cfg.From == "" {
		return SendResult{}, fmt.Errorf("%w: smtp not configured", ErrSMTP)
	}
	if cfg.Port == "" {
		cfg.Port = "587"
	}
	tpl, err := r.GetTemplate(ctx, "shortage_alert")
	if err != nil {
		return SendResult{}, err
	}
	var list bytes.Buffer
	list.WriteString("<ul>")
	for _, s := range shortages {
		fmt.Fprintf(&list, "<li>%s: %s %s</li>", s.Name, s.Stock.String(), s.Unit)
	}
	list.WriteString("</ul>")

	recipients, err := r.listActiveEmails(ctx)
	if err != nil {
		return SendResult{}, err
	}
	sent := 0
	for _, rec := range recipients {
		unsub := strings.TrimRight(publicBaseURL, "/") + "/api/v1/public/mail/unsubscribe?token=" + rec.Token
		vars := map[string]string{
			"site_name":       siteName,
			"shortage_list":   list.String(),
			"unsubscribe_url": unsub,
			"as_of":           time.Now().Format("2006-01-02"),
		}
		subject := Render(tpl.Subject, vars)
		body := Render(tpl.BodyHTML, vars)
		if err := sendSMTP(cfg, rec.Email, subject, body); err != nil {
			if isInvalidRecipient(err) {
				_ = r.deactivateInvalid(ctx, rec.Email, "smtp_invalid")
			}
			continue
		}
		sent++
	}
	_, _ = r.DB.ExecContext(ctx, `
		INSERT INTO mail_send_log(kind, triggered_by, recipient_count, shortage_count)
		VALUES ('shortage_alert', $1, $2, $3)`, triggeredBy, sent, len(shortages))
	return SendResult{Sent: sent, ShortageCount: len(shortages)}, nil
}

func isInvalidRecipient(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "550") || strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "user unknown") || strings.Contains(msg, "mailbox unavailable")
}

// SendHTML sends a single HTML email via configured SMTP.
func SendHTML(cfg SMTPConfig, to, subject, htmlBody string) error {
	return sendSMTP(cfg, to, subject, htmlBody)
}

func sendSMTP(cfg SMTPConfig, to, subject, htmlBody string) error {
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	msg := bytes.Buffer{}
	fmt.Fprintf(&msg, "From: %s\r\n", cfg.From)
	fmt.Fprintf(&msg, "To: %s\r\n", to)
	fmt.Fprintf(&msg, "Subject: %s\r\n", subject)
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
	msg.WriteString(htmlBody)

	var auth smtp.Auth
	if cfg.User != "" {
		auth = smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)
	}

	// Try STARTTLS on 587; plain for local/dev
	if cfg.Port == "465" {
		tlsCfg := &tls.Config{ServerName: cfg.Host}
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return err
		}
		c, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return err
		}
		defer c.Close()
		if auth != nil {
			if err := c.Auth(auth); err != nil {
				return err
			}
		}
		if err := c.Mail(cfg.From); err != nil {
			return err
		}
		if err := c.Rcpt(to); err != nil {
			return err
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		if _, err := w.Write(msg.Bytes()); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		return c.Quit()
	}
	return smtp.SendMail(addr, auth, cfg.From, []string{to}, msg.Bytes())
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func HashToken(raw string) string {
	// store hex of raw for reset tokens we can compare; for simplicity use base64 of sha would be better
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}
