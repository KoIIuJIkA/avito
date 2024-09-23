package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"avitointern/pkg/database"
	"avitointern/pkg/session"
	"avitointern/pkg/tenders"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type TendersHandler struct {
	SQL         database.Database
	Tmpl        *template.Template
	TendersRepo tenders.TendersRepo
	Logger      *zap.SugaredLogger
}

type TenderResponse struct {
	TenderID          string `json:"id"`
	TenderName        string `json:"name"`
	TenderDescription string `json:"description"`
	Status            string `json:"status"`
	ServiceType       string `json:"serviceType"`
	Version           int32  `json:"version"`
	CreatedAt         string `json:"createdAt"` // RFC3339 format.
}

func (h *TendersHandler) Tenders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	limit, err := parseInt32(r, "limit", 5)
	if err != nil {
		h.errSend(w, "bad query", http.StatusBadRequest)
		return
	}

	offset, err := parseInt32(r, "offset", 0)
	if err != nil {
		h.errSend(w, "bad query", http.StatusBadRequest)
		return
	}

	var serviceType []tenders.ServiceType
	if serviceStr := r.URL.Query()["service_type"]; len(serviceType) != 0 {
		for _, service := range serviceStr {
			serviceType = append(serviceType, tenders.ServiceType(service))
		}
	}

	tenders, err := h.SQL.GetQuery(limit, offset, serviceType)
	if err != nil {
		h.errSend(w, "db err", http.StatusInternalServerError)
		return
	}

	for _, elem := range tenders {
		fmt.Println(elem)
	}

	err = json.NewEncoder(w).Encode(tenders)
	if err != nil {
		h.errSend(w, "json encoding error", http.StatusInternalServerError)
		return
	}
}

