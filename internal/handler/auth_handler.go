package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/response"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	"github.com/azevedoguigo/demostore_api.git/pkg/utils"
	"gorm.io/gorm"
)

type AuthHandler struct {
	service service.AuthService
}

func NewAuthHandler(service service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req request.LoginRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	token, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		if err.Error() == "invalid password" {
			utils.HandleErrorResponse(w, http.StatusUnauthorized, err.Error())
			return
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HandleErrorResponse(w, http.StatusNotFound, "User not found")
			return
		}

		utils.HandleErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := response.AuthResponseDTO{
		AccessToken: token,
	}

	utils.JsonResponse(w, http.StatusOK, resp)
}
