// Package mailer provides functions to easily send emails.
//
// This package should be used with or inside a package that helps writing
// emails like it is done in the package github.com/alexcesaro/mail/gomail
//
//	package main
//
//	import (
//		"log"
//		"net/mail"
//		"strings"
//
//		"github.com/alexcesaro/mail/mailer"
//	)
//
//	func main() {
//		msg := &mail.Message{
//			mail.Header{
//				"From":         {"alex@example.com"},
//				"To":           {"bob@example.com", "cora@example.com"},
//				"Subject":      {"Hello!"},
//				"Content-Type": {"text/plain"},
//			},
//			strings.NewReader("Hello, how are you ?"),
//		}
//
//		m := mailer.NewMailer("smtp.example.com", "user", "123456", 25)
//		if err := m.Send(msg); err != nil { // This will send the email to Bob and Cora
//			log.Println(err)
//		}
//	}
package mailer

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/mail"
	"net/smtp"
	"strings"
)

// A Mailer represents an SMTP server.
type Mailer struct {
	auth smtp.Auth
	addr string
}

// NewMailer returns a mailer. The given parameters are used to connect to the
// SMTP server via a PLAIN authentication mechanism.
func NewMailer(host string, username string, password string, port int) *Mailer {
	return &Mailer{
		auth: smtp.PlainAuth("", username, password, host),
		addr: fmt.Sprintf("%s:%d", host, port),
	}
}

// NewCustomMailer creates a mailer using any authentication mechanism.
func NewCustomMailer(auth smtp.Auth, addr string) *Mailer {
	return &Mailer{auth, addr}
}

// Send sends the emails to the recipients of the message.
func (m *Mailer) Send(msg *mail.Message) error {
	from, err := getFrom(msg)
	if err != nil {
		return err
	}
	recipients, bcc := getRecipients(msg)

	h := flattenHeader(msg, "")
	body, err := ioutil.ReadAll(msg.Body)
	if err != nil {
		return err
	}

	mail := append(h, body...)
	if err := sendMail(m.addr, m.auth, from, recipients, mail); err != nil {
		return err
	}

	if len(bcc) != 0 {
		for _, to := range bcc {
			h = flattenHeader(msg, to)
			mail = append(h, body...)
			if err := sendMail(m.addr, m.auth, from, []string{to}, mail); err != nil {
				return err
			}
		}
	}

	return nil
}

func flattenHeader(msg *mail.Message, bcc string) []byte {
	var buffer bytes.Buffer
	for field, value := range msg.Header {
		if field != "Bcc" {
			buffer.WriteString(field + ": " + strings.Join(value, ", ") + "\r\n")
		} else if bcc != "" {
			for _, to := range value {
				if strings.Contains(to, bcc) {
					buffer.WriteString(field + ": " + to + "\r\n")
				}
			}
		}
	}
	buffer.WriteString("\r\n")

	return buffer.Bytes()
}

func getFrom(msg *mail.Message) (string, error) {
	field := msg.Header.Get("Sender")
	if field == "" {
		field = msg.Header.Get("From")
		if field == "" {
			return "", errors.New("mailer: invalid message, \"From\" field is absent")
		}
	}

	return parseAddress(field)
}

var destinationFields = []string{"Bcc", "To", "Cc"}

func getRecipients(msg *mail.Message) (recipients, bcc []string) {
	for _, field := range destinationFields {
		if values, ok := msg.Header[field]; ok {
			for _, value := range values {
				address, err := parseAddress(value)
				if err != nil {
					continue
				}
				if field == "Bcc" {
					if !isInList(address, bcc) {
						bcc = append(bcc, address)
					}
				} else if !isInList(address, bcc) && !isInList(address, recipients) {
					recipients = append(recipients, address)
				}
			}
		}
	}

	return recipients, bcc
}

func isInList(address string, list []string) bool {
	for _, addr := range list {
		if address == addr {
			return true
		}
	}

	return false
}

func parseAddress(field string) (string, error) {
	address, err := mail.ParseAddress(field)
	if address == nil {
		return "", err
	}

	return address.Address, err
}

// Stubbed out for testing.
var sendMail = smtp.SendMail
