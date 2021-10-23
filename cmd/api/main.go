package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/petrostrak/an-open-movie-database/internal/data"
)

const (
	version = "1.0.0"
)

// Define a config struct to hold all the configuration settings for our application.
//
// Add a db struct field to hold the configuration setting for our database connection
// pool. For now this only holds the DSN, which we will read in from a command-line flag.
type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
}

// Define an application struct to hold the dependencies for our HTTP handlers, helpers,
// and middleware.
type application struct {
	config config
	logger *log.Logger
	models data.Models
}

// go run ./cmd/api -port=3030 -env=production
func main() {
	// Declare an instance of the config struct.
	var cfg config

	// Read the value of the port and env command-line flags into the config struct. We
	// default to using the port number 4000 and the environment "development" if no
	// corresponding flags are provided.
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment(development|staging|production)")

	// Read the DSN value from the db-dsn command-line flag into the config struct.
	// We default to using our development DSN if no flag is provided.
	//
	// Use the value of the OMDB_DB_DSN environment variable as the default value
	// for the db-dsn command-line flag.
	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("OMDB_DB_DSN"), "PostgreSQL DSN")

	// Read the connection pool settings from command-line flags into the config struct
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL mac connection idle time")

	flag.Parse()

	// Initialize a new logger which writes messages to the standard out stream,
	// prefixed with the current date and time.
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	// Call the openDB() helper function to create the connection pool,
	// passing in the config struct. If this returns an error, we log it and exit the
	// application.
	db, err := openDB(cfg)
	if err != nil {
		logger.Fatal(err)
	}

	// Defer a call to db.Close() so that the connection pool is closed before the
	// main() function exits.
	defer db.Close()

	// Also log a message to say that the connection pool has been successfully
	// established.
	logger.Printf("database connection pool established\n")

	// Declare an instance of the application struct, containing the config struct and
	// the logger.
	//
	// Use the data.NewModels() to initialize a Models struct, passing in the
	// connection pool as a parameter.
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
	}

	// Declare a new servemux and add a /v1/healthcheck route which dispatches requests
	// to the healthcheckHandler method.
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/healthcheck", app.healthcheckHandler)

	// Declare a HTTP server with some sensible timeout settings, which listens on the
	// port provided in the config struct and uses the servemux we created above as the
	// handler.
	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.port),
		// Use the httprouter instance returned by app.routes() as the server handler.
		Handler:     app.routes(),
		IdleTimeout: time.Minute,
		ReadTimeout: 10 * time.Second, WriteTimeout: 30 * time.Second,
	}

	// Start the HTTP server.
	logger.Printf("starting %s server on %s", cfg.env, srv.Addr)
	err = srv.ListenAndServe()
	logger.Fatal(err)
}

// The openDB() function returns a sql.DB connection pool.
func openDB(cfg config) (*sql.DB, error) {
	// Use sql.Open() to create an empty connection pool, using the DSN from the config
	// struct.
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	// Set the maximum number of open (in-use + idle) connections in the pool.
	// Note that passing a value less than or equal to 0 will mean there is no
	// limit.
	//
	// As a rule of thumb, you should explicitly set a MaxOpenConns value.
	// Set a MaxOpenConns limit of 25 connections as it happens to be a
	// reasonable starting point for small-to- medium web applications and APIs.
	db.SetMaxOpenConns(cfg.db.maxOpenConns)

	// Set the maximum number of idle connections in the pool. Again, passing a value
	// less that or equal to 0 will mean there is no limit.
	//
	// In general, higher MaxOpenConns and MaxIdleConns values will lead to better
	// performance. Because MaxIdleConns should always be less than or equal to
	// MaxOpenConns, we’ll also limit MaxIdleConns to 25 connections.
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	// Use the time.ParseDuration() to convert the idle timeout duration string
	// to a time.Duration type.
	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}

	// Set the maximum idle timeout.
	//
	// You should set generally a ConnMaxIdleTime value to remove idle
	// connections thathaven’t been used for a long time. In this project
	// we’ll set a ConnMaxIdleTime duration of 15 minutes.
	db.SetConnMaxIdleTime(duration)

	// Create a context with a 5-second timeout deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use PingContext() to establish a new connection to the database, passing in the
	// context we created above as a parameter. If the connection couldn't be
	// established successfully within the 5 second deadline, then this will return an// error.
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	// Return the sql.DB connection pool
	return db, nil
}
