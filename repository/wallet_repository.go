package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/agroconnect/service-order/models"
)

type WalletRepository interface {
	EnsureWallet(ctx context.Context, farmerID int) (*models.FarmerWallet, error)
	GetWalletOverview(ctx context.Context, farmerID int) (*models.WalletOverviewDTO, error)
	CreateWithdrawalRequest(ctx context.Context, farmerID int, dto models.CreateWithdrawalDTO) (*models.WithdrawalRequest, error)
	UpdateWalletAccount(ctx context.Context, farmerID int, dto models.UpdateWalletAccountDTO) error
	CreditEscrowPending(ctx context.Context, orderID int) error
	ReleaseEscrowToBalance(ctx context.Context, orderCode string) error
	GetPayoutAccounts(ctx context.Context, farmerID int) ([]models.PayoutAccount, error)
	AddPayoutAccount(ctx context.Context, farmerID int, dto models.CreatePayoutAccountDTO) (*models.PayoutAccount, error)
	SetPrimaryPayoutAccount(ctx context.Context, farmerID int, accountID int) error
	DeletePayoutAccount(ctx context.Context, farmerID int, accountID int) error
}

type mysqlWalletRepository struct {
	db *sql.DB
}

func NewWalletRepository(db *sql.DB) WalletRepository {
	return &mysqlWalletRepository{db: db}
}

func (r *mysqlWalletRepository) EnsureWallet(ctx context.Context, farmerID int) (*models.FarmerWallet, error) {
	var w models.FarmerWallet
	query := `SELECT id, farmer_id, balance, pending_balance, COALESCE(bank_name, ''), COALESCE(account_number, ''), COALESCE(account_holder, ''), created_at, updated_at FROM farmer_wallets WHERE farmer_id = ?`
	err := r.db.QueryRowContext(ctx, query, farmerID).Scan(&w.ID, &w.FarmerID, &w.Balance, &w.PendingBalance, &w.BankName, &w.AccountNumber, &w.AccountHolder, &w.CreatedAt, &w.UpdatedAt)
	if err == nil {
		return &w, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	var farmerName string
	nameErr := r.db.QueryRowContext(ctx, `SELECT name FROM users WHERE id = ?`, farmerID).Scan(&farmerName)
	if nameErr != nil {
		farmerName = fmt.Sprintf("Petani #%d", farmerID)
	}

	insertQuery := `INSERT INTO farmer_wallets (farmer_id, balance, pending_balance, bank_name, account_number, account_holder) VALUES (?, 0.00, 0.00, 'Bank Rakyat Indonesia (BRI)', '', ?)`
	res, insertErr := r.db.ExecContext(ctx, insertQuery, farmerID, farmerName)
	if insertErr != nil {
		return nil, insertErr
	}

	lastID, _ := res.LastInsertId()
	w.ID = int(lastID)
	w.FarmerID = farmerID
	w.Balance = 0.00
	w.PendingBalance = 0.00
	w.BankName = "Bank Rakyat Indonesia (BRI)"
	w.AccountHolder = farmerName

	return &w, nil
}

func (r *mysqlWalletRepository) GetWalletOverview(ctx context.Context, farmerID int) (*models.WalletOverviewDTO, error) {
	wallet, err := r.EnsureWallet(ctx, farmerID)
	if err != nil {
		return nil, err
	}

	dto := &models.WalletOverviewDTO{
		Wallet:       *wallet,
		Transactions: make([]models.WalletTransaction, 0),
		Withdrawals:  make([]models.WithdrawalRequest, 0),
	}

	txQuery := `SELECT id, wallet_id, order_id, type, amount, status, description, created_at FROM wallet_transactions WHERE wallet_id = ? ORDER BY created_at DESC LIMIT 50`
	rows, err := r.db.QueryContext(ctx, txQuery, wallet.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tx models.WalletTransaction
			if scanErr := rows.Scan(&tx.ID, &tx.WalletID, &tx.OrderID, &tx.Type, &tx.Amount, &tx.Status, &tx.Description, &tx.CreatedAt); scanErr == nil {
				dto.Transactions = append(dto.Transactions, tx)
			}
		}
	}

	wdQuery := `SELECT id, farmer_id, amount, target_type, target_provider, target_account, account_holder, status, created_at FROM withdrawal_requests WHERE farmer_id = ? ORDER BY created_at DESC LIMIT 20`
	wdRows, err := r.db.QueryContext(ctx, wdQuery, farmerID)
	if err == nil {
		defer wdRows.Close()
		for wdRows.Next() {
			var wd models.WithdrawalRequest
			if scanErr := wdRows.Scan(&wd.ID, &wd.FarmerID, &wd.Amount, &wd.TargetType, &wd.TargetProvider, &wd.TargetAccount, &wd.AccountHolder, &wd.Status, &wd.CreatedAt); scanErr == nil {
				dto.Withdrawals = append(dto.Withdrawals, wd)
			}
		}
	}

	dto.PayoutAccounts = make([]models.PayoutAccount, 0)
	pQuery := `SELECT id, farmer_id, account_type, provider_name, account_number, account_holder, is_primary, created_at, updated_at FROM farmer_payout_accounts WHERE farmer_id = ? ORDER BY is_primary DESC, id ASC`
	pRows, err := r.db.QueryContext(ctx, pQuery, farmerID)
	if err == nil {
		defer pRows.Close()
		for pRows.Next() {
			var pa models.PayoutAccount
			if scanErr := pRows.Scan(&pa.ID, &pa.FarmerID, &pa.AccountType, &pa.ProviderName, &pa.AccountNumber, &pa.AccountHolder, &pa.IsPrimary, &pa.CreatedAt, &pa.UpdatedAt); scanErr == nil {
				dto.PayoutAccounts = append(dto.PayoutAccounts, pa)
			}
		}
	}

	return dto, nil
}

