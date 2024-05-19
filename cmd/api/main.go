package main

import (
	"context"
	"database/sql"
	"flag"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/yousifsabah0/blackbox/internal/data"
	"github.com/yousifsabah0/blackbox/internal/logx"
	"github.com/yousifsabah0/blackbox/internal/mailer"
)

const (
	version = "v1.0.0"
)

type config struct {
	port int
	env  string
	db   struct {
		dsn string
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
}

type application struct {
	config config
	logger *logx.Logger
	models data.Model
	mailer mailer.Mailer
	wg     sync.WaitGroup
}

func main() {
	var cfg config

	flag.StringVar(&cfg.env, "env", "development", "")
	flag.IntVar(&cfg.port, "port", 8080, "")

	flag.StringVar(&cfg.db.dsn, "dns", "postgres://postgres:pa55word@bb-postgres:5432/blackbox?sslmode=disable", "")

	flag.Float64Var(&cfg.limiter.rps, "rps", 2, "")
	flag.IntVar(&cfg.limiter.burst, "burst", 4, "")
	flag.BoolVar(&cfg.limiter.enabled, "enabled", true, "")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 2525, "")

	flag.StringVar(&cfg.smtp.username, "smtp-username", "0b0bab74d623de", "")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "fa70ef1f1a2a96", "")

	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "no-replay@blackbox.com", "")

	flag.Parse()

	logger := logx.NewLogger(os.Stdout, logx.LevelInfo)

	db, err := openDB(cfg.db.dsn)
	if err != nil {
		logger.Fatal(err, nil)
	}
	defer db.Close()
	logger.Info("database connected successfully", nil)

	mailer := mailer.New(cfg.smtp.host, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender, cfg.smtp.port)

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModel(db),
		mailer: mailer,
	}

	if err := app.serve(); err != nil {
		logger.Fatal(err, nil)
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}
