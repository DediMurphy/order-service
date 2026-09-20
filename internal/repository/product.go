package repository

import (
	"context"
	"database/sql"
	"errors"

	"order-service/internal/domain"
)

type ProductRepository struct{}

func NewProductRepository() *ProductRepository {
	return &ProductRepository{}
}

func (r *ProductRepository) Create(ctx context.Context, q Querier, p *domain.Product) error {
	res, err := q.ExecContext(ctx,
		`INSERT INTO products (sku, name, price, stock) VALUES (?, ?, ?, ?)`,
		p.SKU, p.Name, p.Price, p.Stock)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = id
	return r.reload(ctx, q, p)
}

func (r *ProductRepository) reload(ctx context.Context, q Querier, p *domain.Product) error {
	got, err := r.FindByID(ctx, q, p.ID)
	if err != nil {
		return err
	}
	*p = *got
	return nil
}

func (r *ProductRepository) FindByID(ctx context.Context, q Querier, id int64) (*domain.Product, error) {
	var p domain.Product
	err := q.QueryRowContext(ctx,
		`SELECT id, sku, name, price, stock, created_at FROM products WHERE id = ?`, id).
		Scan(&p.ID, &p.SKU, &p.Name, &p.Price, &p.Stock, &p.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepository) List(ctx context.Context, q Querier, search string, limit, offset int) ([]domain.Product, error) {
	query := `SELECT id, sku, name, price, stock, created_at FROM products`
	args := []any{}

	if search != "" {
		query += ` WHERE name LIKE ?`
		args = append(args, "%"+search+"%")
	}
	query += ` ORDER BY id LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() 

	products := []domain.Product{}
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Price, &p.Stock, &p.CreatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *ProductRepository) Count(ctx context.Context, q Querier, search string) (int, error) {
	query := `SELECT COUNT(*) FROM products`
	args := []any{}
	if search != "" {
		query += ` WHERE name LIKE ?`
		args = append(args, "%"+search+"%")
	}
	var total int
	err := q.QueryRowContext(ctx, query, args...).Scan(&total)
	return total, err
}

func (r *ProductRepository) Update(ctx context.Context, q Querier, p *domain.Product) error {
	_, err := q.ExecContext(ctx,
		`UPDATE products SET name = ?, price = ?, stock = ? WHERE id = ?`,
		p.Name, p.Price, p.Stock, p.ID)
	if err != nil {
		return err
	}
	return r.reload(ctx, q, p)
}

func (r *ProductRepository) DecreaseStock(ctx context.Context, q Querier, productID int64, qty int) (bool, error) {
	res, err := q.ExecContext(ctx,
		`UPDATE products SET stock = stock - ? WHERE id = ? AND stock >= ?`,
		qty, productID, qty)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *ProductRepository) IncreaseStock(ctx context.Context, q Querier, productID int64, qty int) error {
	_, err := q.ExecContext(ctx,
		`UPDATE products SET stock = stock + ? WHERE id = ?`, qty, productID)
	return err
}
