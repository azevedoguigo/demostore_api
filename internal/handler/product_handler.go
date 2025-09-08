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

func (h *ProductHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.service.GetAllProducts()
	if err != nil {
		utils.HandleErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.JsonResponse(w, http.StatusOK, products)
}
