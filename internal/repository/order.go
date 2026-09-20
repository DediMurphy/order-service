package repository

import (
	"context"
	"database/sql"
	"errors"

	"order-service/internal/domain"
)

type OrderRepository struct{}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{}
}

func (r *OrderRepository) Create(ctx context.Context, q Querier, o *domain.Order) error {
	res, err := q.ExecContext(ctx,
		`INSERT INTO orders (customer_id, status, total_amount) VALUES (?, ?, ?)`,
		o.CustomerID, string(o.Status), o.TotalAmount)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	o.ID = id
	return q.QueryRowContext(ctx,
		`SELECT created_at FROM orders WHERE id = ?`, id).Scan(&o.CreatedAt)
}

func (r *OrderRepository) CreateItem(ctx context.Context, q Querier, orderID int64, item *domain.OrderItem) error {
	res, err := q.ExecContext(ctx,
		`INSERT INTO order_items (order_id, product_id, qty, price_at_order) VALUES (?, ?, ?, ?)`,
		orderID, item.ProductID, item.Qty, item.PriceAtOrder)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	item.ID = id
	return nil
}

func (r *OrderRepository) FindByID(ctx context.Context, q Querier, id int64) (*domain.Order, error) {
	var o domain.Order
	var status string
	err := q.QueryRowContext(ctx,
		`SELECT id, customer_id, status, total_amount, created_at FROM orders WHERE id = ?`, id).
		Scan(&o.ID, &o.CustomerID, &status, &o.TotalAmount, &o.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	o.Status = domain.OrderStatus(status)
	return &o, nil
}

func (r *OrderRepository) FindItems(ctx context.Context, q Querier, orderID int64) ([]domain.OrderItem, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT oi.id, oi.product_id, p.name, oi.qty, oi.price_at_order
		FROM order_items oi
		JOIN products p ON p.id = oi.product_id
		WHERE oi.order_id = ?
		ORDER BY oi.id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.OrderItem{}
	for rows.Next() {
		var it domain.OrderItem
		if err := rows.Scan(&it.ID, &it.ProductID, &it.ProductName, &it.Qty, &it.PriceAtOrder); err != nil {
			return nil, err
		}
		it.Subtotal = it.PriceAtOrder * int64(it.Qty)
		items = append(items, it)
	}
	return items, rows.Err()
}

func (r *OrderRepository) List(ctx context.Context, q Querier, customerID *int64, status *string) ([]domain.Order, error) {
	query := `SELECT id, customer_id, status, total_amount, created_at FROM orders WHERE 1 = 1`
	args := []any{}

	if customerID != nil {
		query += ` AND customer_id = ?`
		args = append(args, *customerID)
	}
	if status != nil {
		query += ` AND status = ?`
		args = append(args, *status)
	}
	query += ` ORDER BY id`

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []domain.Order{}
	for rows.Next() {
		var o domain.Order
		var st string
		if err := rows.Scan(&o.ID, &o.CustomerID, &st, &o.TotalAmount, &o.CreatedAt); err != nil {
			return nil, err
		}
		o.Status = domain.OrderStatus(st)
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, q Querier, id int64, status domain.OrderStatus) error {
	_, err := q.ExecContext(ctx,
		`UPDATE orders SET status = ? WHERE id = ?`, string(status), id)
	return err
}