func (r *mysqlWalletRepository) CreateWithdrawalRequest(ctx context.Context, farmerID int, dto models.CreateWithdrawalDTO) (*models.WithdrawalRequest, error) {
	if dto.Amount < 10000 {
		return nil, errors.New("minimum penarikan dana adalah Rp 10.000")
	}

	wallet, err := r.EnsureWallet(ctx, farmerID)
	if err != nil {
		return nil, err
	}

	if wallet.Balance < dto.Amount {
		return nil, fmt.Errorf("saldo tidak mencukupi (Saldo: Rp %.0f, Penarikan: Rp %.0f)", wallet.Balance, dto.Amount)
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	updateQuery := `UPDATE farmer_wallets SET balance = balance - ? WHERE id = ? AND balance >= ?`
	res, err := tx.ExecContext(ctx, updateQuery, dto.Amount, wallet.ID, dto.Amount)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		_ = tx.Rollback()
		return nil, errors.New("gagal memotong saldo, saldo tidak mencukupi")
	}

	insertWd := `INSERT INTO withdrawal_requests (farmer_id, amount, target_type, target_provider, target_account, account_holder, status) VALUES (?, ?, ?, ?, ?, ?, 'transferred')`
	wdRes, err := tx.ExecContext(ctx, insertWd, farmerID, dto.Amount, dto.TargetType, dto.TargetProvider, dto.TargetAccount, dto.AccountHolder)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	wdID, _ := wdRes.LastInsertId()

	desc := fmt.Sprintf("Penarikan Saldo ke %s (%s a.n. %s)", dto.TargetProvider, dto.TargetAccount, dto.AccountHolder)
	insertTx := `INSERT INTO wallet_transactions (wallet_id, order_id, type, amount, status, description) VALUES (?, NULL, 'debit_withdrawal', ?, 'completed', ?)`
	if _, err := tx.ExecContext(ctx, insertTx, wallet.ID, dto.Amount, desc); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &models.WithdrawalRequest{
		ID:             int(wdID),
		FarmerID:       farmerID,
		Amount:         dto.Amount,
		TargetType:     dto.TargetType,
		TargetProvider: dto.TargetProvider,
		TargetAccount:  dto.TargetAccount,
		AccountHolder:  dto.AccountHolder,
		Status:         "transferred",
	}, nil
}

