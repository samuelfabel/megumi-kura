package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr            string
	DatabaseURL         string
	SessionSecret       string
	StaticDir           string
	SiteName            string
	SiteDescription     string
	DefaultLocale       string
	ThemePrimary        string
	ThemeSecondary      string
	ThemeAccent         string
	AllowNegativeStock  bool
	CookieSecure        bool
	SessionHours        int
}

func Load() Config {
	return Config{
		HTTPAddr:           getenv("MK_HTTP_ADDR", ":8080"),
		DatabaseURL:        getenv("MK_DATABASE_URL", "postgres://megumi:megumi@localhost:5432/megumi_kura?sslmode=disable"),
		SessionSecret:      getenv("MK_SESSION_SECRET", "change-me-dev-only-secret"),
		StaticDir:          getenv("MK_STATIC_DIR", "web/dist"),
		SiteName:           getenv("MK_SITE_NAME", "Megumi Kura"),
		SiteDescription:    getenv("MK_SITE_DESCRIPTION", "Community Food Management"),
		DefaultLocale:      getenv("MK_DEFAULT_LOCALE", "pt-BR"),
		ThemePrimary:       getenv("MK_THEME_PRIMARY", "#2E7D32"),
		ThemeSecondary:     getenv("MK_THEME_SECONDARY", "#795548"),
		ThemeAccent:        getenv("MK_THEME_ACCENT", "#F9A825"),
		AllowNegativeStock: getenvBool("MK_ALLOW_NEGATIVE_STOCK", false),
		CookieSecure:       getenvBool("MK_COOKIE_SECURE", false),
		SessionHours:       getenvInt("MK_SESSION_HOURS", 24),
	}
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getenvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
