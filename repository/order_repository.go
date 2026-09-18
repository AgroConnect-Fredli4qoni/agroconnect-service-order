package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/agroconnect/service-order/models"
)

type StockDeductorFunc func(ctx context.Context, productID string, quantity int) error

type OrderRepository interface {
	CreateOrderACID(ctx context.Context, order *models.Order, items []models.OrderItem, deductor StockDeductorFunc) error
	FindByUser(ctx context.Context, userID int) ([]models.Order, error)
	FindByCode(ctx context.Context, code string) (*models.Order, error)
}

type mysqlOrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) OrderRepository {
	return &mysqlOrderRepository{db: db}
}

func (r *mysqlOrderRepository) CreateOrderACID(ctx context.Context, order *models.Order, items []models.OrderItem, deductor StockDeductorFunc) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	orderQuery := `INSERT INTO orders (order_code, user_id, total_amount, status, shipping_address) VALUES (?, ?, ?, ?, ?)`
	res, err := tx.ExecContext(ctx, orderQuery, order.OrderCode, order.UserID, order.TotalAmount, order.Status, order.ShippingAddress)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	orderID, err := res.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	order.ID = int(orderID)

	itemQuery := `INSERT INTO order_items (order_id, product_id, product_name, price, quantity, subtotal) VALUES (?, ?, ?, ?, ?, ?)`
	stmt, err := tx.PrepareContext(ctx, itemQuery)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for i := range items {
		items[i].OrderID = order.ID
		_, err := stmt.ExecContext(ctx, items[i].OrderID, items[i].ProductID, items[i].ProductName, items[i].Price, items[i].Quantity, items[i].Subtotal)
		if err != nil {
			_ = tx.Rollback()
			return err
		}

		if deductor != nil {
			if err := deductor(ctx, items[i].ProductID, items[i].Quantity); err != nil {
				_ = tx.Rollback()
				return errors.New("failed to synchronize stock with catalog service: " + err.Error())
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	order.Items = items
	return nil
}

func (r *mysqlOrderRepository) FindByUser(ctx context.Context, userID int) ([]models.Order, error) {
	query := `SELECT id, order_code, user_id, total_amount, status, shipping_address, created_at FROM orders WHERE user_id = ? ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.OrderCode, &o.UserID, &o.TotalAmount, &o.Status, &o.ShippingAddress, &o.CreatedAt); err != nil {
			return nil, err
		}

		itemRows, err := r.db.QueryContext(ctx, `SELECT id, order_id, product_id, product_name, price, quantity, subtotal FROM order_items WHERE order_id = ?`, o.ID)
		if err == nil {
			for itemRows.Next() {
				var it models.OrderItem
				if err := itemRows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.ProductName, &it.Price, &it.Quantity, &it.Subtotal); err == nil {
					o.Items = append(o.Items, it)
				}
			}
			itemRows.Close()
		}

		orders = append(orders, o)
	}

	if orders == nil {
		orders = []models.Order{}
	}

	return orders, nil
}

func (r *mysqlOrderRepository) FindByCode(ctx context.Context, code string) (*models.Order, error) {
	query := `SELECT id, order_code, user_id, total_amount, status, shipping_address, created_at FROM orders WHERE order_code = ?`
	row := r.db.QueryRowContext(ctx, query, code)

	var o models.Order
	if err := row.Scan(&o.ID, &o.OrderCode, &o.UserID, &o.TotalAmount, &o.Status, &o.ShippingAddress, &o.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}

	itemRows, err := r.db.QueryContext(ctx, `SELECT id, order_id, product_id, product_name, price, quantity, subtotal FROM order_items WHERE order_id = ?`, o.ID)
	if err == nil {
		for itemRows.Next() {
			var it models.OrderItem
			if err := itemRows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.ProductName, &it.Price, &it.Quantity, &it.Subtotal); err == nil {
				o.Items = append(o.Items, it)
			}
		}
		itemRows.Close()
	}

	return &o, nil
}
