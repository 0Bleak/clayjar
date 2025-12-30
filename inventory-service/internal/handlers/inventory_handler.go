package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/0Bleak/inventory-service/internal/models"
	"github.com/0Bleak/inventory-service/internal/service"
	"github.com/gorilla/mux"
)

type InventoryHandler struct {
	service service.InventoryService
}

func NewInventoryHandler(service service.InventoryService) *InventoryHandler {
	return &InventoryHandler{
		service: service,
	}
}

func (h *InventoryHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/inventory", h.CreateInventory).Methods(http.MethodPost)
	router.HandleFunc("/inventory/{jar_id}", h.GetInventory).Methods(http.MethodGet)
	router.HandleFunc("/inventory/{jar_id}", h.UpdateInventory).Methods(http.MethodPut)
	router.HandleFunc("/health", h.HealthCheck).Methods(http.MethodGet)
}

func (h *InventoryHandler) CreateInventory(w http.ResponseWriter, r *http.Request) {
	var req models.CreateInventoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	inventory, err := h.service.CreateInventory(r.Context(), &req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, inventory)
}

func (h *InventoryHandler) GetInventory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jarID := vars["jar_id"]

	inventory, err := h.service.GetInventory(r.Context(), jarID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Inventory not found")
		return
	}

	respondWithJSON(w, http.StatusOK, inventory)
}

func (h *InventoryHandler) UpdateInventory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jarID := vars["jar_id"]

	var req models.UpdateInventoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	inventory, err := h.service.UpdateInventory(r.Context(), jarID, &req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, inventory)
}

func (h *InventoryHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
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
