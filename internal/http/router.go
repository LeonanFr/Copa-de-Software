package http

import (
	"copasoftware/internal/modules/checkin"
	"net/http"
	"os"

	"copasoftware/internal/database"
	"copasoftware/internal/modules/auth"
	"copasoftware/internal/modules/draw"
	"copasoftware/internal/modules/participants"
	"copasoftware/internal/modules/ranking"
	"copasoftware/internal/modules/reserves"
	"copasoftware/internal/modules/signup"
	"copasoftware/internal/modules/teamnames"
	"copasoftware/internal/modules/teams"

	"github.com/gorilla/mux"
)

func NewRouter(db *database.Mongo) http.Handler {
	router := mux.NewRouter()

	router.Use(RecoveryMiddleware)
	router.Use(LoggingMiddleware)
	router.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w, r)
		w.WriteHeader(http.StatusOK)
	})

	router.Use(corsMiddleware)

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte("OK"))
		}
	}).Methods("GET", "HEAD")

	participantRepo := participants.NewRepository(db)
	teamRepo := teams.NewRepository(db)
	teamNameRepo := teamnames.NewRepository(db)
	reserveRepo := reserves.NewRepository(db)
	rankingRepo := ranking.NewRepository(db)
	authRepo := auth.NewRepository(db)

	participantSvc := participants.NewService(participantRepo)

	teamNameSvc := teamnames.NewService(teamNameRepo)

	teamSvc := teams.NewService(teamRepo, participantSvc, teamNameSvc)

	participantSvc.SetTeamChecker(teamSvc)

	rankingSvc := ranking.NewService(rankingRepo, teamSvc)
	drawSvc := draw.NewService(participantSvc, teamSvc, rankingSvc)
	signupSvc := signup.NewService(participantSvc, teamSvc, teamNameSvc)
	reserveSvc := reserves.NewService(reserveRepo, participantSvc)

	participantSvc.SetReserveRemover(reserveSvc)

	jwtSecret := os.Getenv("JWT_SECRET")
	authSvc := auth.NewService(authRepo, jwtSecret, 24)

	rankingHandler := ranking.NewHandler(rankingSvc)

	checkinRepo := checkin.NewRepository(db)
	checkinSvc := checkin.NewService(checkinRepo, participantSvc)

	participants.RegisterPublicRoutes(router, participantSvc)
	signup.NewHandler(router, signupSvc)
	teams.RegisterPublicRoutes(router, teamSvc)
	ranking.RegisterPublicRoutes(router, rankingSvc)

	auth.NewHandler(router, authSvc)
	router.HandleFunc("/events/puzzle", rankingHandler.HandlePuzzleEvent).Methods("POST")
	router.HandleFunc("/events/case", rankingHandler.HandleCaseEvent).Methods("POST")
	router.HandleFunc("/events/algorithm", rankingHandler.HandleAlgorithmEvent).Methods("POST")

	teamSvc.SetRankingSvc(rankingSvc)

	adminRouter := router.PathPrefix("/admin").Subrouter()
	adminRouter.Use(AuthMiddleware(authSvc))

	teamnames.NewHandler(adminRouter, teamNameSvc)
	reserves.NewHandler(adminRouter, reserveSvc)
	draw.NewHandler(adminRouter, drawSvc, reserveSvc)
	teams.RegisterAdminRoutes(adminRouter, teamSvc)
	ranking.RegisterAdminRoutes(adminRouter, rankingSvc)
	participants.RegisterAdminRoutes(adminRouter, participantSvc)
	checkin.NewHandler(adminRouter, checkinSvc)

	return router
}
