package handler

import (
	"encoding/json"
	"net/http"

	"github.com/azevedoguigo/demostore_api.git/internal/dto/request"
	"github.com/azevedoguigo/demostore_api.git/internal/service"
	"github.com/azevedoguigo/demostore_api.git/pkg/utils"
)

type ProductHandler struct {
	service service.ProductService
}

func NewProductHandler(service service.ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

// CreateProduct godoc
//
//	@Summary		Cria um novo produto
//	@Description	Cadastra um novo produto na plataforma
//	@Tags			products
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			product	body		request.CreateProductRequestDTO	true	"Dados do produto"
//	@Success		201		{object}	map[string]string
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		403		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/products [post]
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var dto request.CreateProductRequestDTO

	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.service.CreateProduct(dto); err != nil {
		utils.HandleErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.JsonResponse(
		w,
		http.StatusCreated,
		map[string]string{"message": "Product created successfully"},
	)
}

// GetAllProducts godoc
//
//	@Summary		Lista todos os produtos
//	@Description	Retorna todos os produtos cadastrados
//	@Tags			products
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{array}		domain.Product
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/products [get]
func (h *ProductHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.service.GetAllProducts()
	if err != nil {
		utils.HandleErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.JsonResponse(w, http.StatusOK, products)
}

// GetProductByID godoc
//
//	@Summary		Busca um produto pelo ID
//	@Description	Retorna os dados de um produto a partir do seu UUID
//	@Tags			products
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"ID do produto (UUID)"
//	@Success		200	{object}	domain.Product
//	@Failure		400	{object}	utils.ErrorResponse
//	@Failure		404	{object}	utils.ErrorResponse
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/products/{id} [get]
func (h *ProductHandler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		utils.HandleErrorResponse(w, http.StatusBadRequest, "Missing product ID")
		return
	}

	product, err := h.service.GetProductByID(id)
	if err != nil {
		utils.HandleErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	if product == nil {
		utils.HandleErrorResponse(w, http.StatusNotFound, "Product not found")
		return
	}

	utils.JsonResponse(w, http.StatusOK, product)
}
