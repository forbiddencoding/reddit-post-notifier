package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/entity"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/postgres/models"
	"github.com/jackc/pgx/v5"
)

const loadConfigurationAndStateQuery = `
SELECT
    c.id as id,
	c.keyword as keyword,
	sc.id AS subreddit_id,
	sc.subreddit AS subreddit,
	sc.include_nsfw AS include_nsfw,
	sc.sort AS sort,
	sc.restrict_subreddit AS restrict_subreddit,
	sc.fetch_mode AS fetch_mode,
	sc.fetch_limit AS fetch_limit,
	scs.before AS before
FROM
    	configuration c
JOIN
	subreddit_configuration sc ON c.id = sc.configuration_id
LEFT JOIN
	subreddit_configuration_state scs ON sc.id = scs.subreddit_configuration_id
WHERE
    	c.id = @id;
`

func (h *Handle) LoadConfigurationAndState(ctx context.Context, in *entity.LoadConfigurationAndStateInput) (*entity.LoadConfigurationAndStateOutput, error) {
	db := h.db.Load()

	args := pgx.NamedArgs{
		"id": in.ID,
	}

	rows, err := db.Query(ctx, loadConfigurationAndStateQuery, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dbModels, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.LoadConfigurationAndState])
	if err != nil {
		return nil, err
	}

	if len(dbModels) == 0 {
		return &entity.LoadConfigurationAndStateOutput{}, nil
	}

	result := &entity.LoadConfigurationAndStateOutput{
		Keyword: dbModels[0].Keyword,
	}
	subreddits := make([]*entity.Subreddit, 0, len(dbModels))

	for _, m := range dbModels {
		sr := &entity.Subreddit{
			ID:                m.SubredditID,
			Name:              m.Subreddit,
			IncludeNSFW:       m.IncludeNSFW,
			Sort:              m.Sort,
			RestrictSubreddit: m.RestrictSubreddit,
			FetchMode:         m.FetchMode,
		}

		if m.Before.Valid {
			sr.After = m.Before.String
		}

		if m.FetchLimit.Valid {
			sr.FetchLimit = m.FetchLimit.Int64
		}

		subreddits = append(subreddits, sr)
	}

	result.Subreddits = subreddits

	return result, nil
}

const updateStateQuery = `
INSERT INTO subreddit_configuration_state (subreddit_configuration_id, before) VALUES ($1, $2)
ON CONFLICT (subreddit_configuration_id) DO UPDATE 
SET before = EXCLUDED.before, last_updated_at = CURRENT_TIMESTAMP;
`

func (h *Handle) UpdateState(ctx context.Context, in *entity.UpdateStateInput) (*entity.UpdateStateOutput, error) {
	db := h.db.Load()

	batch := &pgx.Batch{}

	for _, v := range in.Values {
		batch.Queue(updateStateQuery, v.SubredditConfigurationID, v.Before)
	}

	br := db.SendBatch(ctx, batch)
	defer func() {
		_ = br.Close()
	}()

	var err error

	for _, v := range in.Values {
		if _, err = br.Exec(); err != nil {
			errors.Join(err, fmt.Errorf("failed to update state for subreddit configuration ID %d: %w", v.SubredditConfigurationID, err))
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to update state: %w", err)
	}

	return nil, nil
}
