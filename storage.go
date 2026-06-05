package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

type Storage interface {
	CreateAccount(*Account) error
	DeleteAccount(int) error
	UpdateAccount(*Account) error
	UpdatePassword(*Account, string) error
	GetAccountByNumber(int) (*Account, error)
	GetAccountByID(int) (*Account, error)
	GetAccounts() ([]*Account, error)
	UpdateBalace(*Account, int) error
	Transfer(fromID int, toAccountNumber int64, amount int64) error
}

type PostgresStore struct {
	db *sql.DB
}

func newPostgresStore() (*PostgresStore, error) {
	// 1. Fetch values from environment variables, or fall back to defaults
	dbUser := getEnv("DB_USER", "postgres")
	dbName := getEnv("DB_NAME", "postgres")
	dbPass := getEnv("DB_PASSWORD", "gobankpassword") // fallback for local dev
	dbHost := getEnv("DB_HOST", "localhost")

	dsn := fmt.Sprintf("user=%s dbname=%s password=%s host=%s sslmode=disable",
		dbUser, dbName, dbPass, dbHost)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresStore{
		db: db,
	}, nil
}

// Helper function to handle environment fallbacks
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// func newPostgresStore() (*PostgresStore, error) {
// 	dsn := "user=postgres dbname=postgres password=gobankpassword sslmode=disable"
// 	db, err := sql.Open("postgres", dsn)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	if err := db.Ping(); err != nil {
// 		return nil, err
// 	}

// 	return &PostgresStore{
// 		db: db,
// 	}, nil
// }

func (s *PostgresStore) init() error {
	return s.createAccountTable()
}

func (s *PostgresStore) createAccountTable() error {
	query := `CREATE TABLE IF NOT EXISTS account
	 (
		id serial primary key,
		phone_number varchar(20),      
		first_name varchar(50),
		last_name varchar(50),
		number serial,
		encrypted_password varchar(100),
		balance numeric,
		created_at timestamp 
	)`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) CreateAccount(acc *Account) error {
	query := `INSERT INTO account 
	(phone_number,first_name,last_name,number,encrypted_password,balance,created_at)
	values
	($1,$2,$3,$4,$5,$6,$7)
	`
	resp, err := s.db.Exec(query,
		acc.PhoneNumber,
		acc.FirstName,
		acc.LastName,
		acc.Number,
		acc.EncryptedPassword,
		acc.Balance,
		acc.CreatedAt)
	if err != nil {
		return err
	}
	fmt.Printf("%+v\n", resp)

	return nil
}
func (s *PostgresStore) UpdateAccount(acc *Account) error {
	query := `UPDATE account 
	set phone_number=$1,first_name=$2,last_name=$3 where id = $4
	`
	res, err := s.db.Exec(query,
		acc.PhoneNumber,
		acc.FirstName,
		acc.LastName,
		acc.ID)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("account with id %d not found", acc.ID)
	}

	return nil
}

func (s *PostgresStore) UpdatePassword(acc *Account, encpw string) error {

	query := `UPDATE account 
	set encrypted_password = $1 where id = $2
	`
	res, err := s.db.Exec(query,
		encpw,
		acc.ID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("account with id %d not found", acc.ID)
	}
	return nil
}

func (s *PostgresStore) DeleteAccount(id int) error {
	result, err := s.db.Exec("DELETE FROM account where id = $1", id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("account with id %d doesn't exist", id)
	}
	return nil
}

func (s *PostgresStore) GetAccountByNumber(number int) (*Account, error) {
	rows, err := s.db.Query("SELECT id, phone_number, first_name, last_name, number, encrypted_password, balance, created_at FROM account WHERE number=$1", number)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		return scanIntoAccount(rows)
	}

	return nil, fmt.Errorf("account with number %d doesnt exist", number)

}
func (s *PostgresStore) GetAccountByID(id int) (*Account, error) {
	rows, err := s.db.Query("SELECT id, phone_number, first_name, last_name, number, encrypted_password, balance, created_at FROM account WHERE id=$1", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		return scanIntoAccount(rows)
	}

	return nil, fmt.Errorf("account with id %d doesnt exist", id)
}

func (s *PostgresStore) GetAccounts() ([]*Account, error) {
	rows, err := s.db.Query("SELECT  * from account")
	if err != nil {
		return nil, err
	}

	accounts := []*Account{}
	for rows.Next() {
		account, err := scanIntoAccount(rows)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, account)
	}
	return accounts, nil
}
func (s *PostgresStore) UpdateBalace(acc *Account, amount int) error {
	query := `UPDATE account 
	set balance= $1 where number = $2`

	res, err := s.db.Exec(query, amount, acc.Number)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("transfer failed")
	}
	return nil
}

func (s *PostgresStore) Transfer(senderID int, ReceiverNumber int64, amount int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	deductQuery := `UPDATE account SET balance = balance - $1 WHERE id = $2 AND balance >= $1`
	res, err := tx.Exec(deductQuery, amount, senderID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()

	if rowsAffected == 0 {
		return fmt.Errorf("transfer failed")
	}
	creditQuery := `UPDATE account SET balance = balance + $1 WHERE number = $2`
	res, err = tx.Exec(creditQuery, amount, ReceiverNumber)
	if err != nil {
		return err
	}
	rowsAffected, err = res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("recipient account number %d does not exist", ReceiverNumber)
	}
	return tx.Commit()
}

func scanIntoAccount(rows *sql.Rows) (*Account, error) {
	account := new(Account)
	err := rows.Scan(
		&account.ID,
		&account.PhoneNumber,
		&account.FirstName,
		&account.LastName,
		&account.Number,
		&account.EncryptedPassword,
		&account.Balance,
		&account.CreatedAt)

	return account, err
}
