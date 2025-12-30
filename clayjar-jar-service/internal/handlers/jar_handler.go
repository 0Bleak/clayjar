package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/0Bleak/clayjar-jar-service/internal/models"
	"github.com/0Bleak/clayjar-jar-service/internal/service"
	"github.com/gorilla/mux"
)

type JarHandler struct {
	service service.JarService
}

func NewJarHandler(service service.JarService) *JarHandler {
	return &JarHandler{
		service: service,
	}
}

func (h *JarHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/jars", h.CreateJar).Methods(http.MethodPost)
	router.HandleFunc("/jars", h.GetAllJars).Methods(http.MethodGet)
	router.HandleFunc("/jars/{id}", h.GetJarByID).Methods(http.MethodGet)
	router.HandleFunc("/jars/{id}", h.UpdateJar).Methods(http.MethodPut)
	router.HandleFunc("/jars/{id}", h.DeleteJar).Methods(http.MethodDelete)
	router.HandleFunc("/health", h.HealthCheck).Methods(http.MethodGet)
}

func (h *JarHandler) CreateJar(w http.ResponseWriter, r *http.Request) {
	var req models.CreateJarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	jar, err := h.service.CreateJar(r.Context(), &req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, jar)
}

func (h *JarHandler) GetJarByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	jar, err := h.service.GetJarByID(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Jar not found")
		return
	}

	respondWithJSON(w, http.StatusOK, jar)
}

func (h *JarHandler) GetAllJars(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := int64(10)
	offset := int64(0)

	if limitStr != "" {
		if l, err := strconv.ParseInt(limitStr, 10, 64); err == nil {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.ParseInt(offsetStr, 10, 64); err == nil {
			offset = o
		}
	}

	jars, err := h.service.GetAllJars(r.Context(), limit, offset)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, jars)
}

func (h *JarHandler) UpdateJar(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req models.CreateJarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	jar, err := h.service.UpdateJar(r.Context(), id, &req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, jar)
}

func (h *JarHandler) DeleteJar(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.service.DeleteJar(r.Context(), id); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Jar deleted successfully"})
}

func (h *JarHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
