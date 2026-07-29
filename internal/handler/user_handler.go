package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	"github.com/azevedoguigo/demostore_api.git/pkg/utils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserHandler struct {
	service service.UserService
}

func NewUserHandler(service service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// CreateUser godoc
//
//	@Summary		Cria um novo usuário
//	@Description	Cadastra um novo usuário na plataforma
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		request.CreateUserRequestDTO	true	"Dados do usuário"
//	@Success		201		{object}	map[string]string
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/users [post]
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateUserRequestDTO

	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.service.CreateUser(&dto); err != nil {
		utils.HandleErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.JsonResponse(
		w,
		http.StatusCreated,
		map[string]string{"message": "User created successfully"},
	)
}

// GetUserByID godoc
//
//	@Summary		Busca um usuário pelo ID
//	@Description	Retorna os dados de um usuário a partir do seu UUID
//	@Tags			users
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"ID do usuário (UUID)"
//	@Success		200	{object}	domain.User
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/users/{id} [get]
func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	userUUID, err := uuid.Parse(id)
	if err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.service.GetUserByID(userUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HandleErrorResponse(w, http.StatusNotFound, "User not found")
			return
		}

		utils.HandleErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.JsonResponse(w, http.StatusOK, user)
}