func (r *mysqlWalletRepository) UpdateWalletAccount(ctx context.Context, farmerID int, dto models.UpdateWalletAccountDTO) error {
	wallet, err := r.EnsureWallet(ctx, farmerID)
	if err != nil {
		return err
	}

	query := `UPDATE farmer_wallets SET bank_name = ?, account_number = ?, account_holder = ? WHERE id = ?`
	_, err = r.db.ExecContext(ctx, query, dto.BankName, dto.AccountNumber, dto.AccountHolder, wallet.ID)
	return err
}

func (r *mysqlWalletRepository) CreditEscrowPending(ctx context.Context, orderID int) error {
	query := `SELECT farmer_id, subtotal FROM order_items WHERE order_id = ? AND farmer_id IS NOT NULL AND farmer_id > 0`
	rows, err := r.db.QueryContext(ctx, query, orderID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var fID int
		var sub float64
		if scanErr := rows.Scan(&fID, &sub); scanErr == nil {
			_, _ = r.EnsureWallet(ctx, fID)
			updateQuery := `UPDATE farmer_wallets SET pending_balance = pending_balance + ? WHERE farmer_id = ?`
			_, _ = r.db.ExecContext(ctx, updateQuery, sub, fID)
		}
	}

	return nil
}

func (r *mysqlWalletRepository) ReleaseEscrowToBalance(ctx context.Context, orderCode string) error {
	var orderID int
	err := r.db.QueryRowContext(ctx, `SELECT id FROM orders WHERE order_code = ?`, orderCode).Scan(&orderID)
	if err != nil {
		return err
	}

	query := `SELECT farmer_id, product_name, subtotal FROM order_items WHERE order_id = ? AND farmer_id IS NOT NULL AND farmer_id > 0`
	rows, err := r.db.QueryContext(ctx, query, orderID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var fID int
		var pName string
		var sub float64
		if scanErr := rows.Scan(&fID, &pName, &sub); scanErr == nil {
			wallet, wErr := r.EnsureWallet(ctx, fID)
			if wErr != nil {
				continue
			}

			tx, txErr := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
			if txErr != nil {
				continue
			}

			transferQuery := `UPDATE farmer_wallets SET pending_balance = GREATEST(0, pending_balance - ?), balance = balance + ? WHERE id = ?`
			if _, execErr := tx.ExecContext(ctx, transferQuery, sub, sub, wallet.ID); execErr != nil {
				_ = tx.Rollback()
				continue
			}

			desc := fmt.Sprintf("Pencairan Dana Hasil Panen %s (Pesanan #%s)", pName, orderCode)
			logQuery := `INSERT INTO wallet_transactions (wallet_id, order_id, type, amount, status, description) VALUES (?, ?, 'credit_earning', ?, 'completed', ?)`
			if _, logErr := tx.ExecContext(ctx, logQuery, wallet.ID, orderID, sub, desc); logErr != nil {
				_ = tx.Rollback()
				continue
			}

			_ = tx.Commit()
		}
	}

	return nil
}

