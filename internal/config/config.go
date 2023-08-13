package config

import (
	"html/template"

	"github.com/alexedwards/scs/v2"
)

//config might be accessed from any part of application
//should not import anything; only should be imported

// AppConfig holds the application config
type AppConfig struct {
	UseCache      bool
	TemplateCache map[string]*template.Template
	InProduction bool
	Session *scs.SessionManager
}
