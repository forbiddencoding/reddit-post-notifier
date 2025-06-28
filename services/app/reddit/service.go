package reddit

import (
	"context"
	"fmt"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence"
	"github.com/forbiddencoding/reddit-post-notifier/common/persistence/entity"
	"github.com/forbiddencoding/reddit-post-notifier/services/digester"
	"github.com/go-playground/validator/v10"
	"github.com/sony/sonyflake/v2"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"sort"
)

type (
	Servicer interface {
		CreateSchedule(ctx context.Context, in *CreateScheduleInput) (*CreateScheduleOutput, error)
		GetSchedule(ctx context.Context, in *GetScheduleInput) (*GetScheduleOutput, error)
		UpdateSchedule(ctx context.Context, in *UpdateScheduleInput) (*UpdateScheduleOutput, error)
		DeleteSchedule(ctx context.Context, in *DeleteScheduleInput) (*DeleteScheduleOutput, error)
		ListSchedules(ctx context.Context, in *ListSchedulesInput) (*ListSchedulesOutput, error)
	}

	Service struct {
		db             persistence.Persistence
		temporalClient client.Client
		sonyflake      *sonyflake.Sonyflake
		validator      *validator.Validate
	}
)

var _ Servicer = (*Service)(nil)

func NewService(
	db persistence.Persistence,
	temporalClient client.Client,
	sonyflake *sonyflake.Sonyflake,
	validator *validator.Validate,
) (Servicer, error) {
	return &Service{
		db:             db,
		temporalClient: temporalClient,
		sonyflake:      sonyflake,
		validator:      validator,
	}, nil
}

func (s *Service) CreateSchedule(ctx context.Context, in *CreateScheduleInput) (*CreateScheduleOutput, error) {
	if err := s.validator.Struct(in); err != nil {
		return nil, err
	}

	id, err := s.sonyflake.NextID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}

	var (
		recipients = make([]*entity.CreateScheduleRecipient, 0, len(in.Recipients))
		subreddits = make([]*entity.CreateScheduleSubreddit, 0, len(in.Subreddits))
	)

	for _, recipient := range in.Recipients {
		recipientID, err := s.sonyflake.NextID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate ID: %w", err)
		}

		recipients = append(recipients, &entity.CreateScheduleRecipient{
			ID:      recipientID,
			Address: recipient.Address,
		})
	}

	for _, subreddit := range in.Subreddits {
		subredditID, err := s.sonyflake.NextID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate ID: %w", err)
		}
		subreddits = append(subreddits, &entity.CreateScheduleSubreddit{
			ID:                subredditID,
			Subreddit:         subreddit.Subreddit,
			IncludeNSFW:       subreddit.IncludeNSFW,
			Sort:              subreddit.Sort,
			RestrictSubreddit: subreddit.RestrictSubreddit,
		})
	}

	_, err = s.db.CreateSchedule(ctx, &entity.CreateScheduleInput{
		ID:         id,
		Keyword:    in.Keyword,
		Schedule:   in.Schedule,
		OwnerID:    0,
		Recipients: recipients,
		Subreddits: subreddits,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	_, err = s.temporalClient.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: fmt.Sprintf("reddit_posts::%d", id),
		Action: &client.ScheduleWorkflowAction{
			Workflow: digester.DigestWorkflow,
			Args: []any{&digester.DigestWorkflowInput{
				ID: id,
			}},
			TaskQueue: "digester",
		},
		Spec: client.ScheduleSpec{
			CronExpressions: []string{in.Schedule},
		},
	})
	if err != nil {
		return nil, err
	}

	return &CreateScheduleOutput{
		Created:    true,
		ScheduleID: id,
	}, nil
}

func (s *Service) GetSchedule(ctx context.Context, in *GetScheduleInput) (*GetScheduleOutput, error) {
	schedule, err := s.db.GetSchedule(ctx, &entity.GetScheduleInput{
		ID: in.ScheduleID,
	})
	if err != nil {
		return nil, err
	}

	var (
		subreddits = make([]*Subreddit, 0, len(schedule.Subreddits))
		recipients = make([]*Recipient, 0, len(schedule.Recipients))
	)

	for _, subreddit := range schedule.Subreddits {
		subreddits = append(subreddits, &Subreddit{
			Subreddit:         subreddit.Name,
			IncludeNSFW:       subreddit.IncludeNSFW,
			Sort:              subreddit.Sort,
			RestrictSubreddit: subreddit.RestrictSubreddit,
		})
	}

	for _, recipient := range schedule.Recipients {
		recipients = append(recipients, &Recipient{
			ID:      recipient.ID,
			Address: recipient.Address,
		})
	}

	handle := s.temporalClient.ScheduleClient().GetHandle(ctx, fmt.Sprintf("reddit_posts::%d", in.ScheduleID))
	desc, err := handle.Describe(ctx)
	if err != nil {
		return nil, err
	}

	var status = "NONE"

	if len(desc.Info.RecentActions) > 0 {
		exec, err := s.temporalClient.DescribeWorkflowExecution(ctx, desc.Info.RecentActions[0].StartWorkflowResult.WorkflowID, "")
		if err != nil {
			return nil, err
		}

		switch exec.WorkflowExecutionInfo.Status {
		case enums.WORKFLOW_EXECUTION_STATUS_RUNNING:
			status = "RUNNING"
		case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
			status = "COMPLETED"
		case enums.WORKFLOW_EXECUTION_STATUS_FAILED:
			status = "FAILED"
		}
	}

	return &GetScheduleOutput{
		ID:                  in.ScheduleID,
		Keyword:             schedule.Keyword,
		Subreddits:          subreddits,
		Schedule:            schedule.Schedule,
		Recipients:          recipients,
		NextActionTimes:     desc.Info.NextActionTimes,
		Paused:              desc.Schedule.State.Paused,
		LastExecutionStatus: status,
	}, nil
}

