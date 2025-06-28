package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/entity"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/postgres/models"
	"github.com/jackc/pgx/v5"
	"strings"
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
	scs.last_post AS before,
	r.id AS recipient_id,
	r.address AS address
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

	dbModels, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.LoadConfigurationAndState])
	if err != nil {
		return nil, err
	}

	if len(dbModels) == 0 {
		return &entity.LoadConfigurationAndStateOutput{}, nil
	}

	keyword := dbModels[0].Keyword
	subredditMap := make(map[int64]*entity.Subreddit)
	recipientMap := make(map[int64]*entity.Recipient)

	for _, m := range dbModels {
		if _, ok := subredditMap[m.SubredditID]; !ok {
			subreddit := &entity.Subreddit{
				ID:                m.SubredditID,
				Name:              m.Subreddit,
				IncludeNSFW:       m.IncludeNSFW,
				Sort:              m.Sort,
				RestrictSubreddit: m.RestrictSubreddit,
			}

			if m.Before.Valid {
				subreddit.Before = m.Before.String
			}

			subredditMap[m.SubredditID] = subreddit
		}

		if m.RecipientID.Valid {
			if _, ok := recipientMap[m.RecipientID.Int64]; !ok {
				recipientMap[m.RecipientID.Int64] = &entity.Recipient{
					ID:      m.RecipientID.Int64,
					Address: m.Address,
				}
			}
		}
	}

	subreddits := make([]*entity.Subreddit, 0, len(subredditMap))
	for _, v := range subredditMap {
		subreddits = append(subreddits, v)
	}

	recipients := make([]*entity.Recipient, 0, len(recipientMap))
	for _, v := range recipientMap {
		recipients = append(recipients, v)
	}

	return &entity.LoadConfigurationAndStateOutput{
		Keyword:    keyword,
		Recipients: recipients,
		Subreddits: subreddits,
	}, nil
}

const updateStateQuery = `
INSERT INTO subreddit_configuration_state (subreddit_configuration_id, last_post) VALUES ($1, $2)
ON CONFLICT (subreddit_configuration_id) DO UPDATE 
SET last_post = EXCLUDED.last_post, last_updated_at = CURRENT_TIMESTAMP;
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

	for i := 0; i < batch.Len(); i++ {
		if _, err = br.Exec(); err != nil {
			errors.Join(errs, err)
		}
	}

	if errs != nil {
		return nil, fmt.Errorf("failed to update state: %w", errs)
	}

	return &entity.UpdateStateOutput{}, nil
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
    recipients (id, configuration_id, address)
VALUES (@id, @configuration_id, @address)`
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
			"address":          recipient.Address,
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
	r.address AS address
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

	dbModels, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.GetSchedule])
	if err != nil {
		return nil, err
	}

	if len(dbModels) == 0 {
		return &entity.GetScheduleOutput{}, nil
	}

	keyword := dbModels[0].Keyword
	schedule := dbModels[0].Schedule
	subredditMap := make(map[int64]*entity.Subreddit)
	recipientMap := make(map[int64]*entity.Recipient)

	for _, m := range dbModels {
		if _, ok := subredditMap[m.SubredditID]; !ok {
			subredditMap[m.SubredditID] = &entity.Subreddit{
				ID:                m.SubredditID,
				Name:              m.Subreddit,
				IncludeNSFW:       m.IncludeNSFW,
				Sort:              m.Sort,
				RestrictSubreddit: m.RestrictSubreddit,
			}
		}

		if m.RecipientID.Valid {
			if _, ok := recipientMap[m.RecipientID.Int64]; !ok {
				recipientMap[m.RecipientID.Int64] = &entity.Recipient{
					ID:      m.RecipientID.Int64,
					Address: m.Address,
				}
			}
		}
	}

	subreddits := make([]*entity.Subreddit, 0, len(subredditMap))
	for _, v := range subredditMap {
		subreddits = append(subreddits, v)
	}

	recipients := make([]*entity.Recipient, 0, len(recipientMap))
	for _, v := range recipientMap {
		recipients = append(recipients, v)
	}

	return &entity.GetScheduleOutput{
		ID:         dbModels[0].ID,
		Keyword:    keyword,
		Schedule:   schedule,
		Recipients: recipients,
		Subreddits: subreddits,
		OwnerID:    0,
	}, nil
}

const deleteScheduleQuery = `DELETE FROM configuration WHERE id = @id`

func (h *Handle) DeleteSchedule(ctx context.Context, in *entity.DeleteScheduleInput) (*entity.DeleteScheduleOutput, error) {
	db, err := h.db()
	if err != nil {
		return nil, err
	}

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

	return &entity.DeleteScheduleOutput{}, nil
}

const listSchedulesQuery = `
SELECT
	c.id AS id,
	c.keyword AS keyword,
	c.schedule AS schedule,
	c.owner_id AS owner_id,
	sc.id AS subreddit_id,
	sc.subreddit AS subreddit,
	sc.include_nsfw AS include_nsfw,
	sc.sort AS sort,
	sc.restrict_subreddit AS restrict_subreddit,
	r.id AS recipient_id,
	r.address AS address
FROM
    configuration c
JOIN
	subreddit_configuration sc ON c.id = sc.configuration_id
LEFT JOIN
    recipients r ON c.id = r.configuration_id
`

