package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/chaiyapluek/go-stripe/internal/cards"
	"github.com/chaiyapluek/go-stripe/internal/models"
	"github.com/go-chi/chi/v5"
)

// display the home page
func (app *application) Home(w http.ResponseWriter, r *http.Request) {

	// stringMap will be available in this template
	if err := app.renderTemplate(w, r, "home", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}

// display the virtual terminal page
func (app *application) VirtualTerminal(w http.ResponseWriter, r *http.Request) {

	// stringMap will be available in this template
	if err := app.renderTemplate(w, r, "terminal", &templateData{}, "stripe-js"); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) PaymentSucceeded(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	// read posted data
	first_name := r.Form.Get("first_name")
	last_name := r.Form.Get("last_name")
	email := r.Form.Get("cardholder_email")
	paymentIntent := r.Form.Get("payment_intent")
	paymentMethod := r.Form.Get("payment_method")
	paymentAmount := r.Form.Get("payment_amount")
	paymentCurrency := r.Form.Get("payment_currency")

	card := cards.Card{
		Secret: app.config.stripe.secret,
		Key:    app.config.stripe.key,
	}

	app.infoLog.Println("Payment Success: Payment Intent:", paymentIntent)
	app.infoLog.Println("Payment Success: Payment Method:", paymentMethod)

	pi, err := card.RetrievePaymentIntent(paymentIntent)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	pm, err := card.GetPaymentMethod(paymentMethod)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	last4 := pm.Card.Last4
	expiryMonth := pm.Card.ExpMonth
	expiryYear := pm.Card.ExpYear

	// Create a new Customer
	customerID, err := app.SaveCustomer(first_name, last_name, email)
	if err != nil {
		app.errorLog.Println("Failed to save customer:", err)
		return
	}
	app.infoLog.Println("CustomerID:", customerID)

	// Create a new transaction
	amount, err := strconv.Atoi(paymentAmount)
	if err != nil {
		app.errorLog.Println("Cannot convert payment amount:", err)
		return
	}
	txn := models.Transaction{
		Amount:              amount,
		Currency:            paymentCurrency,
		LastFour:            last4,
		ExpiryMonth:         int(expiryMonth),
		ExpiryYear:          int(expiryYear),
		BankReturnCode:      pi.Charges.Data[0].ID,
		TransactionStatusID: 2,
	}
	txnID, err := app.SaveTransaction(txn)
	if err != nil {
		app.errorLog.Println("Failed to save transaction:", err)
		return
	}
	app.infoLog.Println("TransactionID:", txnID)

	// Create a new order
	widgetID, err := strconv.Atoi(r.Form.Get("product_id"))
	if err != nil {
		app.errorLog.Println("Cannot convert product id:", err)
		return
	}
	order := models.Order{
		WidgetID:      widgetID,
		TransactionID: txnID,
		CustomerID:    customerID,
		StatusID:      1,
		Quantity:      1,
		Amount:        amount,
		CreateAt:      time.Now(),
		UpdateAt:      time.Now(),
	}
	_, err = app.SaveOrder(order)
	if err != nil {
		app.errorLog.Println("Failed to save order:", err)
		return
	}

	data := make(map[string]interface{})
	data["first_name"] = first_name
	data["last_name"] = last_name
	data["email"] = email
	data["pi"] = paymentIntent
	data["pm"] = paymentMethod
	data["pa"] = paymentAmount
	data["pc"] = paymentCurrency
	data["last_four"] = last4
	data["expiry_month"] = expiryMonth
	data["expiry_year"] = expiryYear
	data["bank_return_code"] = pi.Charges.Data[0].ID

	if err := app.renderTemplate(w, r, "succeeded", &templateData{
		Data: data,
	}); err != nil {
		app.errorLog.Println(err)
	}

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

func (app *application) ChargeOnce(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	widgetId, _ := strconv.Atoi(id)

	widget, err := app.DB.GetWidget(widgetId)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	data := make(map[string]interface{})
	data["widget"] = widget

	if err := app.renderTemplate(w, r, "buy-once", &templateData{
		Data: data,
	}, "stripe-js"); err != nil {
		app.errorLog.Println(err)
	}
}
