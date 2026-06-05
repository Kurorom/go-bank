package main

import (
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type TransferRequet struct {
	ToAccount int64 `json:"toAccount"`
	Amount    int64 `json:"amount"`
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
	PhoneNumber       string `json:"phoneNumber"`
	FirstName         string `json:"firstName"`
	LastName          string `json:"lastName"`
	EncryptedPassword string `json:"password"`
}
type UpdateAccountRequest struct {
	PhoneNumber *string `json:"phoneNumber"`
	FirstName   *string `json:"firstName"`
	LastName    *string `json:"lastName"`
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

func (acc *Account) validatePassword(pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(acc.EncryptedPassword), []byte(pw)) == nil

}
