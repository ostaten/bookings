package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/ostaten/bookings/internal/config"
	"github.com/ostaten/bookings/internal/driver"
	"github.com/ostaten/bookings/internal/handlers"
	"github.com/ostaten/bookings/internal/helpers"
	"github.com/ostaten/bookings/internal/models"
	"github.com/ostaten/bookings/internal/render"
)

const portNumber = "localhost:8080"
var app config.AppConfig
var session *scs.SessionManager
var infoLog *log.Logger
var errorLog *log.Logger

//Note: to run on Windows, do "go run ./cmd/web/."
// or "go run ./cmd/web/main.go ./cmd/web/routes.go ./cmd/web/middleware.go"
// OR "./run.bat"
// Autoformat: "alt + shift + f"
//run all tests: "go test -v ./..."
// test coverage: "go test -coverprofile=coverage; go tool cover -html=coverage "
func main() {
	db, err := run()
	
	if err != nil {
		log.Fatal(err)
	}
	defer db.SQL.Close()

	defer close(app.MailChan)

	fmt.Println("Starting mail listener . . .")
	listenForMail()

	fmt.Printf("Starting application on port %s", portNumber)
	srv := &http.Server {
		Addr: portNumber,
		Handler: routes(&app),
	}
	err = srv.ListenAndServe()
	log.Fatal(err)
}

func run() (*driver.DB, error) {
	//what am I going to put in the session
	gob.Register(models.Reservation{})
	gob.Register(models.User{})
	gob.Register(models.Room{})
	gob.Register(models.Restriction{})

	mailChan := make(chan models.MailData)
	app.MailChan = mailChan

	//change this to true when in production
	app.InProduction = false

	infoLog = log.New(os.Stdout, "INFO\t", log.Ldate)
	app.InfoLog = infoLog

	errorLog = log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	app.ErrorLog = errorLog

	session = scs.New()
	session.Lifetime = 24 * time.Hour 
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = app.InProduction

	app.Session = session
	
	//connect to database
	log.Println("Connnect to database...")
	db, err := driver.ConnectSQL("host=localhost port=5432 dbname=bookings user=postgres password=$49NzQerqgsxMB")
	if err != nil {
		log.Fatal("Cannot connect to database! Dying ...")
	}
	log.Println("Connected to database!")

	tc, err := render.CreateTemplateCache()
	if err != nil {
		log.Fatal("cannot create template cache")
		return nil, err
	}

	app.TemplateCache = tc

	app.UseCache = false
	repo := handlers.NewRepo(&app, db)
	handlers.NewHandlers(repo)
	render.NewRenderer(&app)
	helpers.NewHelpers(&app)

	return db, nil
}