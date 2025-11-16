package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"pullreq/internal/pr"
	"pullreq/internal/team"
	"pullreq/internal/user"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: .env file not found, using system environment variables")
	}
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	serverPort := os.Getenv("SERVER_PORT")

	db, err := initDB(sugar, dbHost, dbPort, dbUser, dbPassword, dbName)
	if err != nil {
		log.Println(dbHost, dbPort, dbUser, dbPassword, dbName, "Your data")
		sugar.Fatalw("Failed to connect to DB", "error", err)
	}
	defer db.Close()

	userRepo := &user.UserRepo{DB: db}
	teamRepo := &team.TeamRepo{DB: db, UR: userRepo}
	prRepo := &pr.PullRequestRepo{DB: db, UR: userRepo, TR: teamRepo}

	teamRouter := &team.TeamRouter{TR: teamRepo}
	userRouter := &user.UserRouter{UR: userRepo}
	prRouter := &pr.PrRouter{PR: prRepo}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(15 * time.Second))
	r.Use(zapLoggerMiddleware(sugar))
	r.Use(middleware.Recoverer)

	r.Route("/team", func(r chi.Router) {
		r.Post("/add", teamRouter.HandleAddTeam)
		r.Get("/get", teamRouter.GetTeamWithMembersHandler)
		r.Post("/deactivation", teamRouter.DeactivateTeam)
	})

	r.Route("/users", func(r chi.Router) {
		r.Post("/setIsActive", userRouter.RouterSetActiviry)
		r.Get("/getReview", userRouter.GetUserReviewsHandler)
		r.Get("/getStat", userRouter.GetStat)
	})

	r.Route("/pullRequest", func(r chi.Router) {
		r.Post("/create", prRouter.CreatePullRequest)
		r.Post("/merge", prRouter.Merge)
		r.Post("/reassign", prRouter.AssignedReviewer)
	})

	srv := &http.Server{
		Addr:    ":" + serverPort,
		Handler: r,
	}

	go func() {
		sugar.Infow("Server listening", "port", serverPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sugar.Fatalw("HTTP server error", "error", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	sugar.Infow("Shutting down server gracefully")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		sugar.Fatalw("Server forced to shutdown", "error", err)
	}
	sugar.Infow("Server stopped")
}

func initDB(logger *zap.SugaredLogger, host, port, user, password, dbname string) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	logger.Infow("Connected to DB successfully")
	return db, nil
}

func zapLoggerMiddleware(logger *zap.SugaredLogger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(rw, r)
			logger.Infow("HTTP request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.Status(),
				"duration", time.Since(start),
				"remote_ip", r.RemoteAddr,
			)
		})
	}
}
