package main

import (
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type CreateTransactionRequest struct {
	Receivernumber int64 `json:"receiverNumber" validate:"required,gt=0"`
	Amount         int64 `json:"amount" validate:"required,min=1"`
}
type Transaction struct {
	ID             int       `json:"id"`
	SenderID       int       `json:"senderID"`
	ReceiverNumber int64     `json:"receiverNumber"`
	Amount         int64     `json:"amount"`
	CreatedAt      time.Time `json:"createdAt"`
}
type Account struct {
	ID                int       `json:"id"`
	PhoneNumber       string    `json:"phoneNumber"`
	FirstName         string    `json:"firstName"`
	LastName          string    `json:"lastName"`
	Number            int64     `json:"number"`
	EncryptedPassword string    `json:"-"`
	Balance           int64     `json:"balance"`
	CreatedAt         time.Time `json:"createdAt"`
}

type CreateAccountRequest struct {
	FirstName         string `json:"firstName" validate:"required,min=2,max=50"`
	LastName          string `json:"lastName" validate:"required,min=2,max=50"`
	PhoneNumber       string `json:"phoneNumber" validate:"required,min=7,max=15"`
	EncryptedPassword string `json:"password" validate:"required,min=8"`
}
type UpdateAccountRequest struct {
	PhoneNumber *string `json:"phoneNumber" validate:"omitempty,gt=0"`
	FirstName   *string `json:"firstName" validate:"omitempty,gt=0"`
	LastName    *string `json:"lastName" validate:"omitempty,gt=0"`
}
type UpdatePasswordRequest struct {
	Oldpw string `json:"oldpw"`
	Newpw string `json:"newpw"`
}
type LoginRequest struct {
	Number   int64  `json:"number"`
	Password string `json:"password"`
}
type LoginResponse struct {
	Number int64  `json:"number"`
	Token  string `json:"token"`
}

func newAccount(phoneNumber string, firstName, lastName string, password string) (*Account, error) {
	encpw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &Account{
		// ID:        rand.Intn(1000),
		PhoneNumber:       phoneNumber,
		FirstName:         firstName,
		LastName:          lastName,
		Number:            int64(rand.Intn(100000)),
		EncryptedPassword: string(encpw),
		CreatedAt:         time.Now().UTC(),
	}, nil
}
func newTransaction(senderID int, receivernumber int64, amount int64) (*Transaction, error) {

	return &Transaction{
		SenderID:       senderID,
		ReceiverNumber: receivernumber,
		Amount:         amount,
		CreatedAt:      time.Now().UTC(),
	}, nil
}

func (acc *Account) validatePassword(pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(acc.EncryptedPassword), []byte(pw)) == nil

}
