// api/handler.go
package api

import (
	"encoding/json"
	"net/http"

	"api-movement/services"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type MovementHandler struct {
	service *services.MovimientoService
}

func NewMovementHandler(s *services.MovimientoService) *MovementHandler {
	return &MovementHandler{service: s}
}

func (h *MovementHandler) GetMovements(w http.ResponseWriter, r *http.Request) {
	movements, err := h.service.FindAllMovements()
	if err != nil {
		http.Error(w, "Error al consultar movimientos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movements)
}

func (h *MovementHandler) GetMovementByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "ID de movimiento inválido", http.StatusBadRequest)
		return
	}

	movement, err := h.service.FindMovementByID(id)
	if err != nil {
		http.Error(w, "Movimiento no encontrado: "+err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movement)
}