func (r *mysqlWalletRepository) GetPayoutAccounts(ctx context.Context, farmerID int) ([]models.PayoutAccount, error) {
	query := `SELECT id, farmer_id, account_type, provider_name, account_number, account_holder, is_primary, created_at, updated_at FROM farmer_payout_accounts WHERE farmer_id = ? ORDER BY is_primary DESC, id ASC`
	rows, err := r.db.QueryContext(ctx, query, farmerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := make([]models.PayoutAccount, 0)
	for rows.Next() {
		var pa models.PayoutAccount
		if err := rows.Scan(&pa.ID, &pa.FarmerID, &pa.AccountType, &pa.ProviderName, &pa.AccountNumber, &pa.AccountHolder, &pa.IsPrimary, &pa.CreatedAt, &pa.UpdatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, pa)
	}
	return accounts, nil
}

func (r *mysqlWalletRepository) AddPayoutAccount(ctx context.Context, farmerID int, dto models.CreatePayoutAccountDTO) (*models.PayoutAccount, error) {
	if dto.ProviderName == "" || dto.AccountNumber == "" || dto.AccountHolder == "" {
		return nil, errors.New("nama penyedia, nomor rekening/e-wallet, dan nama pemilik wajib diisi")
	}

	var count int
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM farmer_payout_accounts WHERE farmer_id = ?`, farmerID).Scan(&count)
	if count == 0 {
		dto.IsPrimary = true
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if dto.IsPrimary {
		if _, err := tx.ExecContext(ctx, `UPDATE farmer_payout_accounts SET is_primary = 0 WHERE farmer_id = ?`, farmerID); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}

	insertQuery := `INSERT INTO farmer_payout_accounts (farmer_id, account_type, provider_name, account_number, account_holder, is_primary) VALUES (?, ?, ?, ?, ?, ?)`
	res, err := tx.ExecContext(ctx, insertQuery, farmerID, dto.AccountType, dto.ProviderName, dto.AccountNumber, dto.AccountHolder, dto.IsPrimary)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	lastID, _ := res.LastInsertId()

	if dto.IsPrimary {
		_, _ = tx.ExecContext(ctx, `UPDATE farmer_wallets SET bank_name = ?, account_number = ?, account_holder = ? WHERE farmer_id = ?`, dto.ProviderName, dto.AccountNumber, dto.AccountHolder, farmerID)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &models.PayoutAccount{
		ID:            int(lastID),
		FarmerID:      farmerID,
		AccountType:   dto.AccountType,
		ProviderName:  dto.ProviderName,
		AccountNumber: dto.AccountNumber,
		AccountHolder: dto.AccountHolder,
		IsPrimary:     dto.IsPrimary,
	}, nil
}

func (r *mysqlWalletRepository) SetPrimaryPayoutAccount(ctx context.Context, farmerID int, accountID int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	var providerName, accountNumber, accountHolder string
	err = tx.QueryRowContext(ctx, `SELECT provider_name, account_number, account_holder FROM farmer_payout_accounts WHERE id = ? AND farmer_id = ?`, accountID, farmerID).Scan(&providerName, &accountNumber, &accountHolder)
	if err != nil {
		_ = tx.Rollback()
		return errors.New("rekening penarikan tidak ditemukan")
	}

	if _, err := tx.ExecContext(ctx, `UPDATE farmer_payout_accounts SET is_primary = 0 WHERE farmer_id = ?`, farmerID); err != nil {
		_ = tx.Rollback()
		return err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE farmer_payout_accounts SET is_primary = 1 WHERE id = ? AND farmer_id = ?`, accountID, farmerID); err != nil {
		_ = tx.Rollback()
		return err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE farmer_wallets SET bank_name = ?, account_number = ?, account_holder = ? WHERE farmer_id = ?`, providerName, accountNumber, accountHolder, farmerID); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (r *mysqlWalletRepository) DeletePayoutAccount(ctx context.Context, farmerID int, accountID int) error {
	var isPrimary bool
	err := r.db.QueryRowContext(ctx, `SELECT is_primary FROM farmer_payout_accounts WHERE id = ? AND farmer_id = ?`, accountID, farmerID).Scan(&isPrimary)
	if err != nil {
		return errors.New("rekening tujuan tidak ditemukan")
	}

	var count int
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM farmer_payout_accounts WHERE farmer_id = ?`, farmerID).Scan(&count)
	if count <= 1 {
		return errors.New("rekening utama tidak dapat dihapus jika hanya tersisa satu rekening")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if _, err := tx.ExecContext(ctx, `DELETE FROM farmer_payout_accounts WHERE id = ? AND farmer_id = ?`, accountID, farmerID); err != nil {
		_ = tx.Rollback()
		return err
	}

	if isPrimary {
		var nextID int
		var nextProvider, nextAccount, nextHolder string
		if err := tx.QueryRowContext(ctx, `SELECT id, provider_name, account_number, account_holder FROM farmer_payout_accounts WHERE farmer_id = ? ORDER BY id ASC LIMIT 1`, farmerID).Scan(&nextID, &nextProvider, &nextAccount, &nextHolder); err == nil {
			_, _ = tx.ExecContext(ctx, `UPDATE farmer_payout_accounts SET is_primary = 1 WHERE id = ?`, nextID)
			_, _ = tx.ExecContext(ctx, `UPDATE farmer_wallets SET bank_name = ?, account_number = ?, account_holder = ? WHERE farmer_id = ?`, nextProvider, nextAccount, nextHolder, farmerID)
		}
	}

	return tx.Commit()
}

