package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const loadConfigurationAndStateQuery = `
SELECT
    c.keyword,
    (
        SELECT COALESCE(jsonb_agg(s), '[]')
        FROM (
            SELECT 
                sc.id, 
                sc.subreddit as name, 
                sc.include_nsfw, 
                sc.sort, 
                sc.restrict_subreddit,
                scs.last_post AS before
            FROM subreddit_configuration sc
            LEFT JOIN subreddit_configuration_state scs ON sc.id = scs.subreddit_configuration_id
            WHERE sc.configuration_id = c.id
        ) s
    ) AS subreddits,
    (
        SELECT COALESCE(jsonb_agg(r), '[]')
        FROM (
            SELECT r.id, r.address
            FROM recipients r
            WHERE r.configuration_id = c.id
        ) r
    ) AS recipients
FROM
    configuration c
WHERE
    c.id = @id;
`

func (h *Handle) LoadConfigurationAndState(ctx context.Context, in *LoadConfigurationAndStateInput) (*LoadConfigurationAndStateOutput, error) {
	args := pgx.NamedArgs{
		"id": in.ID,
	}

	rows, err := h.db.Query(ctx, loadConfigurationAndStateQuery, args)
	if err != nil {
		return nil, err
	}

	dbModel, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[models.LoadConfigurationAndState])
	if err != nil {
		return nil, err
	}

	var subreddits []*Subreddit
	if err = json.Unmarshal(dbModel.Subreddits, &subreddits); err != nil {
		return nil, err
	}

	var recipients []*Recipient
	if err = json.Unmarshal(dbModel.Recipients, &recipients); err != nil {
		return nil, err
	}

	return &LoadConfigurationAndStateOutput{
		Keyword:    dbModel.Keyword,
		Recipients: recipients,
		Subreddits: subreddits,
	}, nil
}

const updateStateQuery = `
INSERT INTO subreddit_configuration_state (subreddit_configuration_id, last_post) VALUES ($1, $2)
ON CONFLICT (subreddit_configuration_id) DO UPDATE 
SET last_post = EXCLUDED.last_post, last_updated_at = CURRENT_TIMESTAMP;
`

func (h *Handle) UpdateState(ctx context.Context, in *UpdateStateInput) (*UpdateStateOutput, error) {
	batch := &pgx.Batch{}

	for _, v := range in.Values {
		batch.Queue(updateStateQuery, v.SubredditConfigurationID, v.Before)
	}

	br := h.db.SendBatch(ctx, batch)
	defer func() {
		_ = br.Close()
	}()

	var errs error

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			errors.Join(errs, err)
		}
	}

	if errs != nil {
		return nil, fmt.Errorf("failed to update state: %w", errs)
	}

	return &UpdateStateOutput{}, nil
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

func (h *Handle) CreateSchedule(ctx context.Context, in *CreateScheduleInput) (*CreateScheduleOutput, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
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

	return &CreateScheduleOutput{}, nil
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

func (h *Handle) GetSchedule(ctx context.Context, in *GetScheduleInput) (*GetScheduleOutput, error) {
	args := pgx.NamedArgs{
		"id": in.ID,
	}

	rows, err := h.db.Query(ctx, getScheduleQuery, args)
	if err != nil {
		return nil, err
	}

	dbModels, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.GetSchedule])
	if err != nil {
		return nil, err
	}

	if len(dbModels) == 0 {
		return &GetScheduleOutput{}, nil
	}

	keyword := dbModels[0].Keyword
	schedule := dbModels[0].Schedule
	subredditMap := make(map[uuid.UUID]*Subreddit)
	recipientMap := make(map[uuid.UUID]*Recipient)

	for _, m := range dbModels {
		if _, ok := subredditMap[m.SubredditID]; !ok {
			subredditMap[m.SubredditID] = &Subreddit{
				ID:                m.SubredditID,
				Name:              m.Subreddit,
				IncludeNSFW:       m.IncludeNSFW,
				Sort:              m.Sort,
				RestrictSubreddit: m.RestrictSubreddit,
			}
		}

		if _, ok := recipientMap[m.RecipientID]; !ok {
			recipientMap[m.RecipientID] = &Recipient{
				ID:      m.RecipientID,
				Address: m.Address,
			}
		}
	}

	subreddits := make([]*Subreddit, 0, len(subredditMap))
	for _, v := range subredditMap {
		subreddits = append(subreddits, v)
	}

	recipients := make([]*Recipient, 0, len(recipientMap))
	for _, v := range recipientMap {
		recipients = append(recipients, v)
	}

	return &GetScheduleOutput{
		ID:         dbModels[0].ID,
		Keyword:    keyword,
		Schedule:   schedule,
		Recipients: recipients,
		Subreddits: subreddits,
	}, nil
}