func (h *TendersHandler) New(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sess, err := session.SessionFromContext(r.Context())
	if sess.User.OrganizationID == "" {
		h.errSend(w, "user does not have an organization", http.StatusForbidden)
		return
	}
	if err != nil {
		h.errSend(w, "err with sess", http.StatusBadRequest)
		return
	}

	if err = r.ParseForm(); err != nil {
		h.errSend(w, "err with parseform", http.StatusBadRequest)
		return
	}
	var updateRequest struct {
		Name           *string              `json:"name"`
		Description    *string              `json:"description"`
		ServiceType    *tenders.ServiceType `json:"serviceType"`
		OrganizationID *string              `json:"organizationId"`
		Author         *string              `json:"creatorUsername"`
	}
	if err = json.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		h.errSend(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if updateRequest.Name == nil || updateRequest.Description == nil ||
		updateRequest.OrganizationID == nil || updateRequest.Author == nil {
		h.errSend(w, "bad json parse", http.StatusUnauthorized)
		return
	}

	tender := new(tenders.Tender)
	tender.TenderID = uuid.New().String()
	tender.TenderName = *updateRequest.Name
	tender.TenderDescription = *updateRequest.Description
	tender.ServiceType = *updateRequest.ServiceType
	tender.Status = tenders.Created
	tender.OrganizationID = *updateRequest.OrganizationID
	tender.Version = 1
	tender.CreatedAt = time.Now().Format(time.RFC3339) // RFC3339 format.
	tender.Author = sess.User.Username
	tender.Versions = make(map[int32]*tenders.TenderVer)

	tender.Versions[tender.Version] = &tenders.TenderVer{
		TenderName:        tender.TenderName,
		TenderDescription: tender.TenderDescription,
		ServiceType:       string(tender.ServiceType),
		Version:           1,
		Status:            tender.Status,
	}

	lastID, err := h.SQL.InsertTender(tender)
	if err != nil {
		h.errSend(w, "sql DB err", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	h.Logger.Infof("Insert with id LastInsertId: %v", lastID)
}

func (h *TendersHandler) My(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	limit, err := parseInt32(r, "limit", 5)
	if err != nil {
		h.errSend(w, "bad query in limit", http.StatusBadRequest)
		return
	}

	offset, err := parseInt32(r, "offset", 0)
	if err != nil {
		h.errSend(w, "bad query in offset", http.StatusBadRequest)
		return
	}

	username := r.URL.Query().Get("username")

	sess, err := session.SessionFromContext(r.Context())
	if err != nil {
		h.errSend(w, "session err", http.StatusInternalServerError)
		return
	}
	if username != sess.User.Username {
		h.errSend(w, "session and username err", http.StatusInternalServerError)
		return
	}

	tenders, err := h.SQL.My(limit, offset, sess.User.Username)
	if err != nil {
		h.errSend(w, "db err", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(tenders)
	if err != nil {
		h.errSend(w, "json encoding error", http.StatusInternalServerError)
		return
	}
}

func (h *TendersHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	username := r.URL.Query().Get("username")
	sess, err := session.SessionFromContext(r.Context())
	if username != sess.User.Username {
		h.errSend(w, "user Unauthorized", http.StatusUnauthorized)
		return
	}
	if err != nil {
		h.errSend(w, "session err", http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	id := vars["tenderID"]
	elem, err := h.SQL.GetTenderByID(id)
	if err != nil {
		h.errSend(w, "err with GetTenderByID", http.StatusBadRequest)
		return
	}
	if elem == nil {
		h.errSend(w, "the tender was not found", http.StatusNotFound)
		return
	}
	if elem.Author != username {
		h.errSend(w, "there are not enough permissions to perform the action", http.StatusForbidden)
		return
	}

	err = json.NewEncoder(w).Encode(elem.Status)
	if err != nil {
		h.errSend(w, "bad json encode", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	h.Logger.Infof("Status by ID: %v", elem.Status)
}

func (h *TendersHandler) EditStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := r.URL.Query().Get("status")
	if status == "" || !ContainsString([]string{"Created", "Published", "Closed"}, status) {
		h.errSend(w, "invalid format status", http.StatusBadRequest)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		h.errSend(w, "invalid format username", http.StatusBadRequest)
		return
	}
	sess, err := session.SessionFromContext(r.Context())
	if username != sess.User.Username {
		h.errSend(w, "user Unauthorized", http.StatusUnauthorized)
		return
	}
	if err != nil {
		h.errSend(w, "sess err", http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	tenderID := vars["tenderID"]
	elem, err := h.SQL.UpdateTenderStatus(tenderID, tenders.Status(status))
	if elem == nil {
		h.errSend(w, "the tender was not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.errSend(w, "bad json encode", http.StatusBadRequest)
		return
	}

	tender := TenderResponse{
		TenderID:          elem.TenderID,
		TenderName:        elem.TenderName,
		TenderDescription: elem.TenderDescription,
		Status:            string(elem.Status),
		ServiceType:       string(elem.ServiceType),
		Version:           elem.Version,
		CreatedAt:         elem.CreatedAt,
	}

	if err := json.NewEncoder(w).Encode(tender); err != nil {
		h.errSend(w, "error encoding JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	h.Logger.Infof("Edit status by ID: %v", elem.Status)
}

func (h *TendersHandler) Edit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	username := r.URL.Query().Get("username")
	if username == "" {
		h.errSend(w, "invalid format username", http.StatusBadRequest)
		return
	}
	sess, err := session.SessionFromContext(r.Context())
	if username != sess.User.Username {
		h.errSend(w, "user Unauthorized", http.StatusUnauthorized)
		return
	}
	if err != nil {
		h.errSend(w, "sess err", http.StatusBadRequest)
		return
	}

	var updateRequest struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		ServiceType *string `json:"serviceType"`
	}
	if err = json.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		h.errSend(w, "invalid request body", http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	tenderID := vars["tenderID"]
	elem, err := h.SQL.GetTenderByID(tenderID)
	if elem == nil {
		h.errSend(w, "the tender was not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.errSend(w, "err with GetTenderByID", http.StatusBadRequest)
		return
	}

	err = json.NewEncoder(w).Encode(elem.Status)
	if err != nil {
		h.errSend(w, "bad json encode", http.StatusBadRequest)
		return
	}

	if updateRequest.Name != nil {
		elem.TenderName = *updateRequest.Name
	}
	if updateRequest.Description != nil {
		elem.TenderDescription = *updateRequest.Description
	}
	if updateRequest.ServiceType != nil {
		elem.ServiceType = tenders.ServiceType(*updateRequest.ServiceType)
	}

	var tender *tenders.Tender
	if updateRequest.Name != nil || updateRequest.Description != nil || updateRequest.ServiceType != nil {
		tender, err = h.SQL.EditTender(elem.TenderID, elem.TenderName, elem.TenderDescription, elem.ServiceType)
		if err != nil {
			h.errSend(w, "my tender err", http.StatusInternalServerError)
			return
		}
	}
	if err := json.NewEncoder(w).Encode(tender); err != nil {
		h.errSend(w, "error encoding JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	h.Logger.Infof("EditTender PUT status by ID: %v", elem.Status)
}

func (h *TendersHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	username := r.URL.Query().Get("username")
	if username == "" {
		h.errSend(w, "invalid format username", http.StatusBadRequest)
		return
	}
	sess, err := session.SessionFromContext(r.Context())
	if username != sess.User.Username {
		h.errSend(w, "user Unauthorized", http.StatusUnauthorized)
		return
	}
	if err != nil {
		h.errSend(w, "sess err", http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	tenderID := vars["tenderID"]
	elem, err := h.SQL.GetTenderByID(tenderID)
	if elem == nil {
		h.errSend(w, "the tender was not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.errSend(w, "err with GetTenderByID", http.StatusBadRequest)
		return
	}

	err = json.NewEncoder(w).Encode(elem.Status)
	if err != nil {
		h.errSend(w, "bad json encode", http.StatusBadRequest)
		return
	}

	versionStr := vars["version"]
	version, err := strconv.ParseInt(versionStr, 10, 32)
	if err != nil {
		h.errSend(w, "bad parse version", http.StatusNotFound)
		return
	}

	tender, err := h.SQL.Rollback(tenderID, int32(version))
	if err != nil {
		h.errSend(w, "bad sql request wherer", http.StatusNotFound)
		return
	}

	fmt.Println(tender)

	if err := json.NewEncoder(w).Encode(tender); err != nil {
		h.errSend(w, "error encoding JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	h.Logger.Infof("EditTender PUT status by ID: %v", elem.Status)
}

func ContainsString(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func parseInt32(r *http.Request, param string, defaultVal int32) (int32, error) {
	re := r.URL.Query()
	if strNum := re.Get(param); strNum != "" {
		num, err := strconv.ParseInt(strNum, 10, 32)
		if err != nil {
			return 0, err
		}
		return int32(num), nil
	}
	return defaultVal, nil
}

func (h *TendersHandler) errSend(w http.ResponseWriter, reason string, status int) {
	var errorResponse struct {
		Reason string `json:"reason"`
	}
	w.WriteHeader(status)
	errorResponse.Reason = reason
	err := json.NewEncoder(w).Encode(errorResponse)
	if err != nil {
		h.Logger.Infof("err in h.errSend with encode")
	}
}
