package response

import (
	"github.com/azevedoguigo/demostore_api.git/internal/domain"
	"github.com/google/uuid"
)

type CartItemResponse struct {
	ProductID uuid.UUID `json:"product_id"`
	Name      string    `json:"name"`
	UnitPrice float64   `json:"unit_price"`
	Quantity  int       `json:"quantity"`
	Subtotal  float64   `json:"subtotal"`
}

type CartResponse struct {
	ID    uuid.UUID          `json:"id"`
	Items []CartItemResponse `json:"items"`
	Total float64            `json:"total"`
}

func NewCartResponse(cart *domain.Cart) CartResponse {
	resp := CartResponse{ID: cart.ID, Items: make([]CartItemResponse, 0, len(cart.Items))}

	for _, item := range cart.Items {
		line := CartItemResponse{ProductID: item.ProductID, Quantity: item.Quantity}
		if item.Product != nil {
			line.Name = item.Product.Name
			line.UnitPrice = item.Product.Price
			line.Subtotal = item.Product.Price * float64(item.Quantity)
		}

		resp.Total += line.Subtotal
		resp.Items = append(resp.Items, line)
	}

	return resp
}
