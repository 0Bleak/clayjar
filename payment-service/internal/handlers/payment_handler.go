package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/0Bleak/payment-service/internal/models"
	"github.com/0Bleak/payment-service/internal/service"
	"github.com/gorilla/mux"
)

type PaymentHandler struct {
	service service.PaymentService
}

func NewPaymentHandler(service service.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		service: service,
	}
}

func (h *PaymentHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/payments", h.CreatePayment).Methods(http.MethodPost)
	router.HandleFunc("/payments/{id}", h.GetPaymentByID).Methods(http.MethodGet)
	router.HandleFunc("/payments/order/{order_id}", h.GetPaymentByOrderID).Methods(http.MethodGet)
	router.HandleFunc("/health", h.HealthCheck).Methods(http.MethodGet)
}

func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var req models.CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	payment, err := h.service.CreatePayment(r.Context(), &req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, payment)
}

func (h *PaymentHandler) GetPaymentByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payment ID")
		return
	}

	payment, err := h.service.GetPaymentByID(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Payment not found")
		return
	}

	respondWithJSON(w, http.StatusOK, payment)
}

func (h *PaymentHandler) GetPaymentByOrderID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderIDStr := vars["order_id"]

	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	payment, err := h.service.GetPaymentByOrderID(r.Context(), orderID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Payment not found")
		return
	}

	respondWithJSON(w, http.StatusOK, payment)
}

func (h *PaymentHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
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
