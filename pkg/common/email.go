package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/mail"
	"net/smtp"

	"github.com/NKTKLN/todo-api/models"
	"github.com/NKTKLN/todo-api/pkg/db"
	"github.com/scorredoira/email"
)

type EmailAuthData struct {
	email    string
	password string
	server   string
	port     int
}

type EmailProvider interface {
	UserEmailVerification(context.Context, db.RedisClient, models.UserData) error
	UserPasswordReset(context.Context, db.RedisClient, string) error
	UserEmailReset(context.Context, db.RedisClient, string, int) error
}

// Creating new service for email
func NewEmailProvider(senderEmail, emailPassword, emailServer string, emailServerPort int) EmailProvider {
	return &EmailAuthData{
		email:    senderEmail,
		password: emailPassword,
		server:   emailServer,
		port:     emailServerPort,
	}
}

func (d *EmailAuthData) UserEmailVerification(ctx context.Context, client db.RedisClient, data models.UserData) (err error) {
	// Convert data to json
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	// Create a temporary access key
	key, err := client.AddEmailData(ctx, jsonData)
	if err != nil {
		return
	}

	// Generate message from template
	htmlTemplate := template.Must(template.ParseFiles("templates/email_verification.html"))
	buf := new(bytes.Buffer)
	if err = htmlTemplate.Execute(buf, map[string]string{"verificationCode": key}); err != nil {
		return
	}

	// Sending an email verification code to a user
	return d.sendEmail(data.Email, "Login confirmation", buf.String())
}

func (d *EmailAuthData) UserPasswordReset(ctx context.Context, client db.RedisClient, userEmail string) (err error) {
	// Create a temporary access key
	key, err := client.AddEmailData(ctx, userEmail)
	if err != nil {
		return
	}

	// Generate message from template
	htmlTemplate := template.Must(template.ParseFiles("templates/passwors_reset.html"))
	buf := new(bytes.Buffer)
	if err = htmlTemplate.Execute(buf, map[string]string{"verificationCode": key}); err != nil {
		return
	}

	// Sending an email with password reset code to a user
	return d.sendEmail(userEmail, "Reset password", buf.String())
}

func (d *EmailAuthData) UserEmailReset(ctx context.Context, client db.RedisClient, userEmail string, userId int) (err error) {
	// Convert data to json
	jsonData, err := json.Marshal(models.Users{Id: userId, Email: userEmail})
	if err != nil {
		return
	}

	// Create a temporary access key
	key, err := client.AddEmailData(ctx, jsonData)
	if err != nil {
		return
	}

	// Generate message from template
	htmlTemplate := template.Must(template.ParseFiles("templates/change_email.html"))
	buf := new(bytes.Buffer)
	if err = htmlTemplate.Execute(buf, map[string]string{"verificationCode": key}); err != nil {
		return
	}

	// Sending an email with email verification code to a user
	return d.sendEmail(userEmail, "Changing email", buf.String())
}

func (d *EmailAuthData) sendEmail(userEmail, subject, body string) error {
	message := email.NewHTMLMessage(subject, body)
	message.From = mail.Address{
		Name:    "ToDo",
		Address: d.email,
	}
	message.To = []string{userEmail}

	smtpAuth := smtp.PlainAuth("",
		d.email,
		d.password,
		d.server,
	)

	return email.Send(fmt.Sprintf("%s:%d", d.server, d.port), smtpAuth, message)
}
