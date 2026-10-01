package request

type RefundItemRequestDTO struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type CreateRefundRequestDTO struct {
	Items  []RefundItemRequestDTO `json:"items"`
	Reason string                 `json:"reason"`
}
