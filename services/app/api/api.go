package api

import (
	"github.com/forbiddencoding/reddit-post-notifier/services/app"
	v1 "github.com/forbiddencoding/reddit-post-notifier/services/app/api/v1"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"net/http"
)

func NewRouter(app *app.App) http.Handler {
	r := chi.NewRouter()
	r.Use(
		middleware.CleanPath,
		middleware.RealIP,
		middleware.RequestID,
		middleware.Recoverer,
		middleware.RedirectSlashes,
		cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300,
		}),
	)

	r.Route("/v1", func(r chi.Router) {
		r.Route("/schedule", func(r chi.Router) {
			scheduleHandler := v1.NewScheduleHandler(app.ScheduleService(), app.Validator())

			r.Post("/", scheduleHandler.CreateSchedulePost())
			r.Get("/", scheduleHandler.ListSchedulesGet())

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", scheduleHandler.GetScheduleGet())
				r.Put("/", scheduleHandler.UpdateSchedulePut())
				r.Delete("/", scheduleHandler.DeleteScheduleDelete())
			})
		})
	})

	return r
}
