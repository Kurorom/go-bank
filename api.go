package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type APIServer struct {
	listenAddr string
	store      Storage
}
type contextKey string

const jwtClaimsKey contextKey = "jwtClaims"

func NewAPIServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) Run() {
	router := mux.NewRouter()

	router.HandleFunc("/login", makeHTTPHandlerFunc(s.handleLogin))
	router.HandleFunc("/account", makeHTTPHandlerFunc(s.handleAccount))
	router.HandleFunc("/account/{id}", withJWTAuth(makeHTTPHandlerFunc(s.handleGetAccountByID), s.store))
	router.HandleFunc("/account/{id}/password", withJWTAuth(makeHTTPHandlerFunc(s.handleUpdatePassword), s.store)) // to do
	router.HandleFunc("/transfer", withJWTAuth(makeHTTPHandlerFunc(s.handleTransfer), s.store))

	log.Println("JSON API server listening in port: ", s.listenAddr)

	http.ListenAndServe(s.listenAddr, router)
}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf("method not allowed %s", r.Method)
	}
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	acc, err := s.store.GetAccountByNumber(int(req.Number))
	if err != nil {
		return err
	}

	if !acc.validatePassword(req.Password) {
		return fmt.Errorf("login failed")
	}
	token, err := createJWT(acc)
	if err != nil {
		return err
	}
	resp := LoginResponse{
		Token:  token,
		Number: acc.Number,
	}
	return WriteJSON(w, http.StatusOK, resp)
}
func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "GET":
		return s.handleGetAccount(w, r)
	case "POST":
		return s.handleCreateAccount(w, r)
	}
	return fmt.Errorf("method not allowed %s", r.Method)
}
func (s *APIServer) handleGetAccount(w http.ResponseWriter, r *http.Request) error {
	accounts, err := s.store.GetAccounts()
	if err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, accounts)
}

func (s *APIServer) handleGetAccountByID(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {

		id, err := getID(r)
		if err != nil {
			return err
		}
		account, err := s.store.GetAccountByID(id)
		if err != nil {
			return err
		}
		return WriteJSON(w, http.StatusOK, account)
	}
	if r.Method == "DELETE" {
		return s.handleDeleteAccount(w, r)
	}

	if r.Method == "PATCH" {

		return s.handleUpdateAccount(w, r)

	}
	return fmt.Errorf("method not allowed %s", r.Method)

}

func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	req := new(CreateAccountRequest)
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return err
	}
	if req.PhoneNumber == "" || req.FirstName == "" || req.LastName == "" || req.EncryptedPassword == "" {
		return fmt.Errorf("fields cannot be empty")
	}
	account, err := newAccount(req.PhoneNumber, req.FirstName, req.LastName, req.EncryptedPassword)
	if err != nil {
		return err
	}
	if err := s.store.CreateAccount(account); err != nil {
		return err
	}

	// tokenString, err := createJWT(account)
	// if err != nil {
	// 	return err
	// }
	// fmt.Println("JWT token : ", tokenString)
	return WriteJSON(w, http.StatusOK, account)
}

func (s *APIServer) handleUpdateAccount(w http.ResponseWriter, r *http.Request) error {
	req := new(UpdateAccountRequest)
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return err
	}
	id, err := getID(r)
	if err != nil {
		return err
	}
	existingAccount, err := s.store.GetAccountByID(id)
	if err != nil {
		return err
	}
	if req.PhoneNumber != nil {
		existingAccount.PhoneNumber = *req.PhoneNumber
	}
	if req.FirstName != nil {
		existingAccount.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		existingAccount.LastName = *req.LastName
	}
	//preventing first name from being set to empty string
	if existingAccount.FirstName == "" || existingAccount.LastName == "" || existingAccount.PhoneNumber == "" {
		return fmt.Errorf("field cannot be empty")
	}

	if err := s.store.UpdateAccount(existingAccount); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, existingAccount)
}
func (s *APIServer) handleUpdatePassword(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf("method not allowed %s", r.Method)
	}

	claims, ok := r.Context().Value(jwtClaimsKey).(jwt.MapClaims)
	if !ok {
		permissionDenied(w)
		return nil
	}

	id, err := getID(r)
	if err != nil {
		return err
	}
	acc, err := s.store.GetAccountByID(id)
	if err != nil {
		return err
	}

	if acc.Number != int64(claims["accountNumber"].(float64)) {
		permissionDenied(w)
		return nil
	}
	req := new(UpdatePasswordRequest)
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return err
	}

	if !acc.validatePassword(req.Oldpw) {
		return fmt.Errorf("old password doesnt match")
	}
	encpw, err := bcrypt.GenerateFromPassword([]byte(req.Newpw), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = s.store.UpdatePassword(acc, string(encpw))
	if err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, map[string]string{"message": "password updated successfully"})
}

