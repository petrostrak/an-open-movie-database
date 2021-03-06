package main

import (
	"context"
	"database/sql"
	"expvar"
	"flag"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/petrostrak/an-open-movie-database/internal/data"
	"github.com/petrostrak/an-open-movie-database/internal/jsonlog"
	"github.com/petrostrak/an-open-movie-database/internal/mailer"
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
	// Add a new limiter struct containing fields for the requests-per-second and burst
	// values, and a boolean field which we can ust to enable/disable rate limiting
	// altogether.
	limiter struct {
		rps    float64
		burst  int
		enable bool
	}
	// Update the config struct to hold the SMTP server settings.
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	// Add a cors struct and trustedOrigins field with the type []string.
	cors struct {
		trustedOrigins []string
	}
}

// Define an application struct to hold the dependencies for our HTTP handlers, helpers,
// and middleware.
//
// Change the logger field to have the type *jsonlog.Logger
//
// Update the application struct to hold a new Mailer instance.
//
// Include a sync.WaitGroup in the application struct. The zero-value for a
// sync.WaitGroup type is a valid, usable sync.WaitGroup with a 'counter' value of 0,
// so we don't need to do anything else to initialize it before we can use it.
type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
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

	// Create command line flags to read the setting values into the config struct.
	// We use true as the default for the enabled setting
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enable, "limiter-enable", true, "Enable rate limiter")

	// Read the SMTP server configuration settings into the config struct, using the
	// Mailtrap settings as the default values. IMPORTANT: If you're following along,
	// make sure to replace the default values for smtp-username and smtp-password
	// with your own Mailtrap credentials.
	flag.StringVar(&cfg.smtp.host, "smtp-host", "    smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "a8c6ea4f80cc3f", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "e6231e9d245f54", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Online Movie DB <no-reply@omdb.net", "SMTP sender")

	// Use the flag.Func() to process the -cors-trusted-origins command line
	// flag. In this we use the strings.Fields() to split the flag value into a
	// slice based on whitespace characters and assign it to our config struct.
	// Importantly, if the -cors-trusted-origins flag is not present, contains
	// the empty string, or contains only whitespace, then strings.Fields() will
	// return an empty []string slice.
	flag.Func("cors-trusted-origins", "Trusted CORS origin (space separated)", func(s string) error {
		cfg.cors.trustedOrigins = strings.Fields(s)
		return nil
	})

	flag.Parse()

	// Initialize a new jsonlog.Logger which writes any message -at or above- the INFO
	// severity level to the standart out stream.
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	// Call the openDB() helper function to create the connection pool,
	// passing in the config struct. If this returns an error, we log it and exit the
	// application.
	db, err := openDB(cfg)
	if err != nil {
		// Use the PrintFatal() to write a log entry containing the error at the
		// FATAL level and exit.
		logger.PrintFatal(err, nil)
	}

	// Defer a call to db.Close() so that the connection pool is closed before the
	// main() function exits.
	defer db.Close()

	// Also log a message to say that the connection pool has been successfully
	// established.
	logger.PrintInfo("database connection pool established", nil)

	// Publish a new "version" variable in the expvar handler containing our application
	// version number
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

	// Declare an instance of the application struct, containing the config struct and
	// the logger.
	//
	// Use the data.NewModels() to initialize a Models struct, passing in the
	// connection pool as a parameter.
	//
	// Initialize a new Mailer instance using the settings from the command line
	// flags, and add it to the application struct.
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.username, cfg.smtp.sender),
	}

	// Call app.serve() to start the server.
	err = app.serve()
	if err != nil {
		logger.PrintFatal(err, nil)
	}
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
	// MaxOpenConns, we???ll also limit MaxIdleConns to 25 connections.
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
	// connections thathaven???t been used for a long time. In this project
	// we???ll set a ConnMaxIdleTime duration of 15 minutes.
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
