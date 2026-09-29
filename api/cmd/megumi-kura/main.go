package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/samuelfabel/megumi-kura/api/internal/auth"
	"github.com/samuelfabel/megumi-kura/api/internal/basket"
	"github.com/samuelfabel/megumi-kura/api/internal/category"
	"github.com/samuelfabel/megumi-kura/api/internal/config"
	"github.com/samuelfabel/megumi-kura/api/internal/database"
	"github.com/samuelfabel/megumi-kura/api/internal/delivery"
	"github.com/samuelfabel/megumi-kura/api/internal/food"
	"github.com/samuelfabel/megumi-kura/api/internal/need"
	"github.com/samuelfabel/megumi-kura/api/internal/person"
	"github.com/samuelfabel/megumi-kura/api/internal/promise"
	"github.com/samuelfabel/megumi-kura/api/internal/server"
	"github.com/samuelfabel/megumi-kura/api/internal/settings"
	"github.com/samuelfabel/megumi-kura/api/internal/stock"
	"github.com/samuelfabel/megumi-kura/api/internal/user"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "apply migrations and exit")
	migrationsDir := flag.String("migrations", "", "path to postgres migrations directory")
	flag.Parse()

	cfg := config.Load()
	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	migDir := *migrationsDir
	if migDir == "" {
		migDir = defaultMigrationsDir()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := database.Migrate(ctx, db, migDir); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if *migrateOnly {
		log.Printf("migrations applied from %s", migDir)
		return
	}

	users := &user.Repository{DB: db}
	foods := &food.Repository{DB: db}
	stockRepo := &stock.Repository{DB: db, AllowNegativeStock: cfg.AllowNegativeStock}
	promises := &promise.Repository{DB: db}
	basketRepo := &basket.Repository{DB: db, Stock: stockRepo}
	settingsRepo := &settings.Repository{DB: db, Cfg: cfg}
	categories := &category.Repository{DB: db}
	needs := &need.Repository{DB: db}
	people := &person.Repository{DB: db}
	deliveries := &delivery.Repository{DB: db, AllowNegativeStock: cfg.AllowNegativeStock}
	authSvc := &auth.Service{
		Users:        users,
		Secret:       []byte(cfg.SessionSecret),
		SessionHours: cfg.SessionHours,
		CookieSecure: cfg.CookieSecure,
	}

	api := &server.API{
		Cfg:        cfg,
		Auth:       authSvc,
		Foods:      foods,
		Stock:      stockRepo,
		Promises:   promises,
		Basket:     basketRepo,
		Settings:   settingsRepo,
		Categories: categories,
		Needs:      needs,
		People:     people,
		Deliveries: deliveries,
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("Megumi Kura listening on %s", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}

func defaultMigrationsDir() string {
	candidates := []string{
		"migrations/postgres",
		filepath.Join("api", "migrations", "postgres"),
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates,
			filepath.Join(filepath.Dir(exe), "migrations", "postgres"),
			filepath.Join(filepath.Dir(exe), "..", "migrations", "postgres"),
		)
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return "migrations/postgres"
}
