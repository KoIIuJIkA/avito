package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"avitointern/pkg/session"
	"avitointern/pkg/tenders"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"go.uber.org/zap"
)

type TendersHandler struct {
	Tmpl        *template.Template
	TendersRepo tenders.TendersRepo
	Logger      *zap.SugaredLogger
}

func (h *TendersHandler) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var query struct {
		Limit       *int32                `json:"limit"`
		Offset      *int32                `json:"offset"`
		ServiceType []tenders.ServiceType `json:"service_type"`
	}
	var errorResponse struct {
		Reason string `json:"reason"`
	}

	err := json.NewDecoder(r.Body).Decode(&query)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorResponse.Reason = "Bad query"
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	limit := int32(5)
	offset := int32(0)

	if query.Limit != nil {
		limit = *query.Limit
	}
	if query.Offset != nil {
		offset = *query.Offset
	}

	elems, err := h.TendersRepo.GetQuery(limit, offset, query.ServiceType)
	if err != nil {
		http.Error(w, `DB err`, http.StatusInternalServerError)
		return
	}

	for _, elem := range elems {
		fmt.Println(elem)
	}

	err = json.NewEncoder(w).Encode(elems)
	if err != nil {
		http.Error(w, `JSON encoding error`, http.StatusInternalServerError)
		return
	}
}

// To fill in via an html form, depricated.
func (h *TendersHandler) NewForm(w http.ResponseWriter, r *http.Request) {
	err := h.Tmpl.ExecuteTemplate(w, "create.html", nil)
	if err != nil {
		http.Error(w, `Template error`, http.StatusInternalServerError)
		return
	}
}

func (h *TendersHandler) New(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sess, _ := session.SessionFromContext(r.Context())
	if sess.User.OrganizationID == "" {
		w.WriteHeader(http.StatusForbidden)
		var errorResponse struct {
			Reason string `json:"reason"`
		}
		errorResponse.Reason = "User does not have an organization"
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	r.ParseForm()
	tender := new(tenders.Tender)
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err := decoder.Decode(tender, r.PostForm)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		var errorResponse struct {
			Reason string `json:"reason"`
		}
		errorResponse.Reason = "Bad form here"
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	tender.TenderID = uuid.New().String()
	tender.Status = tenders.Created
	tender.Version = 1
	tender.CreatedAt = time.Now().Format(time.RFC3339) // RFC3339 format.
	tender.Author = sess.User.Username
	lastID, err := h.TendersRepo.Add(tender)
	if err != nil {
		http.Error(w, `DB err`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

	h.Logger.Infof("Insert with id LastInsertId: %v", lastID)
}

func (h *TendersHandler) My(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var query struct {
		Limit  *int32 `json:"limit"`
		Offset *int32 `json:"offset"`
	}
	var errorResponse struct {
		Reason string `json:"reason"`
	}

	err := json.NewDecoder(r.Body).Decode(&query)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorResponse.Reason = "Bad query"
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	limit := int32(5)
	offset := int32(0)

	if query.Limit != nil {
		limit = *query.Limit
	}
	if query.Offset != nil {
		offset = *query.Offset
	}
	sess, _ := session.SessionFromContext(r.Context())

	elems, err := h.TendersRepo.GetMy(limit, offset, sess.User.Username)
	if err != nil {
		http.Error(w, `DB err`, http.StatusInternalServerError)
		return
	}

	for _, elem := range elems {
		fmt.Println(elem)
	}

	err = json.NewEncoder(w).Encode(elems)
	if err != nil {
		http.Error(w, `JSON encoding error`, http.StatusInternalServerError)
		return
	}
}

func (h *TendersHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var errorResponse struct {
		Reason string `json:"reason"`
	}
	username := r.URL.Query().Get("username")
	sess, _ := session.SessionFromContext(r.Context())

	// fmt.Print("username = ", username, ", sess.User.Username = ", sess.User.Username)
	if username != sess.User.Username {
		w.WriteHeader(http.StatusUnauthorized)
		errorResponse.Reason = "User Unauthorized"
		json.NewEncoder(w).Encode(errorResponse)
		return
	}
	if !h.TendersRepo.Check(username) {
		w.WriteHeader(http.StatusForbidden)
		errorResponse.Reason = "There are not enough permissions to perform the action."
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	vars := mux.Vars(r)
	id := vars["tenderID"]
	elem, err := h.TendersRepo.GetByID(id)
	if elem == nil {
		w.WriteHeader(http.StatusNotFound)
		errorResponse.Reason = "The tender was not found."
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	err = json.NewEncoder(w).Encode(elem.Status)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorResponse.Reason = "bad json encode"
		json.NewEncoder(w).Encode(errorResponse)
		return
	}
	w.WriteHeader(http.StatusOK)

	h.Logger.Infof("Status by ID: %v", elem.Status)
}

// WARNING! Tempalate
func (h *TendersHandler) Edit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	tender, err := h.TendersRepo.GetByID(id)
	if err != nil {
		http.Error(w, `DB err`, http.StatusInternalServerError)
		return
	}
	if tender == nil {
		http.Error(w, `no tender`, http.StatusNotFound)
		return
	}

	err = h.Tmpl.ExecuteTemplate(w, "edit.html", tender)
	if err != nil {
		http.Error(w, `Template error`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// WARNING! Template.
func (h *TendersHandler) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	r.ParseForm()
	tender := new(tenders.Tender)
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	err := decoder.Decode(tender, r.PostForm)
	if err != nil {
		http.Error(w, `Bad forms`, http.StatusBadRequest)
		return
	}
	tender.TenderID = id

	ok, err := h.TendersRepo.Update(tender)
	if err != nil {
		http.Error(w, `db error`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	h.Logger.Infof("update: %v %v", tender, ok)
}

// WARNING! Template
func (h *TendersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ok, err := h.TendersRepo.Delete(id)
	if err != nil {
		http.Error(w, `{"error": "db error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-type", "application/json")
	respJSON, _ := json.Marshal(map[string]bool{
		"success": ok,
	})
	w.Write(respJSON)
}
