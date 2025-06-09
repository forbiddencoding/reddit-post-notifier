package config

type (
	Config struct {
		Temporal    Temporal    `koanf:"temporal" yaml:"temporal" validate:"required"`
		Reddit      Reddit      `koanf:"reddit" yaml:"reddit" validate:"required"`
		Persistence Persistence `koanf:"persistence" yaml:"persistence" validate:"required"`
		Mailer      Mailer      `koanf:"mailer" yaml:"mailer" validate:"required"`
	}

	Temporal struct {
		HostPort  string `koanf:"hostPort" validate:"required"`
		Namespace string `koanf:"namespace" validate:"required"`
	}

	Reddit struct {
		ClientID     string `koanf:"clientId" validate:"required"`
		ClientSecret string `koanf:"clientSecret" validate:"required"`
		RedirectURI  string `koanf:"redirectUri" validate:"required"`
		UserAgent    string `koanf:"userAgent" validate:"required"`
	}

	Persistence struct {
		Driver string `koanf:"driver" validate:"required,oneof=postgres sqlite mysql"`
		DSN    string `koanf:"dsn" validate:"required"`
	}

	Mailer struct {
		Provider      string `koanf:"provider" validate:"required,oneof=gmail resend ses mailersend scaleway plunk"`
		SenderEmail   string `koanf:"senderEmail" validate:"required,email"`
		SenderName    string `koanf:"senderName" validate:"required"`
		SubjectPrefix string `koanf:"subjectPrefix" validate:"required"`
		GMail         struct {
			AppPassword string `koanf:"appPassword" validate:"required_if=Provider gmail"`
		} `koanf:"gmail" validate:"required_if=Provider gmail"`
		Resend struct {
			APIKey string `koanf:"apiKey" validate:"required_if=Provider resend"`
		} `koanf:"resend" validate:"required_if=Provider resend"`
	}
)
