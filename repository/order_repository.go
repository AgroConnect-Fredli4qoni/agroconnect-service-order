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
	GetSalesStats(ctx context.Context, userID int, role string) (*models.OrderStatsDTO, error)
	UpdateOrderStatus(ctx context.Context, code string, status string) error
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

	itemQuery := `INSERT INTO order_items (order_id, product_id, product_name, price, quantity, subtotal, farmer_id) VALUES (?, ?, ?, ?, ?, ?, ?)`
	stmt, err := tx.PrepareContext(ctx, itemQuery)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for i := range items {
		items[i].OrderID = order.ID
		_, err := stmt.ExecContext(ctx, items[i].OrderID, items[i].ProductID, items[i].ProductName, items[i].Price, items[i].Quantity, items[i].Subtotal, items[i].FarmerID)
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

		itemRows, err := r.db.QueryContext(ctx, `SELECT id, order_id, product_id, product_name, price, quantity, subtotal, COALESCE(farmer_id, 0) FROM order_items WHERE order_id = ?`, o.ID)
		if err == nil {
			for itemRows.Next() {
				var it models.OrderItem
				if err := itemRows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.ProductName, &it.Price, &it.Quantity, &it.Subtotal, &it.FarmerID); err == nil {
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

	itemRows, err := r.db.QueryContext(ctx, `SELECT id, order_id, product_id, product_name, price, quantity, subtotal, COALESCE(farmer_id, 0) FROM order_items WHERE order_id = ?`, o.ID)
	if err == nil {
		for itemRows.Next() {
			var it models.OrderItem
			if err := itemRows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.ProductName, &it.Price, &it.Quantity, &it.Subtotal, &it.FarmerID); err == nil {
				o.Items = append(o.Items, it)
			}
		}
		itemRows.Close()
	}

	return &o, nil
}

func (r *mysqlOrderRepository) GetSalesStats(ctx context.Context, userID int, role string) (*models.OrderStatsDTO, error) {
	stats := &models.OrderStatsDTO{
		StatusBreakdown: []models.StatusCountDTO{},
		TopProducts:     []models.TopProductDTO{},
		RecentOrders:    []models.Order{},
	}

	var revenueQuery string
	var revenueArgs []interface{}
	var itemsQuery string
	var itemsArgs []interface{}
	var statusQuery string
	var statusArgs []interface{}
	var topProductsQuery string
	var topProductsArgs []interface{}
	var recentOrdersQuery string
	var recentOrdersArgs []interface{}

	if role == "farmer" && userID > 0 {
		revenueQuery = `SELECT COALESCE(SUM(oi.subtotal), 0), COUNT(DISTINCT o.id), COALESCE(AVG(oi.subtotal), 0) FROM order_items oi JOIN orders o ON oi.order_id = o.id WHERE oi.farmer_id = ? AND o.status != 'CANCELLED'`
		revenueArgs = []interface{}{userID}

		itemsQuery = `SELECT COALESCE(SUM(oi.quantity), 0) FROM order_items oi JOIN orders o ON oi.order_id = o.id WHERE oi.farmer_id = ? AND o.status != 'CANCELLED'`
		itemsArgs = []interface{}{userID}

		statusQuery = `SELECT o.status, COUNT(DISTINCT o.id) FROM orders o JOIN order_items oi ON o.id = oi.order_id WHERE oi.farmer_id = ? GROUP BY o.status`
		statusArgs = []interface{}{userID}

		topProductsQuery = `SELECT oi.product_id, oi.product_name, SUM(oi.quantity) as total_qty, SUM(oi.subtotal) as total_rev FROM order_items oi JOIN orders o ON oi.order_id = o.id WHERE oi.farmer_id = ? AND o.status != 'CANCELLED' GROUP BY oi.product_id, oi.product_name ORDER BY total_qty DESC LIMIT 5`
		topProductsArgs = []interface{}{userID}

		recentOrdersQuery = `SELECT DISTINCT o.id, o.order_code, o.user_id, o.total_amount, o.status, o.shipping_address, o.created_at FROM orders o JOIN order_items oi ON o.id = oi.order_id WHERE oi.farmer_id = ? ORDER BY o.created_at DESC LIMIT 10`
		recentOrdersArgs = []interface{}{userID}
	} else if role == "buyer" && userID > 0 {
		revenueQuery = `SELECT COALESCE(SUM(total_amount), 0), COUNT(*), COALESCE(AVG(total_amount), 0) FROM orders WHERE user_id = ? AND status != 'CANCELLED'`
		revenueArgs = []interface{}{userID}

		itemsQuery = `SELECT COALESCE(SUM(oi.quantity), 0) FROM order_items oi JOIN orders o ON oi.order_id = o.id WHERE o.user_id = ? AND o.status != 'CANCELLED'`
		itemsArgs = []interface{}{userID}

		statusQuery = `SELECT status, COUNT(*) FROM orders WHERE user_id = ? GROUP BY status`
		statusArgs = []interface{}{userID}

		topProductsQuery = `SELECT oi.product_id, oi.product_name, SUM(oi.quantity) as total_qty, SUM(oi.subtotal) as total_rev FROM order_items oi JOIN orders o ON oi.order_id = o.id WHERE o.user_id = ? AND o.status != 'CANCELLED' GROUP BY oi.product_id, oi.product_name ORDER BY total_qty DESC LIMIT 5`
		topProductsArgs = []interface{}{userID}

		recentOrdersQuery = `SELECT id, order_code, user_id, total_amount, status, shipping_address, created_at FROM orders WHERE user_id = ? ORDER BY created_at DESC LIMIT 10`
		recentOrdersArgs = []interface{}{userID}
	} else {
		revenueQuery = `SELECT COALESCE(SUM(total_amount), 0), COUNT(*), COALESCE(AVG(total_amount), 0) FROM orders WHERE status != 'CANCELLED'`
		itemsQuery = `SELECT COALESCE(SUM(quantity), 0) FROM order_items oi JOIN orders o ON oi.order_id = o.id WHERE o.status != 'CANCELLED'`
		statusQuery = `SELECT status, COUNT(*) FROM orders GROUP BY status`
		topProductsQuery = `SELECT oi.product_id, oi.product_name, SUM(oi.quantity) as total_qty, SUM(oi.subtotal) as total_rev FROM order_items oi JOIN orders o ON oi.order_id = o.id WHERE o.status != 'CANCELLED' GROUP BY oi.product_id, oi.product_name ORDER BY total_qty DESC LIMIT 5`
		recentOrdersQuery = `SELECT id, order_code, user_id, total_amount, status, shipping_address, created_at FROM orders ORDER BY created_at DESC LIMIT 10`
	}

	row := r.db.QueryRowContext(ctx, revenueQuery, revenueArgs...)
	if err := row.Scan(&stats.TotalRevenue, &stats.TotalOrders, &stats.AverageOrderValue); err != nil {
		return nil, err
	}
	if stats.TotalOrders > 0 {
		stats.AverageOrderValue = stats.TotalRevenue / float64(stats.TotalOrders)
	}

	itemsRow := r.db.QueryRowContext(ctx, itemsQuery, itemsArgs...)
	if err := itemsRow.Scan(&stats.TotalItemsSold); err != nil {
		return nil, err
	}

	statusRows, err := r.db.QueryContext(ctx, statusQuery, statusArgs...)
	if err == nil {
		defer statusRows.Close()
		for statusRows.Next() {
			var sc models.StatusCountDTO
			if err := statusRows.Scan(&sc.Status, &sc.Count); err == nil {
				if stats.TotalOrders > 0 {
					sc.Percentage = float64(int((float64(sc.Count)/float64(stats.TotalOrders))*1000)) / 10.0
				}
				stats.StatusBreakdown = append(stats.StatusBreakdown, sc)
			}
		}
	}

	topRows, err := r.db.QueryContext(ctx, topProductsQuery, topProductsArgs...)
	if err == nil {
		defer topRows.Close()
		for topRows.Next() {
			var tp models.TopProductDTO
			if err := topRows.Scan(&tp.ProductID, &tp.ProductName, &tp.TotalQuantity, &tp.TotalRevenue); err == nil {
				stats.TopProducts = append(stats.TopProducts, tp)
			}
		}
	}

	recentRows, err := r.db.QueryContext(ctx, recentOrdersQuery, recentOrdersArgs...)
	if err == nil {
		defer recentRows.Close()
		for recentRows.Next() {
			var o models.Order
			if err := recentRows.Scan(&o.ID, &o.OrderCode, &o.UserID, &o.TotalAmount, &o.Status, &o.ShippingAddress, &o.CreatedAt); err == nil {
				var itemQuery string
				var itemArgs []interface{}
				if role == "farmer" && userID > 0 {
					itemQuery = `SELECT id, order_id, product_id, product_name, price, quantity, subtotal, COALESCE(farmer_id, 0) FROM order_items WHERE order_id = ? AND farmer_id = ?`
					itemArgs = []interface{}{o.ID, userID}
				} else {
					itemQuery = `SELECT id, order_id, product_id, product_name, price, quantity, subtotal, COALESCE(farmer_id, 0) FROM order_items WHERE order_id = ?`
					itemArgs = []interface{}{o.ID}
				}

				itemRows, err := r.db.QueryContext(ctx, itemQuery, itemArgs...)
				if err == nil {
					for itemRows.Next() {
						var it models.OrderItem
						if err := itemRows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.ProductName, &it.Price, &it.Quantity, &it.Subtotal, &it.FarmerID); err == nil {
							o.Items = append(o.Items, it)
						}
					}
					itemRows.Close()
				}

				if role == "farmer" && userID > 0 {
					var farmerTotal float64
					for _, it := range o.Items {
						farmerTotal += it.Subtotal
					}
					o.TotalAmount = farmerTotal
				}

				stats.RecentOrders = append(stats.RecentOrders, o)
			}
		}
	}

	return stats, nil
}

func (r *mysqlOrderRepository) UpdateOrderStatus(ctx context.Context, code string, status string) error {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders WHERE order_code = ?`, code).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("order not found")
	}

	_, err = r.db.ExecContext(ctx, `UPDATE orders SET status = ? WHERE order_code = ?`, status, code)
	return err
}
