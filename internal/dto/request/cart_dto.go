package request

type AddCartItemRequestDTO struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type UpdateCartItemRequestDTO struct {
	Quantity int `json:"quantity"`
}
