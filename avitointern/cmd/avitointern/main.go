package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"avitointern/pkg/handlers"
	"avitointern/pkg/middleware"
	"avitointern/pkg/session"
	"avitointern/pkg/tenders"
	"avitointern/pkg/user"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v4"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, proceeding with environment variables")
	}

	serverAddress := os.Getenv("SERVER_ADDRESS")
	postgresConn := os.Getenv("POSTGRES_CONN")
	postgresUsername := os.Getenv("POSTGRES_USERNAME")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresPort := os.Getenv("POSTGRES_PORT")
	postgresDatabase := os.Getenv("POSTGRES_DATABASE")

	fmt.Println(serverAddress, postgresConn)
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", postgresUsername, postgresPassword, postgresHost, postgresPort, postgresDatabase)

	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer conn.Close(context.Background())
	fmt.Println("Successfully connected to the database!")

	templates := template.Must(template.ParseGlob("./static/html/*"))

	sm := session.NewSessionsManager()
	zapLogger, _ := zap.NewProduction()
	defer zapLogger.Sync()
	logger := zapLogger.Sugar()

	userRepo := user.NewMemoryRepo()
	tendersRepo := tenders.NewMemoryRepo()

	userHandler := &handlers.UserHandler{
		Tmpl:     templates,
		UserRepo: userRepo,
		Logger:   logger,
		Sessions: sm,
	}

	handlers := &handlers.TendersHandler{
		Tmpl:        templates,
		Logger:      logger,
		TendersRepo: tendersRepo,
	}

	r := mux.NewRouter()
	r.HandleFunc("/", userHandler.Index).Methods("GET")
	r.HandleFunc("/ping", userHandler.Ping).Methods("GET")
	r.HandleFunc("/login", userHandler.Login).Methods("POST")
	r.HandleFunc("/logout", userHandler.Logout).Methods("POST")

	r.HandleFunc("/tenders", handlers.List).Methods("GET")
	r.HandleFunc("/tenders/new", handlers.NewForm).Methods("GET")
	r.HandleFunc("/tenders/new", handlers.New).Methods("POST")
	r.HandleFunc("/tenders/my", handlers.My).Methods("GET")
	r.HandleFunc("/tenders/{tenderID}/status", handlers.GetStatus).Methods("GET")
	r.HandleFunc("/tenders/{id}", handlers.Edit).Methods("GET")
	r.HandleFunc("/tenders/{id}", handlers.Update).Methods("POST")
	r.HandleFunc("/tenders/{id}", handlers.Delete).Methods("DELETE")

	mux := middleware.Auth(sm, r)
	mux = middleware.AccessLog(logger, mux)
	mux = middleware.Panic(mux)

	addr := ":8080"
	logger.Infow("starting server",
		"type", "START",
		"addr", addr,
	)
	http.ListenAndServe(addr, mux)
}
