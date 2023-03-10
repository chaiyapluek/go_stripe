package models

import (
	"context"
	"database/sql"
	"time"
)

// DBModel is the type for database connection
type DBModel struct {
	DB *sql.DB
}

// Models is the wrapper for all models
type Models struct {
	DB DBModel
}

// NewModel return a model type with database connection pool
func NewModel(db *sql.DB) Models {
	return Models{
		DB: DBModel{
			DB: db,
		},
	}
}

// Widget is the type for all widgets
type Widget struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	InventoryLevel int       `json:"inventory_level"`
	Price          int       `json:"price"`
	Image          string    `json:"image"`
	CreateAt       time.Time `json:"-"`
	UpdateAt       time.Time `json:"-"`
}

// Order is the type for all orders
type Order struct {
	ID            int       `json:"id"`
	WidgetID      int       `json:"widget_id"`
	TransactionID int       `json:"transaction_id"`
	StatusID      int       `json:"status_id"`
	Quantity      int       `json:"quantity"`
	Amount        int       `json:"amount"`
	CreateAt      time.Time `json:"-"`
	UpdateAt      time.Time `json:"-"`
}

// Status is the type for all order status
type Status struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	CreateAt time.Time `json:"-"`
	UpdateAt time.Time `json:"-"`
}

// TransactionStatus is the type for all transaction status
type TransactionStatus struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	CreateAt time.Time `json:"-"`
	UpdateAt time.Time `json:"-"`
}

// Transaction is the type for all transactions
type Transaction struct {
	ID                  int       `json:"id"`
	Amount              int       `json:"amount"`
	Currency            string    `json:"currency"`
	LastFour            string    `json:"last_four"`
	BankReturnCode      string    `json:"bank_return_code"`
	TransactionStatusID int       `json:"transaction_status_id"`
	CreateAt            time.Time `json:"-"`
	UpdateAt            time.Time `json:"-"`
}

type User struct {
	ID        int       `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	CreateAt  time.Time `json:"-"`
	UpdateAt  time.Time `json:"-"`
}

// coalesce return the first non-null value in a list:
func (m *DBModel) GetWidget(id int) (Widget, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var widget Widget

	row := m.DB.QueryRowContext(ctx, `
		SELECT id, name, description, inventory_level, price, COALESCE(image, ""), created_at, updated_at 
		FROM widgets 
		WHERE id=?`, id)
	err := row.Scan(
		&widget.ID,
		&widget.Name,
		&widget.Description,
		&widget.InventoryLevel,
		&widget.Price,
		&widget.Image,
		&widget.CreateAt,
		&widget.UpdateAt,
	)
	if err != nil {
		return widget, err
	}
	return widget, nil
}
