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

type CategoryHandler struct {
	service service.CategoryService
}

func NewCategoryHandler(service service.CategoryService) *CategoryHandler {
	return &CategoryHandler{service: service}
}

// CreateCategory godoc
//
//	@Summary		Cria uma nova categoria
//	@Description	Cadastra uma nova categoria de produtos (somente admin)
//	@Tags			categories
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			category	body		request.CreateCategoryRequestDTO	true	"Dados da categoria"
//	@Success		201			{object}	map[string]string
//	@Failure		400			{object}	utils.ErrorResponse
//	@Failure		403			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/categories [post]
func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateCategoryRequestDTO

	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.service.CreateCategory(dto); err != nil {
		utils.HandleErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.JsonResponse(
		w,
		http.StatusCreated,
		map[string]string{"message": "Category created successfully"},
	)
}

// GetAllCategories godoc
//
//	@Summary		Lista todas as categorias
//	@Description	Retorna todas as categorias cadastradas
//	@Tags			categories
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		domain.Category
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/categories [get]
func (h *CategoryHandler) GetAllCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.GetAllCategories()
	if err != nil {
		utils.HandleErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.JsonResponse(w, http.StatusOK, categories)
}

// GetCategoryByID godoc
//
//	@Summary		Busca uma categoria pelo ID
//	@Description	Retorna os dados de uma categoria a partir do seu UUID
//	@Tags			categories
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"ID da categoria (UUID)"
//	@Success		200	{object}	domain.Category
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/categories/{id} [get]
func (h *CategoryHandler) GetCategoryByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, err := uuid.Parse(id); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	category, err := h.service.GetCategoryByID(id)
	if err != nil {
		utils.HandleErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	if category == nil {
		utils.HandleErrorResponse(w, http.StatusNotFound, "Category not found")
		return
	}

	utils.JsonResponse(w, http.StatusOK, category)
}

// UpdateCategory godoc
//
//	@Summary		Atualiza uma categoria existente
//	@Description	Atualiza os dados de uma categoria a partir do seu UUID (somente admin)
//	@Tags			categories
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		string								true	"ID da categoria (UUID)"
//	@Param			category	body		request.UpdateCategoryRequestDTO	true	"Dados da categoria"
//	@Success		200			{object}	map[string]string
//	@Failure		400			{object}	utils.ErrorResponse
//	@Failure		403			{object}	utils.ErrorResponse
//	@Failure		404			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, err := uuid.Parse(id); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	var dto request.UpdateCategoryRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.service.UpdateCategory(id, dto); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HandleErrorResponse(w, http.StatusNotFound, "Category not found")
			return
		}

		utils.HandleErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.JsonResponse(
		w,
		http.StatusOK,
		map[string]string{"message": "Category updated successfully"},
	)
}

// DeleteCategory godoc
//
//	@Summary		Remove uma categoria
//	@Description	Remove uma categoria a partir do seu UUID (somente admin)
//	@Tags			categories
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"ID da categoria (UUID)"
//	@Success		204
//	@Failure		403	{object}	utils.ErrorResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, err := uuid.Parse(id); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.DeleteCategory(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HandleErrorResponse(w, http.StatusNotFound, "Category not found")
			return
		}

		utils.HandleErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
