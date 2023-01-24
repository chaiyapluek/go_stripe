package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chaiyapluek/go-stripe/internal/cards"
	"github.com/chaiyapluek/go-stripe/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/stripe/stripe-go/v72"
)

type stripePayload struct {
	Currency      string `json:"currency"`
	Amount        string `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	Email         string `json:"email"`
	LastFour      string `json:"last_four"`
	Plan          string `json:"plan"`
	CardBrand     string `json:"card_brand"`
	ExpiryMonth   int    `json:"expiry_month"`
	ExpiryYear    int    `json:"expiry_year"`
	ProductID     string `json:"product_id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
}

type jsonResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Content string `json:"content,omitempty"`
	ID      int    `json:"id,omitempty"`
}

func (app *application) GetPaymentIntent(w http.ResponseWriter, r *http.Request) {

	var payload stripePayload

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		app.errorLog.Println((*r).Body, err)
		return
	}

	amount, err := strconv.Atoi(payload.Amount)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: payload.Currency,
	}

	okey := true
	pi, msg, err := card.Charge(payload.Currency, amount)
	if err != nil {
		okey = false
	}

	if okey {
		out, err := json.MarshalIndent(pi, "", " ")
		if err != nil {
			app.errorLog.Println(err)
			return
		}
		w.Header().Set("Content-type", "application/json")
		w.Write(out)
	} else {
		j := jsonResponse{
			OK:      false,
			Message: msg,
			Content: "",
		}
		out, err := json.MarshalIndent(j, "", " ")
		if err != nil {
			app.errorLog.Println(err)
		}
		w.Header().Set("Content-type", "application/json")
		w.Write(out)
	}

}

func (app *application) GetWidgetById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	widgetId, _ := strconv.Atoi(id)

	widget, err := app.DB.GetWidget(widgetId)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	out, err := json.MarshalIndent(widget, "", " ")
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	w.Header().Set("Content-type", "application/json")
	w.Write(out)
}

func (app *application) CreateCustomerAndSubscribeToPlan(w http.ResponseWriter, r *http.Request) {
	var data stripePayload
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		app.errorLog.Println("Cannot decode payload", err)
		return
	}
	app.infoLog.Println(data.Email)
	app.infoLog.Println(data.PaymentMethod)
	app.infoLog.Println(data.Plan)
	app.infoLog.Println(data.LastFour)

	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: data.Currency,
	}

	okey := true
	var subscription *stripe.Subscription
	txnMsg := "Transaction successful"

	stripeCustomer, msg, err := card.CreateCustomer(data.PaymentMethod, data.Email)
	if err != nil {
		app.errorLog.Println("Cannot create customer", err)
		okey = false
		txnMsg = msg
	}

	if okey {
		subscription, err = card.SubscribeToPlan(stripeCustomer, data.Plan, data.Email, data.LastFour, "")
		if err != nil {
			app.errorLog.Println("Cannot subscribe", err)
			okey = false
			txnMsg = "Error subscribing customer"
		}
		app.infoLog.Println("Subsciption id is", subscription.ID)
	}

	// Assume every transaction is a new customer
	if okey {
		productID, _ := strconv.Atoi(data.ProductID)
		customerID, err := app.SaveCustomer(data.FirstName, data.LastName, data.Email)
		if err != nil {
			app.errorLog.Println("Unable to save customer", err)
			return
		}

		// create a new transaction
		amount, _ := strconv.Atoi(data.Amount)
		txn := models.Transaction{
			Amount:              amount,
			Currency:            data.Currency,
			LastFour:            data.LastFour,
			ExpiryMonth:         data.ExpiryMonth,
			ExpiryYear:          data.ExpiryYear,
			TransactionStatusID: 2,
		}
		txnID, err := app.SaveTransaction(txn)
		if err != nil {
			app.errorLog.Println("Unable to save transaction", err)
			return
		}

		// create a new order
		order := models.Order{
			WidgetID:      productID,
			TransactionID: txnID,
			CustomerID:    customerID,
			StatusID:      1,
			Quantity:      1,
			Amount:        amount,
		}
		_, err = app.SaveOrder(order)
		if err != nil {
			app.errorLog.Println("Unable to save order", err)
			return
		}
	}

	resp := jsonResponse{
		OK:      okey,
		Message: txnMsg,
	}

	out, err := json.MarshalIndent(resp, "", " ")
	if err != nil {
		app.errorLog.Println("Cannot encode response", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

// save a customer and return an id
func (app *application) SaveCustomer(first_name, last_name, email string) (int, error) {
	customer := models.Customer{
		FirstName: first_name,
		LastName:  last_name,
		Email:     email,
	}

	id, err := app.DB.InsertCustomer(customer)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// save a transaction and return an id
func (app *application) SaveTransaction(txn models.Transaction) (int, error) {
	id, err := app.DB.InsertTransaction(txn)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// save a order and return an id
func (app *application) SaveOrder(order models.Order) (int, error) {
	id, err := app.DB.InsertOrder(order)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (app *application) CreateAuthToken(w http.ResponseWriter, r *http.Request) {
	var userInput struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &userInput)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	// get user from database by emai send error if the email is invalid
	user, err := app.DB.GetUserByEmail(userInput.Email)
	if err != nil {
		app.invalidCredentials(w)
		return
	}

	// validate the password send error if the password is invalid
	validPassword, err := app.passwordMatch(user.Password, userInput.Password)
	if err != nil || !validPassword {
		app.invalidCredentials(w)
		return
	}

	// generate token
	token, err := models.GenerateToken(user.ID, 24*time.Hour, models.ScopeAuthentication)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	// save to database
	err = app.DB.InsertToken(token, user)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	var payload struct {
		Error   bool          `json:"error"`
		Message string        `json:"message"`
		Token   *models.Token `json:"authentication_token"`
	}
	payload.Error = false
	payload.Message = fmt.Sprintf("token for %s is created", userInput.Email)
	payload.Token = token

	_ = app.writeJSON(w, http.StatusOK, payload)
}

func (app *application) authenticateToken(r *http.Request) (*models.User, error) {
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		return nil, errors.New("no authorization header")
	}

	headerParts := strings.Split(authorizationHeader, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		return nil, errors.New("no authorization header")
	}

	token := headerParts[1]
	if len(token) != 26 {
		return nil, errors.New("invalid token")
	}

	// get the user from tokens table
	user, err := app.DB.GetUserForToken(token)
	if err != nil {
		return nil, errors.New("invalid token")
	}
	return user, nil
}

func (app *application) CheckAuthentication(w http.ResponseWriter, r *http.Request) {
	// validate the token
	user, err := app.authenticateToken(r)
	if err != nil {
		app.invalidCredentials(w)
		return
	}

	// valid user
	var payload struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}
	payload.Error = false
	payload.Message = fmt.Sprintf("authenticated user %s", user.Email)
	app.writeJSON(w, http.StatusOK, payload)
}
