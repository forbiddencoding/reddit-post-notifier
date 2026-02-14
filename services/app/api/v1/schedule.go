package v1

import (
	"encoding/json"
	"github.com/forbiddencoding/reddit-post-notifier/services/app/reddit"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"time"
)

type ScheduleHandler struct {
	scheduleService reddit.Servicer
	validator       *validator.Validate
}

func NewScheduleHandler(
	scheduleService reddit.Servicer,
	validator *validator.Validate,
) *ScheduleHandler {
	return &ScheduleHandler{
		scheduleService: scheduleService,
		validator:       validator,
	}
}

func (h *ScheduleHandler) CreateSchedulePost() http.HandlerFunc {
	type (
		subreddit struct {
			Subreddit         string `json:"subreddit" validate:"required"`
			IncludeNSFW       bool   `json:"include_nsfw"`
			Sort              string `json:"sort"`
			RestrictSubreddit bool   `json:"restrict_subreddit"`
		}
		recipient struct {
			Address string `json:"address" validate:"required,email"`
		}

		request struct {
			Keyword    string       `json:"keyword" validate:"required"`
			Subreddits []*subreddit `json:"subreddits" validate:"required,min=1,max=10"`
			Schedule   string       `json:"schedule" validate:"required,cron"`
			Recipients []*recipient `json:"recipients" validate:"required,min=1,max=10"`
		}
		response struct {
			ID uuid.UUID `json:"id"`
		}
	)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req request

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := h.validator.Struct(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var (
			subreddits = make([]*reddit.Subreddit, 0, len(req.Subreddits))
			recipients = make([]*reddit.Recipient, 0, len(req.Recipients))
		)

		for _, sub := range req.Subreddits {
			subreddits = append(subreddits, &reddit.Subreddit{
				Subreddit:         sub.Subreddit,
				IncludeNSFW:       sub.IncludeNSFW,
				Sort:              sub.Sort,
				RestrictSubreddit: sub.RestrictSubreddit,
			})
		}

		for _, rec := range req.Recipients {
			recipients = append(recipients, &reddit.Recipient{
				Address: rec.Address,
			})
		}

		res, err := h.scheduleService.CreateSchedule(ctx, &reddit.CreateScheduleInput{
			Keyword:    req.Keyword,
			Subreddits: subreddits,
			Schedule:   req.Schedule,
			Recipients: recipients,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		if err = json.NewEncoder(w).Encode(response{
			ID: res.ScheduleID,
		}); err != nil {
			slog.Error("write response", slog.Any("error", err))
		}
	}
}

func (h *ScheduleHandler) GetScheduleGet() http.HandlerFunc {
	type (
		subreddit struct {
			ID                uuid.UUID `json:"id"`
			Subreddit         string    `json:"subreddit" validate:"required"`
			IncludeNSFW       bool      `json:"includeNSFW"`
			Sort              string    `json:"sort"`
			RestrictSubreddit bool      `json:"restrictSubreddit"`
		}

		recipient struct {
			ID      uuid.UUID `json:"id"`
			Address string    `json:"address" validate:"required,email"`
		}

		response struct {
			ID                  uuid.UUID    `json:"id"`
			Keyword             string       `json:"keyword"`
			Subreddits          []*subreddit `json:"subreddits"`
			Schedule            string       `json:"schedule"`
			Recipients          []*recipient `json:"recipients"`
			NextActionTimes     []time.Time  `json:"nextActionTimes"`
			Paused              bool         `json:"paused"`
			LastExecutionStatus string       `json:"lastExecutionStatus"`
		}
	)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		schedule, err := h.scheduleService.GetSchedule(ctx, &reddit.GetScheduleInput{
			ScheduleID: id,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var (
			subreddits      = make([]*subreddit, 0, len(schedule.Subreddits))
			recipients      = make([]*recipient, 0, len(schedule.Subreddits))
			nextActionTimes = make([]time.Time, 0, len(schedule.Subreddits))
		)

		for _, sub := range schedule.Subreddits {
			subreddits = append(subreddits, &subreddit{
				ID:                sub.ID,
				Subreddit:         sub.Subreddit,
				IncludeNSFW:       sub.IncludeNSFW,
				Sort:              sub.Sort,
				RestrictSubreddit: sub.RestrictSubreddit,
			})
		}

		for _, rec := range schedule.Recipients {
			recipients = append(recipients, &recipient{
				ID:      rec.ID,
				Address: rec.Address,
			})
		}

		for _, nextActionTime := range schedule.NextActionTimes {
			nextActionTimes = append(nextActionTimes, nextActionTime)
		}

		res := response{
			ID:                  schedule.ID,
			Keyword:             schedule.Keyword,
			Subreddits:          subreddits,
			Schedule:            schedule.Schedule,
			Recipients:          recipients,
			NextActionTimes:     nextActionTimes,
			Paused:              schedule.Paused,
			LastExecutionStatus: schedule.LastExecutionStatus,
		}

		w.Header().Set("Content-Type", "application/json")
		if err = json.NewEncoder(w).Encode(res); err != nil {
			slog.Error("write response", slog.Any("error", err))
		}
	}
}

func (h *ScheduleHandler) DeleteScheduleDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		_, err = h.scheduleService.DeleteSchedule(ctx, &reddit.DeleteScheduleInput{
			ID: id,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		_, _ = w.Write(nil)
	}
}

func (h *ScheduleHandler) UpdateSchedulePut() http.HandlerFunc {
	type (
		subreddit struct {
			ID                uuid.UUID `json:"id,omitempty"`
			Subreddit         string    `json:"subreddit" validate:"required"`
			IncludeNSFW       bool      `json:"includeNSFW"`
			Sort              string    `json:"sort"`
			RestrictSubreddit bool      `json:"restrictSubreddit"`
		}
		recipient struct {
			ID      uuid.UUID `json:"id"`
			Address string    `json:"address" validate:"required,email"`
		}

		request struct {
			Keyword    string       `json:"keyword" validate:"required"`
			Subreddits []*subreddit `json:"subreddits" validate:"required,min=1,max=10"`
			Schedule   string       `json:"schedule" validate:"required,cron"`
			Recipients []*recipient `json:"recipients" validate:"required,min=1,max=10"`
		}
	)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var req request

		if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err = h.validator.Struct(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var (
			subreddits = make([]*reddit.Subreddit, 0, len(req.Subreddits))
			recipients = make([]*reddit.Recipient, 0, len(req.Recipients))
		)

		for _, sub := range req.Subreddits {
			subreddits = append(subreddits, &reddit.Subreddit{
				ID:                sub.ID,
				Subreddit:         sub.Subreddit,
				IncludeNSFW:       sub.IncludeNSFW,
				Sort:              sub.Sort,
				RestrictSubreddit: sub.RestrictSubreddit,
			})
		}

		for _, rec := range req.Recipients {
			recipients = append(recipients, &reddit.Recipient{
				ID:      rec.ID,
				Address: rec.Address,
			})
		}

		_, err = h.scheduleService.UpdateSchedule(ctx, &reddit.UpdateScheduleInput{
			ID:         id,
			Keyword:    req.Keyword,
			Subreddits: subreddits,
			Schedule:   req.Schedule,
			Recipients: recipients,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		_, _ = w.Write(nil)
	}
}

func (h *ScheduleHandler) ListSchedulesGet() http.HandlerFunc {
	type (
		subreddit struct {
			ID                uuid.UUID `json:"id"`
			Subreddit         string    `json:"subreddit" validate:"required"`
			IncludeNSFW       bool      `json:"includeNSFW"`
			Sort              string    `json:"sort"`
			RestrictSubreddit bool      `json:"restrictSubreddit"`
		}

		recipient struct {
			ID      uuid.UUID `json:"id"`
			Address string    `json:"address" validate:"required,email"`
		}

		scheduleForList struct {
			ID         uuid.UUID    `json:"id"`
			Keyword    string       `json:"keyword"`
			Subreddits []*subreddit `json:"subreddits"`
			Schedule   string       `json:"schedule"`
			Recipients []*recipient `json:"recipients"`
		}

		response struct {
			Schedules []*scheduleForList `json:"schedules"`
		}
	)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		list, err := h.scheduleService.ListSchedules(ctx, &reddit.ListSchedulesInput{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var schedules = make([]*scheduleForList, 0, len(list.Schedules))
		for _, sched := range list.Schedules {
			var (
				subreddits = make([]*subreddit, 0, len(sched.Subreddits))
				recipients = make([]*recipient, 0, len(sched.Recipients))
			)

			for _, sub := range sched.Subreddits {
				subreddits = append(subreddits, &subreddit{
					ID:                sub.ID,
					Subreddit:         sub.Subreddit,
					IncludeNSFW:       sub.IncludeNSFW,
					Sort:              sub.Sort,
					RestrictSubreddit: sub.RestrictSubreddit,
				})
			}

			for _, rec := range sched.Recipients {
				recipients = append(recipients, &recipient{
					ID:      rec.ID,
					Address: rec.Address,
				})
			}

			schedules = append(schedules, &scheduleForList{
				ID:         sched.ID,
				Keyword:    sched.Keyword,
				Subreddits: subreddits,
				Schedule:   sched.Schedule,
				Recipients: recipients,
			})
		}

		var res = response{Schedules: schedules}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err = json.NewEncoder(w).Encode(res); err != nil {
			slog.Error("write response", slog.Any("error", err))
		}
	}
}
