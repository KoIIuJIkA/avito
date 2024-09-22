package handlers

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"avitointern/pkg/session"
	"avitointern/pkg/user"

	"go.uber.org/zap"
)

type UserHandler struct {
	Tmpl     *template.Template
	Logger   *zap.SugaredLogger
	UserRepo user.UserRepo
	Sessions *session.SessionsManager
}

func (h *UserHandler) Index(w http.ResponseWriter, r *http.Request) {
	_, err := session.SessionFromContext(r.Context())
	if err == nil {
		http.Redirect(w, r, "/tenders", http.StatusFound)
		return
	}

	err = h.Tmpl.ExecuteTemplate(w, "login.html", nil)
	if err != nil {
		http.Error(w, `Template error`, http.StatusInternalServerError)
		return
	}
}

func (h *UserHandler) Ping(w http.ResponseWriter, r *http.Request) {
	client := http.Client{
		Timeout: 1 * time.Second,
	}

	_, err := client.Get("http://localhost:8080")
	if err != nil {
		log.Println("Error:", err)
		http.Redirect(w, r, "/", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusOK)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	login := r.URL.Query().Get("login")
	password := r.URL.Query().Get("password")
	u, err := h.UserRepo.Authorize(login, password)
	if err == user.ErrNoUser {
		http.Error(w, `no user`, http.StatusBadRequest)
		return
	}
	if err == user.ErrBadPass {
		http.Error(w, `bad pass`, http.StatusBadRequest)
		return
	}

	sess, err := h.Sessions.Create(w, u)
	if err != nil {
		log.Println("err in sess in handlers/user.go")
	}

	h.Logger.Infof("created session for %v", sess.UserID)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	err := h.Sessions.DestroyCurrent(w, r)
	if err != nil {
		h.Logger.Infof("err in logout")
	}
	http.Redirect(w, r, "/", http.StatusFound)
}
