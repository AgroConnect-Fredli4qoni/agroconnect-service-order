package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/agroconnect/service-order/models"
)

type AdminRepository interface {
	GetPlatformAdminStats(ctx context.Context) (*models.PlatformAdminStatsDTO, error)
	FindAllUsers(ctx context.Context, roleFilter, search string) ([]models.AdminUserDTO, error)
	FindAllOrdersAdmin(ctx context.Context, statusFilter, search string) ([]models.Order, error)
	FindAllWithdrawalsAdmin(ctx context.Context, statusFilter string) ([]models.AdminWithdrawalItemDTO, error)
	UpdateWithdrawalStatus(ctx context.Context, id int, status, notes string) error
}

type mysqlAdminRepository struct {
	db *sql.DB
}

func NewMySQLAdminRepository(db *sql.DB) AdminRepository {
	return &mysqlAdminRepository{db: db}
}

func (r *mysqlAdminRepository) GetPlatformAdminStats(ctx context.Context) (*models.PlatformAdminStatsDTO, error) {
	stats := &models.PlatformAdminStatsDTO{
		StatusBreakdown: []models.StatusCountDTO{},
		TopProducts:     []models.TopProductDTO{},
		RecentOrders:    []models.Order{},
	}

	revenueQuery := `SELECT COALESCE(SUM(total_amount), 0), COUNT(*), COALESCE(AVG(total_amount), 0) FROM orders WHERE status != 'CANCELLED'`
	row := r.db.QueryRowContext(ctx, revenueQuery)
	if err := row.Scan(&stats.TotalGMV, &stats.TotalOrders, &stats.AverageOrderValue); err != nil {
		return nil, err
	}
	if stats.TotalOrders > 0 {
		stats.AverageOrderValue = stats.TotalGMV / float64(stats.TotalOrders)
	}

	itemsQuery := `SELECT COALESCE(SUM(oi.quantity), 0) FROM order_items oi JOIN orders o ON oi.order_id = o.id WHERE o.status != 'CANCELLED'`
	itemsRow := r.db.QueryRowContext(ctx, itemsQuery)
	if err := itemsRow.Scan(&stats.TotalItemsSold); err != nil {
		return nil, err
	}

	farmerCountQuery := `SELECT COUNT(*) FROM users WHERE role = 'farmer'`
	if err := r.db.QueryRowContext(ctx, farmerCountQuery).Scan(&stats.TotalFarmers); err != nil {
		return nil, err
	}

	buyerCountQuery := `SELECT COUNT(*) FROM users WHERE role = 'buyer'`
	if err := r.db.QueryRowContext(ctx, buyerCountQuery).Scan(&stats.TotalBuyers); err != nil {
		return nil, err
	}

	statusQuery := `SELECT status, COUNT(*) FROM orders GROUP BY status`
	statusRows, err := r.db.QueryContext(ctx, statusQuery)
	if err == nil {
		defer statusRows.Close()
		for statusRows.Next() {
			var sc models.StatusCountDTO
			if err := statusRows.Scan(&sc.Status, &sc.Count); err == nil {
				if stats.TotalOrders > 0 {
					sc.Percentage = float64(int((float64(sc.Count)/float64(stats.TotalOrders))*1000)) / 10.0
				}
				stats.StatusBreakdown = append(stats.StatusBreakdown, sc)

				switch strings.ToUpper(sc.Status) {
				case "PENDING":
					stats.PendingOrders = sc.Count
				case "PAID":
					stats.PaidOrders = sc.Count
				case "SHIPPED":
					stats.ShippedOrders = sc.Count
				case "COMPLETED":
					stats.CompletedOrders = sc.Count
				case "CANCELLED":
					stats.CancelledOrders = sc.Count
				}
			}
		}
	}

	wdStatsQuery := `SELECT COUNT(*), COALESCE(SUM(amount), 0) FROM withdrawal_requests WHERE LOWER(status) = 'pending'`
	_ = r.db.QueryRowContext(ctx, wdStatsQuery).Scan(&stats.PendingWithdrawalsCount, &stats.PendingWithdrawalsAmount)

	topProductsQuery := `SELECT oi.product_id, oi.product_name, SUM(oi.quantity) as total_qty, SUM(oi.subtotal) as total_rev FROM order_items oi JOIN orders o ON oi.order_id = o.id WHERE o.status != 'CANCELLED' GROUP BY oi.product_id, oi.product_name ORDER BY total_qty DESC LIMIT 5`
	topRows, err := r.db.QueryContext(ctx, topProductsQuery)
	if err == nil {
		defer topRows.Close()
		for topRows.Next() {
			var tp models.TopProductDTO
			if err := topRows.Scan(&tp.ProductID, &tp.ProductName, &tp.TotalQuantity, &tp.TotalRevenue); err == nil {
				stats.TopProducts = append(stats.TopProducts, tp)
			}
		}
	}

	recentOrdersQuery := `SELECT o.id, o.order_code, o.user_id, COALESCE(u.name, 'Pembeli'), COALESCE(u.email, ''), o.total_amount, o.status, o.shipping_address, o.created_at FROM orders o LEFT JOIN users u ON o.user_id = u.id ORDER BY o.created_at DESC LIMIT 8`
	recentRows, err := r.db.QueryContext(ctx, recentOrdersQuery)
	if err == nil {
		defer recentRows.Close()
		for recentRows.Next() {
			var o models.Order
			if err := recentRows.Scan(&o.ID, &o.OrderCode, &o.UserID, &o.CustomerName, &o.CustomerEmail, &o.TotalAmount, &o.Status, &o.ShippingAddress, &o.CreatedAt); err == nil {
				stats.RecentOrders = append(stats.RecentOrders, o)
			}
		}
	}

	return stats, nil
}

