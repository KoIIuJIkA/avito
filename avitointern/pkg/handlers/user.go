package handlers

import (
	"fmt"
	"html/template"
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
		http.Redirect(w, r, "/tenders", 302)
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
		fmt.Println("Error:", err)
		http.Redirect(w, r, "/", http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Ok"))
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

	sess, _ := h.Sessions.Create(w, u)

	h.Logger.Infof("created session for %v", sess.UserID)
	http.Redirect(w, r, "/", 302)
}

func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.Sessions.DestroyCurrent(w, r)
	http.Redirect(w, r, "/", 302)
}