func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) error {
	id, err := getID(r)
	if err != nil {
		return err
	}
	if err := s.store.DeleteAccount(id); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, map[string]int{"deleted": id})
}

func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf("method not allowed %s", r.Method)
	}

	claims, ok := r.Context().Value(jwtClaimsKey).(jwt.MapClaims)
	if !ok {
		permissionDenied(w)
		return nil
	}

	transferReq := new(TransferRequet)

	if err := json.NewDecoder(r.Body).Decode(transferReq); err != nil {
		return err
	}

	defer r.Body.Close()

	if transferReq.Amount <= 0 {
		return fmt.Errorf("transfer amount must be greater than zero")
	}

	fromAcc, err := s.store.GetAccountByNumber(int(claims["accountNumber"].(float64)))
	if err != nil {
		return err
	}
	if fromAcc.Number == transferReq.ToAccount {
		return fmt.Errorf("cannot transfer money to your own account number")
	}
	if fromAcc.Balance < transferReq.Amount {
		return fmt.Errorf("insufficient balance")
	}
	err = s.store.Transfer(fromAcc.ID, transferReq.ToAccount, transferReq.Amount)
	if err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, transferReq)
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func permissionDenied(w http.ResponseWriter) {
	WriteJSON(w, http.StatusForbidden, APIError{Error: "Permission Denied"})

}

// jwt stuff
// JWT token :  eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJFeHBpcmVzQXQiOjE1MTYyMzkwMjIsImFjY291bnROdW1iZXIiOjg3ODh9.EOukzfgAFD3kvNsNNtDHC4zaIWHYKT94rWT0dx0FyJM
func createJWT(account *Account) (string, error) {
	// Create the Claims
	claims := &jwt.MapClaims{
		"ExpiresAt":     jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		"accountNumber": account.Number,
	}

	secret := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

func withJWTAuth(handlerFunc http.HandlerFunc, s Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("calling JWT auth middleware..")

		tokenString := r.Header.Get("x-jwt-token")
		token, err := validateJWT(tokenString)
		if err != nil {
			permissionDenied(w)
			return
		}
		if !token.Valid {
			permissionDenied(w)
			return
		}
		claims := token.Claims.(jwt.MapClaims)

		if _, hasID := mux.Vars(r)["id"]; hasID {
			userID, err := getID(r)
			if err != nil {
				permissionDenied(w)
			}
			account, err := s.GetAccountByID(userID)
			if err != nil {
				permissionDenied(w)
				return
			}
			if account.Number != int64(claims["accountNumber"].(float64)) {
				permissionDenied(w)
				return
			}
		}
		ctx := context.WithValue(r.Context(), jwtClaimsKey, claims)
		r = r.WithContext(ctx)

		handlerFunc(w, r)
	}
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")

	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Return the secret as a byte slice
		return []byte(secret), nil
	})
}

type apiFunc func(http.ResponseWriter, *http.Request) error

type APIError struct {
	Error string `json:"error"`
}

func makeHTTPHandlerFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
		}
	}

}

func getID(r *http.Request) (int, error) {
	idstr := mux.Vars(r)["id"]

	id, err := strconv.Atoi(idstr)
	if err != nil {
		return id, fmt.Errorf("invalid id given %s", idstr)

	}
	return id, nil
}
