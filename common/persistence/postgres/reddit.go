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
	scs.before AS before,
	r.id AS recipient_id,
	r.type AS type,
	r.value AS value
FROM
    configuration c
JOIN
	subreddit_configuration sc ON c.id = sc.configuration_id
LEFT JOIN
	subreddit_configuration_state scs ON sc.id = scs.subreddit_configuration_id
LEFT JOIN
    recipients r ON c.id = r.configuration_id
WHERE
    	c.id = @id;
`

func (h *Handle) LoadConfigurationAndState(ctx context.Context, in *entity.LoadConfigurationAndStateInput) (*entity.LoadConfigurationAndStateOutput, error) {
	db, err := h.db()
	if err != nil {
		return nil, err
	}

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

	var (
		subreddits   []*entity.Subreddit
		recipientSet = make(map[int64]struct{})
		recipients   []*entity.Recipient
	)

	for _, m := range dbModels {
		sr := &entity.Subreddit{
			ID:                m.SubredditID,
			Name:              m.Subreddit,
			IncludeNSFW:       m.IncludeNSFW,
			Sort:              m.Sort,
			RestrictSubreddit: m.RestrictSubreddit,
		}

		if m.Before.Valid {
			sr.After = m.Before.String
		}

		subreddits = append(subreddits, sr)

		if m.RecipientID.Valid {
			if _, ok := recipientSet[m.RecipientID.Int64]; !ok {
				recipients = append(recipients, &entity.Recipient{
					ID:    m.RecipientID.Int64,
					Type:  m.Type,
					Value: m.Value,
				})
				recipientSet[m.RecipientID.Int64] = struct{}{}
			}
		}
	}

	return &entity.LoadConfigurationAndStateOutput{
		Keyword:    dbModels[0].Keyword,
		Recipients: recipients,
		Subreddits: subreddits,
	}, nil
}

const updateStateQuery = `
INSERT INTO subreddit_configuration_state (subreddit_configuration_id, before) VALUES ($1, $2)
ON CONFLICT (subreddit_configuration_id) DO UPDATE 
SET before = EXCLUDED.before, last_updated_at = CURRENT_TIMESTAMP;
`

func (h *Handle) UpdateState(ctx context.Context, in *entity.UpdateStateInput) (*entity.UpdateStateOutput, error) {
	db, err := h.db()
	if err != nil {
		return nil, err
	}

	batch := &pgx.Batch{}

	for _, v := range in.Values {
		batch.Queue(updateStateQuery, v.SubredditConfigurationID, v.Before)
	}

	br := db.SendBatch(ctx, batch)
	defer func() {
		_ = br.Close()
	}()

	var errs error

	for _, v := range in.Values {
		if _, err = br.Exec(); err != nil {
			errors.Join(errs, fmt.Errorf("failed to update state for subreddit configuration ID %d: %w", v.SubredditConfigurationID, err))
		}
	}

	if errs != nil {
		return nil, fmt.Errorf("failed to update state: %w", errs)
	}

	return nil, nil
}
