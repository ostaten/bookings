package config

import (
	"html/template"
	"log"

	"github.com/alexedwards/scs/v2"
)

//config might be accessed from any part of application
//should not import anything; only should be imported

// AppConfig holds the application config
type AppConfig struct {
	UseCache      bool
	TemplateCache map[string]*template.Template
	InfoLog       *log.Logger
	ErrorLog      *log.Logger
	InProduction  bool
	Session       *scs.SessionManager
}