const deleteScheduleQuery = `DELETE FROM configuration WHERE id = @id`

func (h *Handle) DeleteSchedule(ctx context.Context, in *DeleteScheduleInput) (*DeleteScheduleOutput, error) {
	args := pgx.NamedArgs{
		"id": in.ID,
	}

	cmd, err := h.db.Exec(ctx, deleteScheduleQuery, args)
	if err != nil {
		return nil, err
	}

	if cmd.RowsAffected() == 0 {
		return nil, errors.New("delete schedule row not found")
	}

	return &DeleteScheduleOutput{}, nil
}

const listSchedulesQuery = `
SELECT
	c.id AS id,
	c.keyword AS keyword,
	c.schedule AS schedule,
	(
	    SELECT COALESCE(jsonb_agg(sc), '[]')
	    FROM (
	        SELECT sc.id, sc.subreddit, sc.include_nsfw, sc.sort, sc.restrict_subreddit
	        FROM subreddit_configuration sc
	        WHERE sc.configuration_id = c.id
	    ) sc
	) as subreddits,
    (
        SELECT COALESCE(jsonb_agg(r), '[]')
        FROM (
            SELECT r.id, r.address
            FROM recipients r
            WHERE r.configuration_id = c.id
        ) r
    ) as recipients
FROM
    configuration c
ORDER BY c.id
`

func (h *Handle) ListSchedules(ctx context.Context, in *ListSchedulesInput) (*ListSchedulesOutput, error) {
	rows, err := h.db.Query(ctx, listSchedulesQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dbModels, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.ListSchedulesModel])
	if err != nil {
		return nil, err
	}

	var schedules = make([]*Schedule, 0, len(dbModels))

	for _, model := range dbModels {
		var subreddits []*Subreddit
		if err = json.Unmarshal(model.Subreddits, &subreddits); err != nil {
			return nil, err
		}

		var recipients []*Recipient
		if err = json.Unmarshal(model.Recipients, &recipients); err != nil {
			return nil, err
		}

		schedules = append(schedules, &Schedule{
			ID:         model.ID,
			Keyword:    model.Keyword,
			Schedule:   model.Keyword,
			Recipients: recipients,
			Subreddits: subreddits,
		})
	}

	return &ListSchedulesOutput{
		Schedules: schedules,
	}, nil
}

