package service

import (
	"context"
	"database/sql"
	"fmt"

	"order-service/internal/domain"
	"order-service/internal/repository"
)

type OrderService struct {
	db        *sql.DB
	orders    *repository.OrderRepository
	products  *repository.ProductRepository
	customers *repository.CustomerRepository
}

func NewOrderService(
	db *sql.DB,
	orders *repository.OrderRepository,
	products *repository.ProductRepository,
	customers *repository.CustomerRepository,
) *OrderService {
	return &OrderService{db: db, orders: orders, products: products, customers: customers}
}

type CreateOrderInput struct {
	CustomerID int64                   `json:"customer_id"`
	Items      []domain.OrderItemInput `json:"items"`
}

func (s *OrderService) Create(ctx context.Context, in CreateOrderInput) (*domain.Order, error) {
	if in.CustomerID <= 0 {
		return nil, domain.NewBadRequest("validation failed", "customer_id is required")
	}
	if len(in.Items) == 0 {
		return nil, domain.NewBadRequest("validation failed", "items must not be empty")
	}
	for _, it := range in.Items {
		if it.ProductID <= 0 {
			return nil, domain.NewBadRequest("validation failed", "each item needs a valid product_id")
		}
		if it.Qty <= 0 {
			return nil, domain.NewBadRequest("validation failed", "qty must be greater than 0")
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	customer, err := s.customers.FindByID(ctx, tx, in.CustomerID)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, domain.NewNotFound(fmt.Sprintf("customer with id %d not found", in.CustomerID))
	}

	order := &domain.Order{
		CustomerID:  in.CustomerID,
		Status:      domain.StatusPending,
		TotalAmount: 0,
	}

	items := make([]domain.OrderItem, 0, len(in.Items))
	var total int64

	for _, input := range in.Items {
		product, err := s.products.FindByID(ctx, tx, input.ProductID)
		if err != nil {
			return nil, err
		}
		if product == nil {
			return nil, domain.NewNotFound(fmt.Sprintf("product with id %d not found", input.ProductID))
		}

		ok, err := s.products.DecreaseStock(ctx, tx, product.ID, input.Qty)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, domain.NewBadRequest("insufficient stock",
				fmt.Sprintf("product '%s' remaining %d, requested %d",
					product.SKU, product.Stock, input.Qty))
		}

		item := domain.OrderItem{
			ProductID:    product.ID,
			ProductName:  product.Name,
			Qty:          input.Qty,
			PriceAtOrder: product.Price,
			Subtotal:     product.Price * int64(input.Qty),
		}
		total += item.Subtotal
		items = append(items, item)
	}

	order.TotalAmount = total
	if err := s.orders.Create(ctx, tx, order); err != nil {
		return nil, err
	}
	for i := range items {
		if err := s.orders.CreateItem(ctx, tx, order.ID, &items[i]); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	order.Items = items
	return order, nil
}

func (s *OrderService) GetByID(ctx context.Context, id int64) (*domain.Order, error) {
	order, err := s.orders.FindByID(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, domain.NewNotFound(fmt.Sprintf("order with id %d not found", id))
	}
	items, err := s.orders.FindItems(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	order.Items = items
	return order, nil
}

func (s *OrderService) List(ctx context.Context, customerID *int64, status *string) ([]domain.Order, error) {
	if status != nil && !domain.IsValidStatus(domain.OrderStatus(*status)) {
		return nil, domain.NewBadRequest("validation failed",
			fmt.Sprintf("unknown status '%s'", *status))
	}
	return s.orders.List(ctx, s.db, customerID, status)
}

func (s *OrderService) UpdateStatus(ctx context.Context, id int64, target domain.OrderStatus) (*domain.Order, error) {
	if !domain.IsValidStatus(target) {
		return nil, domain.NewBadRequest("validation failed",
			fmt.Sprintf("unknown status '%s'", target))
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	order, err := s.orders.FindByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, domain.NewNotFound(fmt.Sprintf("order with id %d not found", id))
	}

	if !order.Status.CanTransitionTo(target) {
		return nil, domain.NewConflict("invalid status transition",
			fmt.Sprintf("cannot change status from %s to %s", order.Status, target))
	}

	items, err := s.orders.FindItems(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	if target == domain.StatusCancelled {
		for _, it := range items {
			if err := s.products.IncreaseStock(ctx, tx, it.ProductID, it.Qty); err != nil {
				return nil, err
			}
		}
	}

	if err := s.orders.UpdateStatus(ctx, tx, id, target); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	order.Status = target
	order.Items = items
	return order, nil
}
