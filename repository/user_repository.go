package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/agroconnect/service-order/models"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id int) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	UpdatePassword(ctx context.Context, id int, passwordHash string) error
}

type mysqlUserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &mysqlUserRepository{db: db}
}

func (r *mysqlUserRepository) CreateUser(ctx context.Context, user *models.User) error {
	query := `INSERT INTO users (name, email, password_hash, role) VALUES (?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, query, user.Name, user.Email, user.PasswordHash, user.Role)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	user.ID = int(id)
	return nil
}

func (r *mysqlUserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, name, email, password_hash, role, created_at FROM users WHERE email = ?`
	row := r.db.QueryRowContext(ctx, query, email)

	var u models.User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &u, nil
}

func (r *mysqlUserRepository) FindByID(ctx context.Context, id int) (*models.User, error) {
	query := `SELECT id, name, email, password_hash, role, created_at FROM users WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var u models.User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &u, nil
}

func (r *mysqlUserRepository) UpdateUser(ctx context.Context, user *models.User) error {
	query := `UPDATE users SET name = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, user.Name, user.ID)
	return err
}

func (r *mysqlUserRepository) UpdatePassword(ctx context.Context, id int, passwordHash string) error {
	query := `UPDATE users SET password_hash = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, passwordHash, id)
	return err
}
