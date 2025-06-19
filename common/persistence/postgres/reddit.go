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
    c.id AS id,
	c.keyword AS keyword,
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
SET before = EXCLUDED.before, lASt_updated_at = CURRENT_TIMESTAMP;
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

const (
	createScheduleConfigurationQuery = `
INSERT INTO 
    configuration (id, keyword, schedule) 
VALUES (@id, @keyword, @schedule)
`
	createScheduleSubredditConfigurationQuery = `
INSERT INTO 
    subreddit_configuration (id, configuration_id, subreddit, include_nsfw, sort, restrict_subreddit) 
VALUES (@id, @configuration_id, @subreddit, @include_nsfw, @sort, @restrict_subreddit) 
`
	createScheduleRecipientsQuery = `
INSERT INTO 
    recipients (id, configuration_id, type, value)
VALUES (@id, @configuration_id, @type, @value)`
)

func (h *Handle) CreateSchedule(ctx context.Context, in *entity.CreateScheduleInput) (*entity.CreateScheduleOutput, error) {
	db, err := h.db()
	if err != nil {
		return nil, err
	}

	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err = tx.Exec(ctx, createScheduleConfigurationQuery, pgx.NamedArgs{
		"id":       in.ID,
		"keyword":  in.Keyword,
		"schedule": in.Schedule,
	}); err != nil {
		return nil, err
	}

	for _, subreddit := range in.Subreddits {
		args := pgx.NamedArgs{
			"id":                 subreddit.ID,
			"configuration_id":   in.ID,
			"subreddit":          subreddit.Subreddit,
			"include_nsfw":       subreddit.IncludeNSFW,
			"sort":               subreddit.Sort,
			"restrict_subreddit": subreddit.RestrictSubreddit,
		}
		if _, err = tx.Exec(ctx, createScheduleSubredditConfigurationQuery, args); err != nil {
			return nil, err
		}
	}

	for _, recipient := range in.Recipients {
		args := pgx.NamedArgs{
			"id":               recipient.ID,
			"configuration_id": in.ID,
			"type":             recipient.Type,
			"value":            recipient.Value,
		}
		if _, err = tx.Exec(ctx, createScheduleRecipientsQuery, args); err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &entity.CreateScheduleOutput{}, nil
}

const getScheduleQuery = `
SELECT
    c.id AS id,
	c.keyword AS keyword,
	c.schedule AS schedule,
	sc.id AS subreddit_id,
	sc.subreddit AS subreddit,
	sc.include_nsfw AS include_nsfw,
	sc.sort AS sort,
	sc.restrict_subreddit AS restrict_subreddit,
	r.id AS recipient_id,
	r.type AS type,
	r.value AS value
FROM
    configuration c
JOIN
	subreddit_configuration sc ON c.id = sc.configuration_id
LEFT JOIN
    recipients r ON c.id = r.configuration_id
WHERE
    	c.id = @id;
`

func (h *Handle) GetSchedule(ctx context.Context, in *entity.GetScheduleInput) (*entity.GetScheduleOutput, error) {
	db, err := h.db()
	if err != nil {
		return nil, err
	}

	args := pgx.NamedArgs{
		"id": in.ID,
	}

	rows, err := db.Query(ctx, getScheduleQuery, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dbModels, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.GetSchedule])
	if err != nil {
		return nil, err
	}

	if len(dbModels) == 0 {
		return &entity.GetScheduleOutput{}, nil
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

	return &entity.GetScheduleOutput{
		Keyword:    dbModels[0].Keyword,
		Schedule:   dbModels[0].Schedule,
		Recipients: recipients,
		Subreddits: subreddits,
		OwnerID:    0,
	}, nil
}

const (
	deleteScheduleQuery = `
DELETE FROM configuration WHERE id = @id;
`
)

func (h *Handle) DeleteSchedule(ctx context.Context, in *entity.DeleteScheduleInput) (*entity.DeleteScheduleOutput, error) {
	db, err := h.db()
	if err != nil {
		return nil, err
	}

	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	args := pgx.NamedArgs{
		"id": in.ID,
	}

	cmd, err := db.Exec(ctx, deleteScheduleQuery, args)
	if err != nil {
		return nil, err
	}

	if cmd.RowsAffected() == 0 {
		return nil, errors.New("delete schedule row not found")
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return nil, nil
}

const listSchedulesQuery = `
SELECT
	c.id AS id,
	c.keyword AS keyword,
	c.schedule AS schedule,
	c.owner_id AS owner_id,
	c.status AS status,
	sc.id AS subreddit_id,
	sc.subreddit AS subreddit,
	sc.include_nsfw AS include_nsfw,
	sc.sort AS sort,
	sc.restrict_subreddit AS restrict_subreddit,
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
`

func (h *Handle) ListSchedules(ctx context.Context, in *entity.ListSchedulesInput) (*entity.ListSchedulesOutput, error) {
	db, err := h.db()
	if err != nil {
		return nil, err
	}

	query := listSchedulesQuery
	args := pgx.NamedArgs{}

	if in.OwnerID != 0 {
		query += "WHERE c.owner_id = @owner_id"
		args["owner_id"] = in.OwnerID
	}

	query += "ORDER BY c.id;"

	rows, err := db.Query(ctx, listSchedulesQuery, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dbModels, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.ListSchedulesModel])
	if err != nil {
		return nil, err
	}

	if len(dbModels) == 0 {
		return &entity.ListSchedulesOutput{}, nil
	}

	schedulesMap := make(map[int64]*entity.Schedule)

	subredditSets := make(map[int64]map[int64]struct{})
	recipientSets := make(map[int64]map[int64]struct{})

	for _, m := range dbModels {
		if _, ok := schedulesMap[m.ID]; !ok {
			schedulesMap[m.ID] = &entity.Schedule{
				ID:         m.ID,
				Keyword:    m.Keyword,
				Schedule:   m.Schedule,
				Status:     m.Status,
				Recipients: []*entity.Recipient{},
				Subreddits: []*entity.Subreddit{},
			}
			subredditSets[m.ID] = make(map[int64]struct{})
			recipientSets[m.ID] = make(map[int64]struct{})
		}

		schedule := schedulesMap[m.ID]

		if _, ok := subredditSets[m.ID][m.SubredditID]; !ok {
			schedule.Subreddits = append(schedule.Subreddits, &entity.Subreddit{
				ID:                m.SubredditID,
				Name:              m.Subreddit,
				IncludeNSFW:       m.IncludeNSFW,
				Sort:              m.Sort,
				RestrictSubreddit: m.RestrictSubreddit,
			})
			subredditSets[m.ID][m.SubredditID] = struct{}{}
		}

		if m.RecipientID.Valid {
			if _, ok := recipientSets[m.ID][m.RecipientID.Int64]; !ok {
				schedule.Recipients = append(schedule.Recipients, &entity.Recipient{
					ID:    m.RecipientID.Int64,
					Type:  m.Type,
					Value: m.Value,
				})
				recipientSets[m.ID][m.RecipientID.Int64] = struct{}{}
			}
		}
	}

	schedules := make([]*entity.Schedule, 0, len(schedulesMap))
	for _, schedule := range schedulesMap {
		schedules = append(schedules, schedule)
	}

	return &entity.ListSchedulesOutput{
		Schedules: schedules,
	}, nil
}
