package config

type (
	Config struct {
		Temporal    Temporal    `koanf:"temporal" validate:"required"`
		Reddit      Reddit      `koanf:"reddit" validate:"required"`
		Persistence Persistence `koanf:"db" validate:"required"`
		Mailer      Mailer      `koanf:"mailer" validate:"required"`
		Server      Server      `koanf:"server" validate:"required"`
	}

	Temporal struct {
		HostPort  string `koanf:"hostport" validate:"required"`
		Namespace string `koanf:"namespace" validate:"required"`
	}

	Reddit struct {
		ClientID     string `koanf:"clientid" validate:"required"`
		ClientSecret string `koanf:"clientsecret" validate:"required"`
		RedirectURI  string `koanf:"redirecturi" validate:"required"`
		UserAgent    string `koanf:"useragent" validate:"required"`
	}

	Persistence struct {
		User     string `koanf:"user" validate:"required"`
		Password string `koanf:"pwd" validate:"required"`
		Host     string `koanf:"host" validate:"required"`
		Port     int    `koanf:"port" validate:"required"`
		Database string `koanf:"dbname" validate:"required"`
	}

	Mailer struct {
		Provider      string `koanf:"provider" validate:"required,oneof=gmail resend ses mailersend scaleway plunk"`
		SenderEmail   string `koanf:"sender" validate:"required,email"`
		SenderName    string `koanf:"sendername" validate:"required"`
		SubjectPrefix string `koanf:"subjectprefix" validate:"required"`
		AppPassword   string `koanf:"gmailapppassword" validate:"required_if=Provider gmail"`
	}

	Server struct {
		Host string `koanf:"host" validate:"required"`
		Port int    `koanf:"port" validate:"required"`
	}
)
