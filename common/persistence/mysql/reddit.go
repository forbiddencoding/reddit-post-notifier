package mysql

import (
	"context"
	"errors"
	"fmt"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/entity"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/mysql/models"
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
    	c.id = :id;
`

func (h *Handle) LoadConfigurationAndState(ctx context.Context, in *entity.LoadConfigurationAndStateInput) (*entity.LoadConfigurationAndStateOutput, error) {
	db, err := h.db()
	if err != nil {
		return nil, err
	}

	args := map[string]any{
		"id": in.ID,
	}

	rows, err := db.NamedQueryContext(ctx, loadConfigurationAndStateQuery, args)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var (
		keyword      string
		subreddits   []*entity.Subreddit
		recipientSet = make(map[int64]struct{})
		recipients   []*entity.Recipient
	)

	for rows.Next() {
		var m models.LoadConfigurationAndState
		if err = rows.StructScan(&m); err != nil {
			return nil, err
		}

		if keyword == "" {
			keyword = m.Keyword
		}

		subreddit := &entity.Subreddit{
			ID:                m.SubredditID,
			Name:              m.Subreddit,
			IncludeNSFW:       m.IncludeNSFW,
			Sort:              m.Sort,
			RestrictSubreddit: m.RestrictSubreddit,
		}

		if m.Before.Valid {
			subreddit.After = m.Before.String
		}

		subreddits = append(subreddits, subreddit)

		if m.RecipientID.Valid {
			if _, ok := recipientSet[m.RecipientID.Int64]; ok {
				recipients = append(recipients, &entity.Recipient{
					ID:    m.RecipientID.Int64,
					Type:  m.Type,
					Value: m.Value,
				})
				recipientSet[m.RecipientID.Int64] = struct{}{}
			}
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &entity.LoadConfigurationAndStateOutput{
		Keyword:    keyword,
		Recipients: recipients,
		Subreddits: subreddits,
	}, nil

}

const updateStateQuery = `
INSERT INTO subreddit_configuration_state (subreddit_configuration_id, before) VALUES (?, ?)
ON CONFLICT (subreddit_configuration_id) DO UPDATE 
SET before = EXCLUDED.before, last_updated_at = CURRENT_TIMESTAMP;
`

func (h *Handle) UpdateState(ctx context.Context, in *entity.UpdateStateInput) (*entity.UpdateStateOutput, error) {
	db, err := h.db()
	if err != nil {
		return nil, err
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.PreparexContext(ctx, updateStateQuery)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = stmt.Close()
	}()

	var errs error

	for _, v := range in.Values {
		if _, err = stmt.ExecContext(ctx, v.SubredditConfigurationID, v.Before); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to update state for subreddit configuration ID %d: %w", v.SubredditConfigurationID, err))
		}
	}

	if errs != nil {
		return nil, errs
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil, nil
}
