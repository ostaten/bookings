package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"time"
	"github.com/alexedwards/scs/v2"
	"github.com/ostaten/bookings/internal/config"
	"github.com/ostaten/bookings/internal/handlers"
	"github.com/ostaten/bookings/internal/models"
	"github.com/ostaten/bookings/internal/render"
)

const portNumber = "localhost:8080"
var app config.AppConfig
var session *scs.SessionManager

//Note: to run on Windows, do "go run ./cmd/web/."
func main() {
	err := run()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Starting application on port %s", portNumber)
	srv := &http.Server {
		Addr: portNumber,
		Handler: routes(&app),
	}
	err = srv.ListenAndServe()
	log.Fatal(err)
}

func run() error {
	//what am I going to put in the session
	gob.Register(models.Reservation{})

	//change this to true when in production
	app.InProduction = false

	session = scs.New()
	session.Lifetime = 24 * time.Hour 
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = app.InProduction

	app.Session = session

	tc, err := render.CreateTemplateCache()
	if err != nil {
		log.Fatal("cannot create template cache")
		return err
	}

	app.TemplateCache = tc

	app.UseCache = false
	repo := handlers.NewRepo(&app)
	handlers.NewHandlers(repo)

	render.NewTemplates(&app)
	return nil
}