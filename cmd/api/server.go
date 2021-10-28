package main

import (
	"fmt"
	"net/http"
	"time"
)

// Declare a HTTP server with some sensible timeout settings, which listens on the
// port provided in the config struct and uses the servemux we created above as the
// handler.
func (app *application) serve() error {
	// Declare a HTTP server as in main()
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	// Likewise log a "starting server" message.
	app.logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  app.config.env,
	})
	// Start the server as normal, returning any error.
	return srv.ListenAndServe()
}
