package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"avitointern/pkg/session"
	"avitointern/pkg/tenders"
	"avitointern/pkg/user"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type TendersHandler struct {
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

	var errorResponse struct {
		Reason string `json:"reason"`
	}

	limit, err := parseInt32(r, "limit", 5)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorResponse.Reason = "Bad query"
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	offset, err := parseInt32(r, "offset", 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorResponse.Reason = "Bad query"
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	var serviceType []tenders.ServiceType
	if serviceStr := r.URL.Query()["service_type"]; len(serviceType) != 0 {
		for _, service := range serviceStr {
			serviceType = append(serviceType, tenders.ServiceType(service))
		}
	}

	fmt.Println("limit = ", limit, ", offset = ", offset, ", serviceType = ", serviceType)
	tenders, err := h.TendersRepo.GetQuery(limit, offset, serviceType)
	if err != nil {
		http.Error(w, `DB err`, http.StatusInternalServerError)
		return
	}

	for _, elem := range tenders {
		fmt.Println(elem)
	}

	fmt.Println(tenders)

	err = json.NewEncoder(w).Encode(tenders)
	if err != nil {
		http.Error(w, `JSON encoding error`, http.StatusInternalServerError)
		return
	}
}

func (h *TendersHandler) New(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var errorResponse struct {
		Reason string     `json:"reason"`
		Debug  *user.User `json:"debug"`
	}
	sess, _ := session.SessionFromContext(r.Context())
	if sess.User.OrganizationID == "" {
		w.WriteHeader(http.StatusForbidden)
		errorResponse.Reason = "User does not have an organization"
		errorResponse.Debug = sess.User
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	r.ParseForm()
	tender := new(tenders.Tender)
	var updateRequest struct {
		Name           *string `json:"name"`
		Description    *string `json:"description"`
		OrganizationId *string `json:"organizationId"`
		Author         *string `json:"creatorUsername"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorResponse.Reason = "Invalid request body."
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	if updateRequest.Name == nil || updateRequest.Description == nil || updateRequest.OrganizationId == nil || updateRequest.Author == nil {
		w.WriteHeader(http.StatusUnauthorized)
		errorResponse.Reason = "bad json parse"
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	tender.TenderID = uuid.New().String()
	tender.Status = tenders.Created
	tender.Version = 1
	tender.CreatedAt = time.Now().Format(time.RFC3339) // RFC3339 format.
	tender.Author = sess.User.Username
	tender.Versions = make(map[int32]*tenders.TenderVer)
	tender.Versions[tender.Version] = &tenders.TenderVer{
		TenderName:        tender.TenderName,
		TenderDescription: tender.TenderDescription,
		ServiceType:       string(tender.ServiceType),
		Version:           1,
	}

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

	var errorResponse struct {
		Reason string `json:"reason"`
	}

	limit, err := parseInt32(r, "limit", 5)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorResponse.Reason = "Bad query in limit"
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	offset, err := parseInt32(r, "offset", 0)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorResponse.Reason = "Bad query in offset"
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	sess, _ := session.SessionFromContext(r.Context())

	tenders, err := h.TendersRepo.GetMy(limit, offset, sess.User.Username)
	if err != nil {
		http.Error(w, `DB err`, http.StatusInternalServerError)
		return
	}

	for _, elem := range tenders {
		fmt.Println(elem)
	}

	err = json.NewEncoder(w).Encode(tenders)
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

func (h *TendersHandler) EditStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var errorResponse struct {
		Reason string `json:"reason"`
	}

	status := r.URL.Query().Get("status")
	if status == "" || !ContainsString([]string{"Created", "Published", "Closed"}, status) {
		w.WriteHeader(http.StatusBadRequest)
		errorResponse.Reason = "invalid format status."
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		w.WriteHeader(http.StatusBadRequest)
		errorResponse.Reason = "invalid format username."
		json.NewEncoder(w).Encode(errorResponse)
		return
	}
	sess, _ := session.SessionFromContext(r.Context())
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
	tenderID := vars["tenderID"]
	elem, err := h.TendersRepo.GetByID(tenderID)
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

	elem.Status = tenders.Status(status)
	elem.Version = int32(len(elem.Versions))
	elem.Versions[elem.Version] = &tenders.TenderVer{
		TenderName:        elem.TenderName,
		TenderDescription: elem.TenderDescription,
		ServiceType:       string(elem.ServiceType),
		Version:           elem.Version,
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

	fmt.Println(tender) // delDEL

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(tender); err != nil {
		http.Error(w, `{"reason": "Error encoding JSON"}`, http.StatusInternalServerError)
		return
	}

	h.Logger.Infof("Edit status by ID: %v", elem.Status)
}

func (h *TendersHandler) Edit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var errorResponse struct {
		Reason string `json:"reason"`
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		w.WriteHeader(http.StatusBadRequest)
		errorResponse.Reason = "invalid format username."
		json.NewEncoder(w).Encode(errorResponse)
		return
	}
	sess, _ := session.SessionFromContext(r.Context())
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

	var updateRequest struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		ServiceType *string `json:"serviceType"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errorResponse.Reason = "Invalid request body."
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	vars := mux.Vars(r)
	tenderID := vars["tenderID"]
	elem, err := h.TendersRepo.GetByID(tenderID)
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

	if updateRequest.Name != nil {
		elem.TenderName = *updateRequest.Name
	}
	if updateRequest.Description != nil {
		elem.TenderDescription = *updateRequest.Description
	}
	if updateRequest.ServiceType != nil {
		elem.ServiceType = tenders.ServiceType(*updateRequest.ServiceType)
	}

	if updateRequest.Name != nil || updateRequest.Description != nil || updateRequest.ServiceType != nil {
		elem.Version = int32(len(elem.Versions))
		elem.Versions[elem.Version] = &tenders.TenderVer{
			TenderName:        elem.TenderName,
			TenderDescription: elem.TenderDescription,
			ServiceType:       string(elem.ServiceType),
		}
	}

	tender := &TenderResponse{
		TenderID:          elem.TenderID,
		TenderName:        elem.TenderName,
		TenderDescription: elem.TenderDescription,
		Status:            string(elem.Status),
		ServiceType:       string(elem.ServiceType),
		Version:           elem.Version,
		CreatedAt:         elem.CreatedAt,
	}

	fmt.Println(tender) // delDEL

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(tender); err != nil {
		http.Error(w, `{"reason": "Error encoding JSON"}`, http.StatusInternalServerError)
		return
	}

	h.Logger.Infof("EditTender PUT status by ID: %v", elem.Status)
}

func (h *TendersHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var errorResponse struct {
		Reason string `json:"reason"`
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		w.WriteHeader(http.StatusBadRequest)
		errorResponse.Reason = "invalid format username."
		json.NewEncoder(w).Encode(errorResponse)
		return
	}
	sess, _ := session.SessionFromContext(r.Context())
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
	tenderID := vars["tenderID"]
	elem, err := h.TendersRepo.GetByID(tenderID)
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

	versionStr := vars["version"]
	version, err := strconv.ParseInt(versionStr, 10, 32)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		errorResponse.Reason = "Bad parse version"
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	if _, err := elem.Versions[int32(version)]; !err {
		w.WriteHeader(http.StatusNotFound)
		errorResponse.Reason = "There is no such version"
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	rollbackTender := elem.Versions[int32(version)]
	rollbackTender.Version = int32(len(elem.Versions))
	elem.Versions[rollbackTender.Version] = rollbackTender

	elem.TenderName = rollbackTender.TenderName
	elem.TenderDescription = rollbackTender.TenderDescription
	elem.ServiceType = tenders.ServiceType(rollbackTender.ServiceType)

	tender := TenderResponse{
		TenderID:          elem.TenderID,
		TenderName:        elem.TenderName,
		TenderDescription: elem.TenderDescription,
		Status:            string(elem.Status),
		ServiceType:       string(elem.ServiceType),
		Version:           elem.Version,
		CreatedAt:         elem.CreatedAt,
	}

	fmt.Println(tender)

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(tender); err != nil {
		http.Error(w, `{"reason": "Error encoding JSON"}`, http.StatusInternalServerError)
		return
	}

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
