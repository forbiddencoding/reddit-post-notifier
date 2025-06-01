package entity

import "github.com/google/uuid"

type (
	LoadConfigurationAndStateInput struct {
		ID uuid.UUID `json:"id"`
	}

	LoadConfigurationAndStateOutput struct {
		Keyword string `json:"keyword"`
	}

	UpdateStateInput struct {
		ConfigurationID uuid.UUID `json:"configuration_id"`
		Subreddit       string    `json:"subreddit"`
		Before          string    `json:"before"`
	}

	UpdateStateOutput struct {
	}
)
