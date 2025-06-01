package config

import (
	"context"
	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"strings"
)

func LoadConfig(ctx context.Context, filepath string, validate *validator.Validate) (*Config, error) {
	k := koanf.New(".")

	if err := k.Load(env.Provider("RPN_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(strings.TrimPrefix(s, "RPN_")), "_", ".", -1)
	}), nil); err != nil {
		return nil, err
	}

	if err := k.Load(file.Provider(filepath), yaml.Parser()); err != nil {
		return nil, err
	}

	var config Config

	if err := k.Unmarshal("", &config); err != nil {
		return nil, err
	}

	if err := validate.StructCtx(ctx, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
