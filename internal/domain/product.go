package domain

type Product struct {
	ID        int64  `json:"id"`
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Price     int64  `json:"price"`
	Stock     int    `json:"stock"`
	CreatedAt string `json:"created_at"`
}