const updateScheduleQ = `
WITH input_data AS (
    SELECT 
        $1::uuid AS cfg_id,
        $2::text AS keyword,
        $3::text AS schedule,
        $4::jsonb AS subreddits,
        $5::jsonb AS recipients
),
update_configuration AS (
    UPDATE configuration 
    SET keyword = (SELECT keyword FROM input_data),
        schedule = (SELECT schedule FROM input_data)
    WHERE id = (SELECT cfg_id FROM input_data)
),
delete_subreddits AS (
    DELETE FROM subreddit_configuration
    WHERE configuration_id = (SELECT cfg_id FROM input_data)
    AND id NOT IN (SELECT (jsonb_array_elements(subreddits)->>'id')::uuid FROM input_data)
),
upsert_subreddits AS (
    INSERT INTO subreddit_configuration (id, configuration_id, subreddit, include_nsfw, sort, restrict_subreddit)
    SELECT 
        (e->>'id')::uuid, (SELECT cfg_id FROM input_data), e->>'name', 
        (e->>'include_nsfw')::bool, e->>'sort', (e->>'restrict_subreddit')::bool
    FROM input_data, jsonb_array_elements(subreddits) AS e
    ON CONFLICT (id) DO UPDATE SET
        subreddit = EXCLUDED.subreddit,
        include_nsfw = EXCLUDED.include_nsfw,
        sort = EXCLUDED.sort,
        restrict_subreddit = EXCLUDED.restrict_subreddit
),
delete_recipients AS (
    DELETE FROM recipients
    WHERE configuration_id = (SELECT cfg_id FROM input_data)
    AND id NOT IN (SELECT (jsonb_array_elements(recipients)->>'id')::uuid FROM input_data)
)
INSERT INTO recipients (id, configuration_id, address)
SELECT (e->>'id')::uuid, (SELECT cfg_id FROM input_data), e->>'address'
FROM input_data, jsonb_array_elements(recipients) AS e
ON CONFLICT (id) DO UPDATE SET address = EXCLUDED.address;
`

func (h *Handle) UpdateSchedule(ctx context.Context, in *UpdateScheduleInput) (*UpdateScheduleOutput, error) {
	subreddits, err := json.Marshal(in.Subreddits)
	if err != nil {
		return nil, err
	}

	recipients, err := json.Marshal(in.Recipients)
	if err != nil {
		return nil, err
	}

	if _, err = h.db.Exec(
		ctx,
		updateScheduleQ,
		in.ID,
		in.Keyword,
		in.Schedule,
		subreddits,
		recipients,
	); err != nil {
		return nil, err
	}

	return &UpdateScheduleOutput{}, nil
}

const queuePostsInsertQ = `INSERT INTO posts (id, configuration_id, data) VALUES (@id, @configuration_id, @data)`

func (h *Handle) QueuePosts(ctx context.Context, in *QueuePostsInput) (*QueuePostsOutput, error) {
	batch := &pgx.Batch{}

	for _, post := range in.Items {
		batch.Queue(
			queuePostsInsertQ,
			pgx.NamedArgs{
				"id":               post.ID,
				"configuration_id": post.ConfigurationID,
				"data":             post.Post,
			},
		)
	}

	br := h.db.SendBatch(ctx, batch)
	defer func() {
		_ = br.Close()
	}()

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return nil, fmt.Errorf("error in post queue batch operation: %w", err)
		}
	}

	return &QueuePostsOutput{}, nil
}

const getPostsSelectQ = `SELECT id, configuration_id, data FROM posts WHERE configuration_id = @configuration_id`

func (h *Handle) GetPosts(ctx context.Context, in *GetPostsInput) (*GetPostsOutput, error) {
	rows, err := h.db.Query(ctx, getPostsSelectQ, pgx.NamedArgs{"configuration_id": in.ConfigurationID})
	if err != nil {
		return nil, err
	}

	type postModel struct {
		ID              uuid.UUID `db:"id"`
		ConfigurationID uuid.UUID `db:"configuration_id"`
		Data            []byte    `db:"data"`
	}

	posts, err := pgx.CollectRows(rows, pgx.RowToStructByName[postModel])
	if err != nil {
		return nil, err
	}

	var items = make([]QueueItem, 0, len(posts))
	for _, post := range posts {
		items = append(items, QueueItem{
			ID:              post.ID,
			ConfigurationID: post.ConfigurationID,
			Post:            post.Data,
		})
	}

	return &GetPostsOutput{
		Items: items,
	}, nil
}

const popPostsDeleteQ = `DELETE FROM posts WHERE configuration_id = @configuration_id`

func (h *Handle) PopPosts(ctx context.Context, in *PopPostsInput) (*PopPostsOutput, error) {
	_, err := h.db.Exec(ctx, popPostsDeleteQ, pgx.NamedArgs{"configuration_id": in.ConfigurationID})
	if err != nil {
		return nil, err
	}

	return &PopPostsOutput{}, nil
}
