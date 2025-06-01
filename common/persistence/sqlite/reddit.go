package sqlite

import (
	"context"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/entity"
)

func (h *Handle) LoadConfigurationAndState(ctx context.Context, in *entity.LoadConfigurationAndStateInput) (*entity.LoadConfigurationAndStateOutput, error) {
	return nil, nil
}

func (h *Handle) UpdateState(ctx context.Context, in *entity.UpdateStateInput) (*entity.UpdateStateOutput, error) {
	return nil, nil
}
