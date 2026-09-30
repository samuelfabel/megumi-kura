package settings

import (
	"context"
	"database/sql"

	"github.com/samuelfabel/megumi-kura/api/internal/config"
)

type PublicSettings struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Logo           string `json:"logo"`
	Favicon        string `json:"favicon"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	AccentColor    string `json:"accent_color"`
	DefaultLocale  string `json:"default_locale"`
}

type Repository struct {
	DB  *sql.DB
	Cfg config.Config
}

func (r *Repository) Public(ctx context.Context) (PublicSettings, error) {
	// Defaults from config / env, then overlay DB settings.
	out := PublicSettings{
		Name:           r.Cfg.SiteName,
		Description:    r.Cfg.SiteDescription,
		PrimaryColor:   r.Cfg.ThemePrimary,
		SecondaryColor: r.Cfg.ThemeSecondary,
		AccentColor:    r.Cfg.ThemeAccent,
		DefaultLocale:  r.Cfg.DefaultLocale,
		Logo:           "",
		Favicon:        "/favicon.svg",
	}

	rows, err := r.DB.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return out, err
	}
	defer rows.Close()

	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return out, err
		}
		switch k {
		case "site.name":
			out.Name = v
		case "site.description":
			out.Description = v
		case "site.logo":
			out.Logo = v
		case "site.favicon":
			out.Favicon = v
		case "locale.default":
			out.DefaultLocale = v
		case "theme.primary":
			out.PrimaryColor = v
		case "theme.secondary":
			out.SecondaryColor = v
		case "theme.accent":
			out.AccentColor = v
		}
	}
	return out, rows.Err()
}

// NeedTarget returns optional need target for a food (settings key need.<code>).
func (r *Repository) NeedTargets(ctx context.Context) (map[string]string, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT key, value FROM settings WHERE key LIKE 'need.%'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}
