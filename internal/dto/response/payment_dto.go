package response

import "github.com/google/uuid"

type PaymentIntentResponse struct {
	PaymentID    uuid.UUID `json:"payment_id"`
	OrderID      uuid.UUID `json:"order_id"`
	ClientSecret string    `json:"client_secret"`
	Amount       int64     `json:"amount"`
	Currency     string    `json:"currency"`
	Status       string    `json:"status"`
}