func (r *mysqlAdminRepository) FindAllUsers(ctx context.Context, roleFilter, search string) ([]models.AdminUserDTO, error) {
	query := `SELECT u.id, u.name, u.email, u.role, COALESCE(u.avatar_url, ''), u.created_at, COUNT(DISTINCT o.id) as order_count, COALESCE(SUM(o.total_amount), 0) as total_volume FROM users u LEFT JOIN orders o ON u.id = o.user_id AND o.status != 'CANCELLED' WHERE 1=1`
	var args []interface{}

	if roleFilter != "" && roleFilter != "all" {
		query += ` AND u.role = ?`
		args = append(args, roleFilter)
	}

	if search != "" {
		query += ` AND (u.name LIKE ? OR u.email LIKE ?)`
		searchTerm := "%" + search + "%"
		args = append(args, searchTerm, searchTerm)
	}

	query += ` GROUP BY u.id, u.name, u.email, u.role, u.avatar_url, u.created_at ORDER BY u.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]models.AdminUserDTO, 0)
	for rows.Next() {
		var u models.AdminUserDTO
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.AvatarURL, &u.CreatedAt, &u.OrderCount, &u.TotalVolume); err != nil {
			return nil, err
		}
		u.TotalOrders = u.OrderCount
		u.TotalSpent = u.TotalVolume
		users = append(users, u)
	}

	return users, nil
}

func (r *mysqlAdminRepository) FindAllOrdersAdmin(ctx context.Context, statusFilter, search string) ([]models.Order, error) {
	query := `SELECT o.id, o.order_code, o.user_id, COALESCE(u.name, 'Pembeli'), COALESCE(u.email, ''), o.total_amount, o.status, o.shipping_address, COALESCE(o.snap_token, ''), COALESCE(o.snap_redirect_url, ''), COALESCE(o.payment_type, ''), COALESCE(o.payment_status, ''), o.paid_at, o.created_at FROM orders o LEFT JOIN users u ON o.user_id = u.id WHERE 1=1`
	var args []interface{}

	if statusFilter != "" && statusFilter != "all" {
		query += ` AND o.status = ?`
		args = append(args, strings.ToUpper(statusFilter))
	}

	if search != "" {
		query += ` AND (o.order_code LIKE ? OR u.name LIKE ? OR u.email LIKE ?)`
		searchTerm := "%" + search + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	query += ` ORDER BY o.created_at DESC LIMIT 100`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]models.Order, 0)
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.OrderCode, &o.UserID, &o.CustomerName, &o.CustomerEmail, &o.TotalAmount, &o.Status, &o.ShippingAddress, &o.SnapToken, &o.SnapRedirectURL, &o.PaymentType, &o.PaymentStatus, &o.PaidAt, &o.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}

	for i := range orders {
		itemRows, err := r.db.QueryContext(ctx, `SELECT id, order_id, product_id, product_name, price, quantity, subtotal, COALESCE(farmer_id, 0) FROM order_items WHERE order_id = ?`, orders[i].ID)
		if err == nil {
			for itemRows.Next() {
				var it models.OrderItem
				if err := itemRows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.ProductName, &it.Price, &it.Quantity, &it.Subtotal, &it.FarmerID); err == nil {
					orders[i].Items = append(orders[i].Items, it)
				}
			}
			itemRows.Close()
		}
	}

	return orders, nil
}

func (r *mysqlAdminRepository) FindAllWithdrawalsAdmin(ctx context.Context, statusFilter string) ([]models.AdminWithdrawalItemDTO, error) {
	query := `SELECT w.id, w.farmer_id, COALESCE(u.name, 'Petani'), COALESCE(u.email, ''), w.amount, w.target_type, w.target_provider, w.target_account, w.account_holder, w.status, COALESCE(w.admin_notes, ''), w.created_at FROM withdrawal_requests w LEFT JOIN users u ON w.farmer_id = u.id WHERE 1=1`
	var args []interface{}

	if statusFilter != "" && statusFilter != "all" {
		query += ` AND LOWER(w.status) = ?`
		args = append(args, strings.ToLower(statusFilter))
	}

	query += ` ORDER BY w.created_at DESC LIMIT 100`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	withdrawals := make([]models.AdminWithdrawalItemDTO, 0)
	for rows.Next() {
		var w models.AdminWithdrawalItemDTO
		if err := rows.Scan(&w.ID, &w.FarmerID, &w.FarmerName, &w.FarmerEmail, &w.Amount, &w.TargetType, &w.TargetProvider, &w.TargetAccount, &w.AccountHolder, &w.Status, &w.AdminNotes, &w.CreatedAt); err != nil {
			return nil, err
		}
		w.BankName = w.TargetProvider
		w.AccountNumber = w.TargetAccount
		w.Status = strings.ToUpper(w.Status)
		withdrawals = append(withdrawals, w)
	}

	return withdrawals, nil
}

func (r *mysqlAdminRepository) UpdateWithdrawalStatus(ctx context.Context, id int, status, notes string) error {
	var res sql.Result
	var err error
	if notes != "" {
		res, err = r.db.ExecContext(ctx, `UPDATE withdrawal_requests SET status = ?, admin_notes = ? WHERE id = ?`, status, notes, id)
	} else {
		res, err = r.db.ExecContext(ctx, `UPDATE withdrawal_requests SET status = ? WHERE id = ?`, status, id)
	}
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("withdrawal request not found")
	}
	return nil
}
