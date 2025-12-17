package main

import (
	"context"
	"database/sql"
	"expvar"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	//pq driver for PostgresSQL
	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/jsonlog"
	"github.com/codercollo/property/backend/internal/mailer"
	_ "github.com/lib/pq"
)

// Application version
var (
	buildTime string
	version   string
)

// Configuration settings for the server
type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
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
	cors struct {
		trustedOrigins []string
	}
	jwt struct {
		secret string
	}
	mpesa struct {
		consumerKey    string
		consumerSecret string
		passkey        string
		shortCode      string
		environment    string
	}
	baseURL string
}

// Application dependencies
type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
}

func main() {

	var cfg config

	//Load configuration from command-line flags
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.StringVar(&cfg.db.dsn, "db-dsn", "", "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")
	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 2525, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "7c529b35aca45a", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "e6cd237eff9652", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <itscollinsmaina@gmail.com>", "SMTP sender")
	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})
	flag.StringVar(&cfg.jwt.secret, "jwt-secret", "", "JWT secret")
	flag.StringVar(&cfg.mpesa.consumerKey, "mpesa-consumer-key", "", "M-Pesa consumer key")
	flag.StringVar(&cfg.mpesa.consumerSecret, "mpesa-consumer-secret", "", "M-Pesa consumer secret")
	flag.StringVar(&cfg.mpesa.passkey, "mpesa-passkey", "", "M-Pesa passkey")
	flag.StringVar(&cfg.mpesa.shortCode, "mpesa-shortcode", "", "M-Pesa business short code")
	flag.StringVar(&cfg.mpesa.environment, "mpesa-env", "sandbox", "M-Pesa environment (sandbox|production)")
	flag.StringVar(&cfg.baseURL, "base-url", "http://localhost:4000", "Base URL for callbacks")

	// Create a new version boolean flag with the default value of false.
	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		fmt.Printf("Build time:\t%s\n", buildTime)
		os.Exit(0)
	}

	//Init JSON logger at INFO level
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	//Open database connection pool
	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil) //Log fatal error and exit
	}
	defer db.Close()
	logger.PrintInfo("database connection pool established", nil)

	//Publish version
	expvar.NewString("version").Set(version)

	// Publish the number of active goroutines.
	expvar.Publish("goroutines", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))
	// Publish the database connection pool statistics.
	expvar.Publish("database", expvar.Func(func() interface{} {
		return db.Stats()
	}))
	// Publish the current Unix timestamp.
	expvar.Publish("timestamp", expvar.Func(func() interface{} {
		return time.Now().Unix()
	}))

	//Initialize the application with config and logger
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer.New(
			cfg.smtp.host,
			cfg.smtp.port,
			cfg.smtp.username,
			cfg.smtp.password,
			cfg.smtp.sender,
		),
	}

	//Start background jobs
	app.startBackgroundJobs()

	//Start the server
	err = app.serve()
	if err != nil {
		logger.PrintFatal(err, nil) //Log fatal error and exit
	}

}

// openDB initializes and pings a PostgreSQL connection pool
func openDB(cfg config) (*sql.DB, error) {
	//Create a new databse connection pool
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	//Configure database connection pool linits and idle tiemout
	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	//Set maximum idle time for connections
	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(duration)

	//Verify connection to the database with a 5-sec timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}
