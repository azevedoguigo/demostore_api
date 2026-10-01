package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/response"
	"github.com/azevedoguigo/demostore_api.git/internal/middleware"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	"github.com/azevedoguigo/demostore_api.git/pkg/utils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CartHandler struct {
	service service.CartService
}

func NewCartHandler(service service.CartService) *CartHandler {
	return &CartHandler{service: service}
}

func (h *CartHandler) userID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		utils.HandleErrorResponse(w, http.StatusUnauthorized, "Invalid token")
		return uuid.Nil, false
	}

	return userID, true
}

func (h *CartHandler) respondCart(w http.ResponseWriter, cart *domain.Cart, err error) {
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.JsonResponse(w, http.StatusOK, response.NewCartResponse(cart))
}

func (h *CartHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrProductNotFound), errors.Is(err, service.ErrCartItemNotFound):
		utils.HandleErrorResponse(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrInvalidQuantity):
		utils.HandleErrorResponse(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrInsufficientStock):
		utils.HandleErrorResponse(w, http.StatusConflict, err.Error())
	default:
		utils.HandleErrorResponse(w, http.StatusInternalServerError, err.Error())
	}
}

// GetCart godoc
//
//	@Summary		Retorna o carrinho do usuário autenticado
//	@Description	Retorna o carrinho atual com itens, subtotais e total (cria um carrinho vazio se não existir)
//	@Tags			cart
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.CartResponse
//	@Failure		401	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/cart [get]
func (h *CartHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	cart, err := h.service.GetCart(userID)
	h.respondCart(w, cart, err)
}

// AddItem godoc
//
//	@Summary		Adiciona um produto ao carrinho
//	@Description	Adiciona um produto ao carrinho; se já existir, soma a quantidade. Respeita o estoque disponível
//	@Tags			cart
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			item	body		request.AddCartItemRequestDTO	true	"Produto e quantidade"
//	@Success		200		{object}	response.CartResponse
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		404		{object}	utils.ErrorResponse
//	@Failure		409		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/cart/items [post]
func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	var dto request.AddCartItemRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if _, err := uuid.Parse(dto.ProductID); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid product_id")
		return
	}

	cart, err := h.service.AddItem(userID, dto)
	h.respondCart(w, cart, err)
}

// UpdateItem godoc
//
//	@Summary		Altera a quantidade de um item do carrinho
//	@Description	Define a nova quantidade de um produto já presente no carrinho
//	@Tags			cart
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			product_id	path		string							true	"ID do produto (UUID)"
//	@Param			item		body		request.UpdateCartItemRequestDTO	true	"Nova quantidade"
//	@Success		200			{object}	response.CartResponse
//	@Failure		400			{object}	utils.ErrorResponse
//	@Failure		404			{object}	utils.ErrorResponse
//	@Failure		409			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/cart/items/{product_id} [put]
func (h *CartHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	productID := chi.URLParam(r, "product_id")
	if _, err := uuid.Parse(productID); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid product_id")
		return
	}

	var dto request.UpdateCartItemRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	cart, err := h.service.UpdateItem(userID, productID, dto)
	h.respondCart(w, cart, err)
}

// RemoveItem godoc
//
//	@Summary		Remove um item do carrinho
//	@Description	Remove um produto do carrinho
//	@Tags			cart
//	@Produce		json
//	@Security		BearerAuth
//	@Param			product_id	path		string	true	"ID do produto (UUID)"
//	@Success		200			{object}	response.CartResponse
//	@Failure		400			{object}	utils.ErrorResponse
//	@Failure		404			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/cart/items/{product_id} [delete]
func (h *CartHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	productID := chi.URLParam(r, "product_id")
	if _, err := uuid.Parse(productID); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid product_id")
		return
	}

	cart, err := h.service.RemoveItem(userID, productID)
	h.respondCart(w, cart, err)
}

// ClearCart godoc
//
//	@Summary		Esvazia o carrinho
//	@Description	Remove todos os itens do carrinho do usuário autenticado
//	@Tags			cart
//	@Produce		json
//	@Security		BearerAuth
//	@Success		204
//	@Failure		401	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/cart [delete]
func (h *CartHandler) ClearCart(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	if err := h.service.ClearCart(userID); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
