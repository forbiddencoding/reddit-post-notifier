package config

type (
	Config struct {
		Temporal    Temporal    `koanf:"temporal" yaml:"temporal" validate:"required"`
		Reddit      Reddit      `koanf:"reddit" yaml:"reddit" validate:"required"`
		Persistence Persistence `koanf:"persistence" yaml:"persistence" validate:"required"`
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
)
