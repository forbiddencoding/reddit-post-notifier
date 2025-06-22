package v1

import (
	"encoding/json"
	"github.com/forbiddencoding/reddit-post-notifier/services/app/reddit"
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
	"strconv"
)

type ScheduleHandler struct {
	scheduleService reddit.Servicer
}

func NewScheduleHandler(scheduleService reddit.Servicer) *ScheduleHandler {
	return &ScheduleHandler{
		scheduleService: scheduleService,
	}
}

func (h *ScheduleHandler) CreateSchedulePost() http.HandlerFunc {
	type (
		subreddit struct {
			Subreddit         string `json:"subreddit"`
			IncludeNSFW       bool   `json:"include_nsfw"`
			Sort              string `json:"sort"`
			RestrictSubreddit bool   `json:"restrict_subreddit"`
		}
		recipient struct {
			Address string `json:"address"`
		}

		request struct {
			Keyword    string       `json:"keyword"`
			Subreddits []*subreddit `json:"subreddits"`
			Schedule   string       `json:"schedule"`
			Recipients []*recipient `json:"recipients"`
		}
		response struct {
			ID int64 `json:"id"`
		}
	)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req request

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
			slog.Error("failed to write response", err)
		}
	}
}

func (h *ScheduleHandler) GetScheduleGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		res, err := h.scheduleService.GetSchedule(ctx, &reddit.GetScheduleInput{
			ScheduleID: id,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err = json.NewEncoder(w).Encode(res); err != nil {
			slog.Error("failed to write response", err)
		}
	}
}

func (h *ScheduleHandler) DeleteScheduleDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
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
			ID                int64  `json:"id,omitzero"`
			Subreddit         string `json:"subreddit"`
			IncludeNSFW       bool   `json:"include_nsfw"`
			Sort              string `json:"sort"`
			RestrictSubreddit bool   `json:"restrict_subreddit"`
		}
		recipient struct {
			ID      int64  `json:"id,omitzero"`
			Address string `json:"address"`
		}

		request struct {
			Keyword    string       `json:"keyword"`
			Subreddits []*subreddit `json:"subreddits"`
			Schedule   string       `json:"schedule"`
			Recipients []*recipient `json:"recipients"`
		}
	)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var req request

		if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		res, err := h.scheduleService.ListSchedules(ctx, &reddit.ListSchedulesInput{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err = json.NewEncoder(w).Encode(res); err != nil {
			slog.Error("failed to write response", err)
		}
	}
}
