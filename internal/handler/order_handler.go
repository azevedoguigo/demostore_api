package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/azevedoguigo/demostore_api.git/internal/middleware"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	"github.com/azevedoguigo/demostore_api.git/pkg/utils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type OrderHandler struct {
	service service.OrderService
}

func NewOrderHandler(service service.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

func (h *OrderHandler) userID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		utils.HandleErrorResponse(w, http.StatusUnauthorized, "Invalid token")
		return uuid.Nil, false
	}

	return userID, true
}

func (h *OrderHandler) orderID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid order id")
		return "", false
	}

	return id, true
}

func (h *OrderHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrOrderNotFound), errors.Is(err, service.ErrProductNotFound):
		utils.HandleErrorResponse(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrEmptyCart), errors.Is(err, service.ErrInvalidOrderStatus):
		utils.HandleErrorResponse(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrInsufficientStock), errors.Is(err, service.ErrInvalidStatusTransition),
		errors.Is(err, service.ErrPaymentAlreadyProcessed):
		utils.HandleErrorResponse(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrPaymentProvider):
		log.Printf("stripe: %v", err)
		utils.HandleErrorResponse(w, http.StatusBadGateway, "Payment provider unavailable")
	default:
		utils.HandleErrorResponse(w, http.StatusInternalServerError, err.Error())
	}
}

func (h *OrderHandler) respondOrder(w http.ResponseWriter, status int, order *domain.Order, err error) {
	if err != nil {
		h.handleError(w, err)
		return
	}

	utils.JsonResponse(w, status, order)
}

func (h *OrderHandler) respondOrders(w http.ResponseWriter, orders []domain.Order, err error) {
	if err != nil {
		h.handleError(w, err)
		return
	}

	if orders == nil {
		orders = []domain.Order{}
	}

	utils.JsonResponse(w, http.StatusOK, orders)
}

// Checkout godoc
//
//	@Summary		Finaliza a compra
//	@Description	Cria um pedido a partir do carrinho, baixa o estoque e esvazia o carrinho. Valores em centavos
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Success		201	{object}	domain.Order
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Failure		409	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/orders [post]
func (h *OrderHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	order, err := h.service.Checkout(userID)
	h.respondOrder(w, http.StatusCreated, order, err)
}

// GetMyOrders godoc
//
//	@Summary		Lista os pedidos do usuário autenticado
//	@Description	Retorna os pedidos do usuário, do mais recente para o mais antigo
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		domain.Order
//	@Failure		401	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/orders [get]
func (h *OrderHandler) GetMyOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	orders, err := h.service.GetUserOrders(userID)
	h.respondOrders(w, orders, err)
}

// GetOrder godoc
//
//	@Summary		Busca um pedido pelo ID
//	@Description	Retorna um pedido do usuário autenticado (admins podem ver qualquer pedido)
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"ID do pedido (UUID)"
//	@Success		200	{object}	domain.Order
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/orders/{id} [get]
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	id, ok := h.orderID(w, r)
	if !ok {
		return
	}

	isAdmin := middleware.RoleFromContext(r.Context()) == domain.RoleAdmin
	order, err := h.service.GetOrder(userID, isAdmin, id)
	h.respondOrder(w, http.StatusOK, order, err)
}

// CancelOrder godoc
//
//	@Summary		Cancela um pedido
//	@Description	Cancela um pedido do usuário autenticado enquanto ainda está pendente, cancela o pagamento no Stripe e devolve o estoque
//	@Tags			orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"ID do pedido (UUID)"
//	@Success		200	{object}	domain.Order
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Failure		409	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Failure		502	{object}	utils.ErrorResponse
//	@Router			/orders/{id}/cancel [post]
func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	id, ok := h.orderID(w, r)
	if !ok {
		return
	}

	order, err := h.service.CancelOrder(userID, id)
	h.respondOrder(w, http.StatusOK, order, err)
}

// GetAllOrders godoc
//
//	@Summary		Lista todos os pedidos
//	@Description	Retorna os pedidos de todos os usuários (somente admin)
//	@Tags			admin-orders
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		domain.Order
//	@Failure		403	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/admin/orders [get]
func (h *OrderHandler) GetAllOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.service.GetAllOrders()
	h.respondOrders(w, orders, err)
}

// UpdateOrderStatus godoc
//
//	@Summary		Atualiza o status de um pedido
//	@Description	Avança o status de um pedido (pending → paid → shipped → delivered, ou cancelled). Cancelar um pedido pago estorna o pagamento. Somente admin
//	@Tags			admin-orders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string								true	"ID do pedido (UUID)"
//	@Param			status	body		request.UpdateOrderStatusRequestDTO	true	"Novo status"
//	@Success		200		{object}	domain.Order
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		403		{object}	utils.ErrorResponse
//	@Failure		404		{object}	utils.ErrorResponse
//	@Failure		409		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Failure		502		{object}	utils.ErrorResponse
//	@Router			/admin/orders/{id}/status [patch]
func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := h.orderID(w, r)
	if !ok {
		return
	}

	var dto request.UpdateOrderStatusRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	order, err := h.service.UpdateOrderStatus(id, dto)
	h.respondOrder(w, http.StatusOK, order, err)
}
