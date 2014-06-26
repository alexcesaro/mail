// Package gomail provides a simple interface to easily write and send emails.
//
// Example:
//
//	package main
//
//	import (
//		"log"
//
//		"github.com/alexcesaro/mail/gomail"
//	)
//
//	func main() {
//		msg := gomail.NewMessage()
//		msg.SetAddressHeader("From", "alex@example.com", "Alex")
//		msg.SetHeader("To", "bob@example.com")
//		msg.AddHeader("To", "cora@example.com")
//		msg.SetHeader("Subject", "Hello!")
//		msg.SetBody("text/plain", "Hello Bob and Cora!")
//		msg.AddAlternative("text/html", "Hello <b>Bob</b> and <i>Cora</i>!")
//		if err := msg.Attach("/home/Alex/lolcat.jpg") {
//			log.Println(err)
//			return
//		}
//
//		m := gomail.NewMailer("smtp.example.com", "user", "123456", 25)
//		if err := m.Send(msg); err != nil { // This will send the email to Bob and Cora
//			log.Println(err)
//		}
//	}
package gomail

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/smtp"
	"path/filepath"
	"time"

	"github.com/alexcesaro/mail/mailer"
	"github.com/alexcesaro/mail/quotedprintable"
)

const (
	// QuotedPrintable represents the quoted-printable encoding as defined in
	// RFC 2045.
	QuotedPrintable = "quoted-printable"
	// Base64 represents the base64 encoding as defined in RFC 2045.
	Base64 = "base64"
)

// Message represents a mail message.
type Message struct {
	header      header
	parts       []part
	attachments []attachment
	charset     string
	encoding    string
	hEncoder    *quotedprintable.HeaderEncoder
}

type header map[string][]string

type part struct {
	contentType string
	body        *bytes.Buffer
}

type attachment struct {
	name    string
	content []byte
}

// NewCustomMessage creates a new message that will use the given encoding and
// charset.
func NewCustomMessage(charset, encoding string) *Message {
	var enc string
	if encoding == Base64 {
		enc = quotedprintable.B
	} else {
		enc = quotedprintable.Q
	}

	// No error will be thrown since we are using existing encodings
	encoder, _ := quotedprintable.NewHeaderEncoder(charset, enc)

	return &Message{
		header:      make(header),
		parts:       make([]part, 0),
		attachments: make([]attachment, 0),
		charset:     charset,
		encoding:    encoding,
		hEncoder:    encoder,
	}
}

// NewMessage creates a new UTF-8 message using quoted-printable encoding.
func NewMessage() *Message {
	return NewCustomMessage("UTF-8", QuotedPrintable)
}

// SetHeader sets a value to the given header field.
func (msg *Message) SetHeader(field, value string) {
	msg.header[field] = []string{msg.encodeHeader(value)}
}

// AddHeader adds a value to the given header field.
func (msg *Message) AddHeader(field, value string) {
	msg.header[field] = append(msg.header[field], msg.encodeHeader(value))
}

func (msg *Message) encodeHeader(value string) string {
	return msg.hEncoder.EncodeHeader(value)
}

// SetAddressHeader sets an address to the given header field.
func (msg *Message) SetAddressHeader(field, address, name string) {
	msg.header[field] = []string{msg.buildAddressHeader(address, name)}
}

// AddAddressHeader adds an address to the given header field.
func (msg *Message) AddAddressHeader(field, address, name string) {
	msg.header[field] = append(msg.header[field], msg.buildAddressHeader(address, name))
}

func (msg *Message) buildAddressHeader(address, name string) string {
	return msg.encodeHeader(name) + " <" + address + ">"
}

// SetDateHeader sets a date to the given header field.
func (msg *Message) SetDateHeader(field string, date time.Time) {
	msg.header[field] = []string{buildDateHeader(date)}
}

// AddDateHeader adds a date to the given header field.
func (msg *Message) AddDateHeader(field string, date time.Time) {
	msg.header[field] = append(msg.header[field], buildDateHeader(date))
}

func buildDateHeader(date time.Time) string {
	return date.Format(time.RFC822)
}

// GetHeader gets a header field.
func (msg *Message) GetHeader(field string) []string {
	return msg.header[field]
}

// DelHeader deletes a header field.
func (msg *Message) DelHeader(field string) {
	delete(msg.header, field)
}

// SetBody sets the body of the message.
func (msg *Message) SetBody(contentType, body string) {
	msg.parts = []part{part{contentType, bytes.NewBufferString(body)}}
}

// AddAlternative adds an alternative body to the message. Usually used to
// provide both an HTML and a text version of the message.
func (msg *Message) AddAlternative(contentType, body string) {
	msg.parts = append(msg.parts, part{contentType, bytes.NewBufferString(body)})
}

// GetBodyWriter gets a writer that writes to the body. It can be useful with
// the templates from packages text/template or html/template.
//
// Example:
//
//	w := msg.GetBodyWriter("text/plain")
//	t := template.Must(template.New("example").Parse("Hello {{.}}!"))
//	t.Execute(w, "Bob")
func (msg *Message) GetBodyWriter(contentType string) io.Writer {
	buf := new(bytes.Buffer)
	msg.parts = append(msg.parts, part{contentType, buf})

	return buf
}

// Attach attaches a file to the message.
func (msg *Message) Attach(filename string) error {
	content, err := readFile(filename)
	if err != nil {
		return err
	}
	msg.attachments = append(msg.attachments, attachment{filepath.Base(filename), content})

	return nil
}

// Stubbed out for testing.
var readFile = ioutil.ReadFile

// A Mailer represents an SMTP server.
type Mailer struct {
	m *mailer.Mailer
}

// NewMailer returns a mailer. The given parameters are used to connect to the
// SMTP server via a PLAIN authentication mechanism.
func NewMailer(host string, username string, password string, port int) Mailer {
	return Mailer{m: mailer.NewMailer(host, username, password, port)}
}

// NewCustomMailer creates a mailer using any authentication mechanism.
func NewCustomMailer(auth smtp.Auth, addr string) Mailer {
	return Mailer{m: mailer.NewCustomMailer(auth, addr)}
}

// Send sends the emails to the recipients of the message.
func (m Mailer) Send(message *Message) error {
	msg, err := message.Export()
	if err != nil {
		return nil
	}

	return m.m.Send(msg)
}
