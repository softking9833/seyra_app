package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"seyra/backend/internal/auth"
	"seyra/backend/internal/chat"
	"seyra/backend/internal/config"
	"seyra/backend/internal/database"
	httpapi "seyra/backend/internal/http"
	"seyra/backend/internal/media"
	"seyra/backend/internal/notify"
	"seyra/backend/migrations"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	_ = config.LoadDotEnv(".env")
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool, migrations.Files); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	store := auth.NewPostgresStore(pool)
	authService := auth.NewService(store, cfg.SessionPepper)
	hub := chat.NewHub()
	chatService := chat.NewService(chat.NewPostgresStore(pool), chat.NewAuthDirectory(store), hub)
	blobs, err := media.NewLocalStore(cfg.MediaDir)
	if err != nil {
		log.Fatalf("media: %v", err)
	}
	chatService.SetBlobStore(blobs)
	var pushSender notify.Sender = notify.NoopSender{}
	if cfg.PushWebhookURL != "" {
		pushSender = &notify.WebhookSender{URL: cfg.PushWebhookURL, Secret: cfg.PushWebhookAuth}
	}
	alerts := notify.NewService(notify.NewPostgresStore(pool), pushSender)
	handler := httpapi.NewServer(authService, chatService, hub, alerts)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		env := strings.ToLower(strings.TrimSpace(os.Getenv("SEYRA_ENV")))
		cert := os.Getenv("SEYRA_TLS_CERT")
		key := os.Getenv("SEYRA_TLS_KEY")
		if env == "production" {
			if cert == "" || key == "" {
				log.Fatal("SEYRA_TLS_CERT and SEYRA_TLS_KEY are required when SEYRA_ENV=production")
			}
			log.Printf("seyra api listening on %s (tls)", cfg.HTTPAddr)
			if err := server.ListenAndServeTLS(cert, key); err != nil && err != http.ErrServerClosed {
				log.Fatalf("http: %v", err)
			}
			return
		}
		log.Printf("seyra api listening on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
