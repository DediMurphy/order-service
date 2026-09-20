package domain

type OrderStatus string

const (
	StatusPending   OrderStatus = "PENDING"
	StatusPaid      OrderStatus = "PAID"
	StatusShipped   OrderStatus = "SHIPPED"
	StatusCompleted OrderStatus = "COMPLETED"
	StatusCancelled OrderStatus = "CANCELLED"
)

var allowedTransitions = map[OrderStatus][]OrderStatus{
	StatusPending:   {StatusPaid, StatusCancelled},
	StatusPaid:      {StatusShipped, StatusCancelled},
	StatusShipped:   {StatusCompleted},
	StatusCompleted: {},
	StatusCancelled: {},
}


func IsValidStatus(s OrderStatus) bool {
	_, ok := allowedTransitions[s]
	return ok
}

func (s OrderStatus) CanTransitionTo(target OrderStatus) bool {
	for _, allowed := range allowedTransitions[s] {
		if allowed == target {
			return true
		}
	}
	return false
}

type Order struct {
	ID          int64       `json:"id"`
	CustomerID  int64       `json:"customer_id"`
	Status      OrderStatus `json:"status"`
	TotalAmount int64       `json:"total_amount"`
	CreatedAt   string      `json:"created_at"`
	Items []OrderItem `json:"items,omitempty"`
}

type OrderItem struct {
	ID           int64  `json:"id"`
	ProductID    int64  `json:"product_id"`
	ProductName  string `json:"product_name"`
	Qty          int    `json:"qty"`
	PriceAtOrder int64  `json:"price_at_order"`
	Subtotal     int64  `json:"subtotal"`
}

type OrderItemInput struct {
	ProductID int64 `json:"product_id"`
	Qty       int   `json:"qty"`
}