func (h *Handle) ListSchedules(ctx context.Context, in *entity.ListSchedulesInput) (*entity.ListSchedulesOutput, error) {
	db, err := h.db()
	if err != nil {
		return nil, err
	}

	var sb strings.Builder
	sb.WriteString(listSchedulesQuery)
	args := pgx.NamedArgs{}

	if in.OwnerID != 0 {
		sb.WriteString(" WHERE c.owner_id = @owner_id")
		args["owner_id"] = in.OwnerID
	}

	sb.WriteString(" ORDER BY c.id;")

	rows, err := db.Query(ctx, sb.String(), args)
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
					ID:      m.RecipientID.Int64,
					Address: m.Address,
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

func (h *Handle) UpdateSchedule(ctx context.Context, in *entity.UpdateScheduleInput) (*entity.UpdateScheduleOutput, error) {
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

	if err = h.updateConfiguration(ctx, tx, in); err != nil {
		return nil, err
	}

	if err = h.syncSubreddits(ctx, tx, in.ID, in.Subreddits); err != nil {
		return nil, err
	}

	if err = h.syncRecipients(ctx, tx, in.ID, in.Recipients); err != nil {
		return nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &entity.UpdateScheduleOutput{}, nil
}

const updateConfigurationQuery = `
UPDATE configuration
SET keyword = @keyword, schedule = @schedule
WHERE id = @id
`

func (h *Handle) updateConfiguration(ctx context.Context, tx pgx.Tx, in *entity.UpdateScheduleInput) error {
	args := pgx.NamedArgs{
		"id":       in.ID,
		"keyword":  in.Keyword,
		"schedule": in.Schedule,
	}
	_, err := tx.Exec(ctx, updateConfigurationQuery, args)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handle) syncSubreddits(ctx context.Context, tx pgx.Tx, configurationID int64, subreddits []*entity.Subreddit) error {
	const getQ = `SELECT id FROM subreddit_configuration WHERE configuration_id = @configuration_id;`
	rows, err := tx.Query(ctx, getQ, pgx.NamedArgs{"configuration_id": configurationID})
	if err != nil {
		return err
	}

	existingIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return fmt.Errorf("failed to collect existing subreddits: %w", err)
	}

	existingIDSet := make(map[int64]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existingIDSet[id] = struct{}{}
	}

	desiredIDSet := make(map[int64]struct{})
	for _, sub := range subreddits {
		desiredIDSet[sub.ID] = struct{}{}
	}

	var toDelete []int64
	for id := range existingIDSet {
		if _, found := desiredIDSet[id]; !found {
			toDelete = append(toDelete, id)
		}
	}

	batch := &pgx.Batch{}
	if len(toDelete) > 0 {
		batch.Queue(`DELETE FROM subreddit_configuration WHERE id = ANY(@ids)`, pgx.NamedArgs{"ids": toDelete})
	}

	const upsertQ = `
INSERT INTO subreddit_configuration (id, configuration_id, subreddit, include_nsfw, sort, restrict_subreddit)
VALUES (@id, @configuration_id, @subreddit, @include_nsfw, @sort, @restrict_subreddit)
ON CONFLICT (id) DO UPDATE SET
                               subreddit = excluded.subreddit,
                               include_nsfw = excluded.include_nsfw,
                               sort = excluded.sort,
                               restrict_subreddit = excluded.restrict_subreddit
`

	for _, sub := range subreddits {
		args := pgx.NamedArgs{
			"id":                 sub.ID,
			"configuration_id":   configurationID,
			"subreddit":          sub.Name,
			"include_nsfw":       sub.IncludeNSFW,
			"sort":               sub.Sort,
			"restrict_subreddit": sub.RestrictSubreddit,
		}
		batch.Queue(upsertQ, args)
	}

	br := tx.SendBatch(ctx, batch)
	defer func() {
		_ = br.Close()
	}()
	for i := 0; i < batch.Len(); i++ {
		if _, err = br.Exec(); err != nil {
			return fmt.Errorf("error in subrredit sync batch operation: %w", err)
		}
	}

	return nil
}

func (h *Handle) syncRecipients(ctx context.Context, tx pgx.Tx, configurationID int64, recipients []*entity.Recipient) error {
	const getQ = `SELECT id FROM recipients WHERE configuration_id = @configuration_id;`
	rows, err := tx.Query(ctx, getQ, pgx.NamedArgs{"configuration_id": configurationID})
	if err != nil {
		return err
	}

	existingIDs, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return fmt.Errorf("failed to collect existing recipients: %w", err)
	}

	desiredIDSet := make(map[int64]struct{}, len(recipients))
	for _, id := range existingIDs {
		desiredIDSet[id] = struct{}{}
	}

	var toDelete []int64
	for _, id := range existingIDs {
		if _, found := desiredIDSet[id]; !found {
			toDelete = append(toDelete, id)
		}
	}

	batch := &pgx.Batch{}
	if len(toDelete) > 0 {
		batch.Queue(`DELETE FROM recipients WHERE id = ANY(@ids);`, pgx.NamedArgs{"ids": toDelete})
	}

	const upsertQ = `
INSERT INTO recipients (id, configuration_id, address)
VALUES (@id, @configuration_id, @address)
ON CONFLICT (id) DO UPDATE SET
                               address = EXCLUDED.address
`

	for _, recipient := range recipients {
		args := pgx.NamedArgs{
			"id":               recipient.ID,
			"configuration_id": configurationID,
			"address":          recipient.Address,
		}
		batch.Queue(upsertQ, args)
	}

	br := tx.SendBatch(ctx, batch)
	defer func() {
		_ = br.Close()
	}()

	for i := 0; i < batch.Len(); i++ {
		if _, err = br.Exec(); err != nil {
			return fmt.Errorf("error in recipient sync batch operation: %w", err)
		}
	}

	return nil
}
