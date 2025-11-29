package main

import (
	"context"
	"database/sql"
	"flag"
	"os"
	"time"

	//pq driver for PostgresSQL
	"github.com/codercollo/property/backend/internal/data"
	"github.com/codercollo/property/backend/internal/jsonlog"
	_ "github.com/lib/pq"
)

// Application version
const version = "1.0.0"

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
}

// Application dependencies
type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
}

func main() {

	var cfg config

	//Load configuration from command-line flags
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("PROPERTY_DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")
	flag.Parse()

	//Init JSON logger at INFO level
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	//Open database connection pool
	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil) //Log fatal error and exit
	}
	defer db.Close()
	logger.PrintInfo("database connection pool established", nil)

	//Initialize the application with config and logger
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
	}

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
