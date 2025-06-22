package mysql

import (
	"context"
	"errors"
	"fmt"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/entity"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/mysql/models"
	"github.com/jmoiron/sqlx"
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
	scs.last_post AS last_post,
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
		subredditMap = make(map[int64]*entity.Subreddit)
		recipientMap = make(map[int64]*entity.Recipient)
	)

	for rows.Next() {
		var m models.LoadConfigurationAndState

		if err = rows.StructScan(&m); err != nil {
			return nil, err
		}

		if keyword == "" {
			keyword = m.Keyword
		}

		if _, ok := subredditMap[m.SubredditID]; !ok {
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
INSERT INTO subreddit_configuration_state (subreddit_configuration_id, last_post) VALUES (:configuration_id, :last_post) 
ON DUPLICATE KEY UPDATE last_post = :last_post;
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

	stmt, err := tx.PrepareNamedContext(ctx, updateStateQuery)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = stmt.Close()
	}()

	var errs error

	for _, v := range in.Values {
		args := map[string]any{
			"configuration_id": v.SubredditConfigurationID,
			"last_post":        v.Before,
		}
		if _, err = stmt.ExecContext(ctx, args); err != nil {
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

const (
	createScheduleConfigurationQuery = `
INSERT INTO 
    configuration (id, keyword, schedule) 
VALUES (:id, :keyword, :schedule)
`
	createScheduleSubredditConfigurationQuery = `
INSERT INTO 
    subreddit_configuration (id, configuration_id, subreddit, include_nsfw, sort, restrict_subreddit) 
VALUES (:id, :configuration_id, :subreddit, :include_nsfw, :sort, :restrict_subreddit) 
`
	createScheduleRecipientsQuery = `
INSERT INTO 
    recipients (id, configuration_id, address)
VALUES (:id, :configuration_id, :address)`
)

func (h *Handle) CreateSchedule(ctx context.Context, in *entity.CreateScheduleInput) (*entity.CreateScheduleOutput, error) {
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

	args := map[string]any{
		"id":       in.ID,
		"keyword":  in.Keyword,
		"schedule": in.Schedule,
	}

	if _, err = tx.NamedExecContext(ctx, createScheduleConfigurationQuery, args); err != nil {
		return nil, fmt.Errorf("failed to create configuration: %w", err)
	}

	for _, subreddit := range in.Subreddits {
		arg := map[string]any{
			"id":                 subreddit.ID,
			"configuration_id":   in.ID,
			"subreddit":          subreddit.Subreddit,
			"include_nsfw":       subreddit.IncludeNSFW,
			"sort":               subreddit.Sort,
			"restrict_subreddit": subreddit.RestrictSubreddit,
		}
		if _, err = tx.NamedExecContext(ctx, createScheduleSubredditConfigurationQuery, arg); err != nil {
			return nil, fmt.Errorf("failed to create subreddit: %w", err)
		}
	}

	for _, recipient := range in.Recipients {
		arg := map[string]any{
			"id":               recipient.ID,
			"configuration_id": in.ID,
			"address":          recipient.Address,
		}
		if _, err = tx.NamedExecContext(ctx, createScheduleRecipientsQuery, arg); err != nil {
			return nil, fmt.Errorf("failed to create recipient: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
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
    	c.id = ?;
`

func (h *Handle) GetSchedule(ctx context.Context, in *entity.GetScheduleInput) (*entity.GetScheduleOutput, error) {
	db, err := h.db()
	if err != nil {
		return nil, err
	}

	args := []any{
		in.ID,
	}

	var dbModels []models.GetSchedule
	if err = db.SelectContext(ctx, &dbModels, getScheduleQuery, args...); err != nil {
		return nil, err
	}

	if len(dbModels) == 0 {
		return &entity.GetScheduleOutput{}, nil
	}

	var (
		subredditMap = make(map[int64]*entity.Subreddit)
		recipientMap = make(map[int64]*entity.Recipient)
	)

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
		Keyword:    dbModels[0].Keyword,
		Schedule:   dbModels[0].Schedule,
		Recipients: recipients,
		Subreddits: subreddits,
		OwnerID:    0,
	}, nil
}

const deleteScheduleQuery = `DELETE FROM configuration WHERE id = :id`

func (h *Handle) DeleteSchedule(ctx context.Context, in *entity.DeleteScheduleInput) (*entity.DeleteScheduleOutput, error) {
	db, err := h.db()
	if err != nil {
		return nil, err
	}

	args := map[string]any{
		"id": in.ID,
	}

	result, err := db.NamedExecContext(ctx, deleteScheduleQuery, args)
	if err != nil {
		return nil, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	if rows == 0 {
		return &entity.DeleteScheduleOutput{}, errors.New("delete schedule row not found")
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
	subreddit_configuration_state scs ON sc.id = scs.subreddit_configuration_id
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
	var args map[string]any

	if in.OwnerID != 0 {
		sb.WriteString(" WHERE c.owner_id = ?")
		args["owner_id"] = in.OwnerID
	}
	sb.WriteString(" ORDER BY c.id;")

	rows, err := db.NamedQueryContext(ctx, sb.String(), args)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	schedulesMap := make(map[int64]*entity.Schedule)
	subredditSets := make(map[int64]map[int64]struct{})
	recipientSets := make(map[int64]map[int64]struct{})

	for rows.Next() {
		var m models.ListSchedulesModel
		if err = rows.StructScan(&m); err != nil {
			return nil, err
		}

		if _, ok := schedulesMap[m.ID]; !ok {
			schedulesMap[m.ID] = &entity.Schedule{
				ID:         m.ID,
				OwnerID:    m.OwnerID,
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

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
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

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
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

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil, nil
}

const updateConfigurationQuery = `
UPDATE configuration
SET keyword = :keyword, schedule = :schedule
WHERE id = :id;
`

func (h *Handle) updateConfiguration(ctx context.Context, tx *sqlx.Tx, in *entity.UpdateScheduleInput) error {
	args := map[string]any{
		"id":       in.ID,
		"keyword":  in.Keyword,
		"schedule": in.Schedule,
	}
	_, err := tx.NamedExecContext(ctx, updateConfigurationQuery, args)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handle) syncSubreddits(ctx context.Context, tx *sqlx.Tx, configurationID int64, subreddits []*entity.Subreddit) error {
	const getQ = `SELECT id FROM subreddit_configuration WHERE configuration_id = ?`
	var existingIDs []int64
	if err := tx.SelectContext(ctx, &existingIDs, getQ, configurationID); err != nil {
		return err
	}

	existingIDSet := make(map[int64]struct{})
	for _, id := range existingIDs {
		existingIDSet[id] = struct{}{}
	}

	desiredIDSet := make(map[int64]struct{})
	for _, id := range subreddits {
		desiredIDSet[id.ID] = struct{}{}
	}

	var toDelete []int64
	for _, id := range existingIDs {
		if _, found := desiredIDSet[id]; !found {
			toDelete = append(toDelete, id)
		}
	}

	if len(toDelete) > 0 {
		const delQ = `DELETE FROM subreddit_configuration WHERE id IN (?)`
		q, args, err := sqlx.In(delQ, toDelete)
		if err != nil {
			return fmt.Errorf("failed to build delete query for subreddits: %w", err)
		}
		query := tx.Rebind(q)
		if _, err = tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to delete subreddits: %w", err)
		}
	}

	const upsertQ = `
INSERT INTO subreddit_configuration (id, configuration_id, subreddit, include_nsfw, sort, restrict_subreddit)
VALUES (:id, :configuration_id, :subreddit, :include_nsfw, :sort, :restrict_subreddit)
ON DUPLICATE KEY UPDATE
	subreddit = :subreddit,
	include_nsfw = :include_nsfw,
	sort = :sort,
	restrict_subreddit = :restrict_subreddit
`
	for _, sub := range subreddits {
		args := map[string]any{
			"id":                 sub.ID,
			"configuration_id":   configurationID,
			"subreddit":          sub.Name,
			"include_nsfw":       sub.IncludeNSFW,
			"sort":               sub.Sort,
			"restrict_subreddit": sub.RestrictSubreddit,
		}
		if _, err := tx.NamedExecContext(ctx, upsertQ, args); err != nil {
			return fmt.Errorf("failed to update subreddit %d: %w", sub.ID, err)
		}
	}

	return nil
}

func (h *Handle) syncRecipients(ctx context.Context, tx *sqlx.Tx, configurationID int64, recipients []*entity.Recipient) error {
	const getQ = `SELECT id FROM recipients WHERE configuration_id = ?`
	var existingIDs []int64
	if err := tx.SelectContext(ctx, &existingIDs, getQ, configurationID); err != nil {
		return err
	}

	existingIDSet := make(map[int64]struct{})
	for _, id := range existingIDs {
		existingIDSet[id] = struct{}{}
	}

	desiredIDSet := make(map[int64]struct{})
	for _, id := range recipients {
		desiredIDSet[id.ID] = struct{}{}
	}

	var toDelete []int64
	for _, id := range existingIDs {
		if _, found := desiredIDSet[id]; !found {
			toDelete = append(toDelete, id)
		}
	}

	if len(toDelete) > 0 {
		const delQ = `DELETE FROM recipients WHERE id IN (?)`
		q, args, err := sqlx.In(delQ, toDelete)
		if err != nil {
			return fmt.Errorf("failed to build delete query for recipients: %w", err)
		}
		query := tx.Rebind(q)
		if _, err = tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to delete recipients: %w", err)
		}
	}

	const upsertQ = `
INSERT INTO recipients (id, configuration_id, address)
VALUES (:id, :configuration_id, :address)
ON DUPLICATE KEY UPDATE
								address = :address
`
	for _, recipient := range recipients {
		args := map[string]any{
			"id":               recipient.ID,
			"configuration_id": configurationID,
			"address":          recipient.Address,
		}
		if _, err := tx.NamedExecContext(ctx, upsertQ, args); err != nil {
			return fmt.Errorf("failed to update recipient %d: %w", recipient.ID, err)
		}
	}

	return nil
}
