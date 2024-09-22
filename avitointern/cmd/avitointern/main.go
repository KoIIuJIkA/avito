package main

import (
	"html/template"
	"log"
	"net/http"

	"avitointern/pkg/database"
	"avitointern/pkg/handlers"
	"avitointern/pkg/middleware"
	"avitointern/pkg/session"
	"avitointern/pkg/tenders"
	"avitointern/pkg/user"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func main() {

	templates := template.Must(template.ParseGlob("./static/html/*"))

	sm := session.NewSessionsManager()
	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Println("err with zapLogger")
	}
	defer func() {
		err = zapLogger.Sync()
		if err != nil {
			log.Println("zapLog err")
		}
	}()
	logger := zapLogger.Sugar()

	userRepo := user.NewMemoryRepo()
	tendersRepo := tenders.NewMemoryRepo()
	sqlManager := database.NewMemoryRepo()
	sqlManager.Init()

	userHandler := &handlers.UserHandler{
		Tmpl:     templates,
		UserRepo: userRepo,
		Logger:   logger,
		Sessions: sm,
	}

	handlers := &handlers.TendersHandler{
		SQL:         sqlManager,
		Tmpl:        templates,
		Logger:      logger,
		TendersRepo: tendersRepo,
	}

	r := mux.NewRouter()
	r.HandleFunc("/", userHandler.Index).Methods("GET")
	r.HandleFunc("/ping", userHandler.Ping).Methods("GET")
	r.HandleFunc("/login", userHandler.Login).Methods("POST")
	r.HandleFunc("/logout", userHandler.Logout).Methods("POST")

	r.HandleFunc("/tenders", handlers.Tenders).Methods("GET")
	r.HandleFunc("/tenders/new", handlers.New).Methods("POST")
	r.HandleFunc("/tenders/my", handlers.My).Methods("GET")
	r.HandleFunc("/tenders/{tenderID}/status", handlers.GetStatus).Methods("GET")
	r.HandleFunc("/tenders/{tenderID}/status", handlers.EditStatus).Methods("PUT")
	r.HandleFunc("/tenders/{tenderID}/edit", handlers.Edit).Methods("PATCH")
	r.HandleFunc("/tenders/{tenderID}/rollback/{version}", handlers.Rollback).Methods("PUT")

	mux := middleware.Auth(sm, r)
	mux = middleware.AccessLog(logger, mux)
	mux = middleware.Panic(mux)

	addr := ":8080"
	logger.Infow("starting server",
		"type", "START",
		"addr", addr,
	)
	err = http.ListenAndServe(addr, mux)
	if err != nil {
		log.Println("err with ListenAndServe")
	}
	handlers.SQL.Close()
}
