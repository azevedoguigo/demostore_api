package handler

import (
	"encoding/json"
	"errors"
	"io"
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

const maxWebhookBodyBytes = 64 * 1024

type PaymentHandler struct {
	service service.PaymentService
}

func NewPaymentHandler(service service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

// CreatePaymentIntent godoc
//
//	@Summary		Inicia o pagamento de um pedido
//	@Description	Cria (ou recupera) o PaymentIntent do Stripe para um pedido pendente e retorna o client_secret para o frontend confirmar o pagamento
//	@Tags			payments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"ID do pedido (UUID)"
//	@Success		200	{object}	response.PaymentIntentResponse
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Failure		409	{object}	utils.ErrorResponse
//	@Failure		502	{object}	utils.ErrorResponse
//	@Router			/orders/{id}/payment [post]
func (h *PaymentHandler) CreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.UserIDFromContext(r.Context())
	if err != nil {
		utils.HandleErrorResponse(w, http.StatusUnauthorized, "Invalid token")
		return
	}

	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid order id")
		return
	}

	resp, err := h.service.CreatePaymentIntent(userID, id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			utils.HandleErrorResponse(w, http.StatusNotFound, err.Error())
		case errors.Is(err, service.ErrOrderNotPayable), errors.Is(err, service.ErrOrderExpired):
			utils.HandleErrorResponse(w, http.StatusConflict, err.Error())
		case errors.Is(err, service.ErrPaymentProvider):
			log.Printf("stripe: %v", err)
			utils.HandleErrorResponse(w, http.StatusBadGateway, "Payment provider unavailable")
		default:
			utils.HandleErrorResponse(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	utils.JsonResponse(w, http.StatusOK, resp)
}

// StripeWebhook godoc
//
//	@Summary		Webhook do Stripe
//	@Description	Recebe eventos do Stripe (assinatura verificada pelo header Stripe-Signature) e atualiza pagamentos e pedidos
//	@Tags			payments
//	@Accept			json
//	@Produce		json
//	@Param			Stripe-Signature	header		string	true	"Assinatura do evento"
//	@Success		200					{object}	map[string]bool
//	@Failure		400					{object}	utils.ErrorResponse
//	@Failure		500					{object}	utils.ErrorResponse
//	@Router			/webhooks/stripe [post]
func (h *PaymentHandler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes))
	if err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.service.HandleWebhook(payload, r.Header.Get("Stripe-Signature")); err != nil {
		if errors.Is(err, service.ErrInvalidWebhookSignature) {
			utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid signature")
			return
		}

		// A 5xx makes Stripe retry the delivery later.
		log.Printf("stripe webhook: %v", err)
		utils.HandleErrorResponse(w, http.StatusInternalServerError, "Failed to process event")
		return
	}

	utils.JsonResponse(w, http.StatusOK, map[string]bool{"received": true})
}

func (h *PaymentHandler) handleRefundError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrOrderNotFound):
		utils.HandleErrorResponse(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrInvalidRefundItems):
		utils.HandleErrorResponse(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrOrderNotRefundable), errors.Is(err, service.ErrRefundQuantityExceeded),
		errors.Is(err, service.ErrRefundNotSupported):
		utils.HandleErrorResponse(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrPaymentProvider):
		log.Printf("stripe: %v", err)
		utils.HandleErrorResponse(w, http.StatusBadGateway, "Payment provider unavailable")
	default:
		utils.HandleErrorResponse(w, http.StatusInternalServerError, err.Error())
	}
}

// CreateRefund godoc
//
//	@Summary		Estorna itens de um pedido
//	@Description	Estorna parcialmente um pedido pago pelos produtos e quantidades informados. O valor vem do preço registrado no pedido e os itens voltam ao estoque quando o estorno é concluído. Pagamentos com boleto não podem ser estornados pelo Stripe. Somente admin
//	@Tags			admin-orders
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string							true	"ID do pedido (UUID)"
//	@Param			refund	body		request.CreateRefundRequestDTO	true	"Itens a estornar"
//	@Success		201		{object}	domain.Refund
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		403		{object}	utils.ErrorResponse
//	@Failure		404		{object}	utils.ErrorResponse
//	@Failure		409		{object}	utils.ErrorResponse
//	@Failure		502		{object}	utils.ErrorResponse
//	@Router			/admin/orders/{id}/refunds [post]
func (h *PaymentHandler) CreateRefund(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid order id")
		return
	}

	var dto request.CreateRefundRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	refund, err := h.service.RefundItems(id, dto)
	if err != nil {
		h.handleRefundError(w, err)
		return
	}

	utils.JsonResponse(w, http.StatusCreated, refund)
}

// GetOrderRefunds godoc
//
//	@Summary		Lista os estornos de um pedido
//	@Description	Retorna os estornos de um pedido com seus itens e status (somente admin)
//	@Tags			admin-orders
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"ID do pedido (UUID)"
//	@Success		200	{array}		domain.Refund
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		403	{object}	utils.ErrorResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/admin/orders/{id}/refunds [get]
func (h *PaymentHandler) GetOrderRefunds(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid order id")
		return
	}

	refunds, err := h.service.GetOrderRefunds(id)
	if err != nil {
		h.handleRefundError(w, err)
		return
	}

	if refunds == nil {
		refunds = []domain.Refund{}
	}

	utils.JsonResponse(w, http.StatusOK, refunds)
}