// UpdateSchedule updates both the database entry and the corresponding temporal schedule. In theory this would also be
// a great use case for a temporal workflow as this would guarantee that a possible cron expression change is properly
// stored in the database and set in the temporal schedule.
func (s *Service) UpdateSchedule(ctx context.Context, in *UpdateScheduleInput) (*UpdateScheduleOutput, error) {
	if err := s.validator.Struct(in); err != nil {
		return nil, err
	}

	var (
		recipients = make([]*entity.Recipient, 0, len(in.Recipients))
		subreddits = make([]*entity.Subreddit, 0, len(in.Subreddits))
	)

	for _, recipient := range in.Recipients {
		id, err := s.sonyflake.NextID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate ID: %w", err)
		}
		if recipient.ID != 0 {
			id = recipient.ID
		}

		recipients = append(recipients, &entity.Recipient{
			ID:      id,
			Address: recipient.Address,
		})
	}

	for _, subreddit := range in.Subreddits {
		id, err := s.sonyflake.NextID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate ID: %w", err)
		}
		if subreddit.ID != 0 {
			id = subreddit.ID
		}
		subreddits = append(subreddits, &entity.Subreddit{
			ID:                id,
			Name:              subreddit.Subreddit,
			IncludeNSFW:       subreddit.IncludeNSFW,
			Sort:              subreddit.Sort,
			RestrictSubreddit: subreddit.RestrictSubreddit,
		})
	}

	_, err := s.db.UpdateSchedule(ctx, &entity.UpdateScheduleInput{
		ID:         in.ID,
		Keyword:    in.Keyword,
		Schedule:   in.Schedule,
		Recipients: recipients,
		Subreddits: subreddits,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update schedule: %w", err)
	}

	handle := s.temporalClient.ScheduleClient().GetHandle(ctx, fmt.Sprintf("reddit_posts::%d", in.ID))

	if _, err = handle.Describe(ctx); err != nil {
		return nil, fmt.Errorf("failed to describe schedule: %w", err)
	}

	updateSchedule := func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
		schedule := input.Description.Schedule

		// Calendars and Intervals are explicitly set to nil, otherwise this would prevent us from updating the schedule
		// using CronExpressions
		schedule.Spec.Calendars = nil
		schedule.Spec.Intervals = nil
		schedule.Spec.CronExpressions = []string{in.Schedule}

		return &client.ScheduleUpdate{
			Schedule: &schedule,
		}, nil
	}

	if err = handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: updateSchedule,
	}); err != nil {
		return nil, err
	}

	return nil, nil
}

func (s *Service) DeleteSchedule(ctx context.Context, in *DeleteScheduleInput) (*DeleteScheduleOutput, error) {
	handle := s.temporalClient.ScheduleClient().GetHandle(ctx, fmt.Sprintf("reddit_posts::%d", in.ID))
	if err := handle.Delete(ctx); err != nil {
		return nil, err
	}

	if _, err := s.db.DeleteSchedule(ctx, &entity.DeleteScheduleInput{ID: in.ID}); err != nil {
		return nil, err
	}

	return nil, nil
}

func (s *Service) ListSchedules(ctx context.Context, in *ListSchedulesInput) (*ListSchedulesOutput, error) {
	res, err := s.db.ListSchedules(ctx, &entity.ListSchedulesInput{
		OwnerID: in.OwnerID,
	})
	if err != nil {
		return nil, err
	}

	schedules := make([]*Schedule, 0, len(res.Schedules))
	for _, schedule := range res.Schedules {
		subreddits := make([]*Subreddit, 0, len(schedule.Subreddits))
		for _, subreddit := range schedule.Subreddits {
			subreddits = append(subreddits, &Subreddit{
				ID:                subreddit.ID,
				Subreddit:         subreddit.Name,
				IncludeNSFW:       subreddit.IncludeNSFW,
				Sort:              subreddit.Sort,
				RestrictSubreddit: subreddit.RestrictSubreddit,
			})
		}

		recipients := make([]*Recipient, 0, len(schedule.Recipients))
		for _, recipient := range schedule.Recipients {
			recipients = append(recipients, &Recipient{
				ID:      recipient.ID,
				Address: recipient.Address,
			})
		}

		schedules = append(schedules, &Schedule{
			ID:         schedule.ID,
			Keyword:    schedule.Keyword,
			Subreddits: subreddits,
			Schedule:   schedule.Schedule,
			Recipients: recipients,
			OwnerID:    schedule.OwnerID,
		})
	}

	sort.Slice(schedules, func(i, j int) bool {
		return schedules[i].ID > schedules[j].ID
	})

	return &ListSchedulesOutput{
		Schedules: schedules,
	}, nil
}
