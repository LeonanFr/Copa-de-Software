package http

import (
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
	router.Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
	})

	router.Use(corsMiddleware)

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}).Methods("GET")

	participantRepo := participants.NewRepository(db)
	teamRepo := teams.NewRepository(db)
	teamNameRepo := teamnames.NewRepository(db)
	reserveRepo := reserves.NewRepository(db)
	rankingRepo := ranking.NewRepository(db)
	authRepo := auth.NewRepository(db)

	participantSvc := participants.NewService(participantRepo)
	teamNameSvc := teamnames.NewService(teamNameRepo)

	teamSvc := teams.NewService(teamRepo, participantSvc, teamNameSvc)
	rankingSvc := ranking.NewService(rankingRepo, teamSvc)
	drawSvc := draw.NewService(participantSvc, teamSvc, teamNameSvc, rankingSvc)
	signupSvc := signup.NewService(participantSvc, teamSvc, teamNameSvc)
	reserveSvc := reserves.NewService(reserveRepo, participantSvc)

	jwtSecret := os.Getenv("JWT_SECRET")
	authSvc := auth.NewService(authRepo, jwtSecret, 24)

	participants.RegisterPublicRoutes(router, participantSvc)
	signup.NewHandler(router, signupSvc)
	teams.RegisterPublicRoutes(router, teamSvc)
	ranking.RegisterPublicRoutes(router, rankingSvc)

	auth.NewHandler(router, authSvc)

	adminRouter := router.PathPrefix("/admin").Subrouter()
	adminRouter.Use(AuthMiddleware(authSvc))

	teamnames.NewHandler(adminRouter, teamNameSvc)
	reserves.NewHandler(adminRouter, reserveSvc)
	draw.NewHandler(adminRouter, drawSvc, reserveSvc)
	teams.RegisterAdminRoutes(adminRouter, teamSvc, rankingSvc)
	ranking.RegisterAdminRoutes(adminRouter, rankingSvc)
	participants.RegisterAdminRoutes(adminRouter, participantSvc)

	return router
}
