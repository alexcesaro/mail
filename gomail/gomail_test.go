package gomail

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"net/mail"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestMessage(t *testing.T) {
	msg := NewMessage()
	msg.SetAddressHeader("From", "from@example.com", "Señor From")
	msg.SetAddressHeader("To", "to@example.com", "Señor To")
	msg.AddAddressHeader("To", "tobis@example.com", "Señor To Bis")
	msg.SetDateHeader("Date", stubNow())
	msg.AddDateHeader("X-Add-Date", stubNow())
	msg.AddDateHeader("X-Add-Date", stubNow())
	msg.AddHeader("X-Add-Text", "cofee")
	msg.AddHeader("X-Add-Text", "café")
	msg.SetHeader("Subject", "¡Hola, señor!")
	msg.SetBody("text/plain", "¡Hola, señor!")

	header := mail.Header{
		"Mime-Version":              {"1.0"},
		"From":                      {"=?UTF-8?Q?Se=C3=B1or_From?= <from@example.com>"},
		"To":                        {"=?UTF-8?Q?Se=C3=B1or_To?= <to@example.com>", "=?UTF-8?Q?Se=C3=B1or_To_Bis?= <tobis@example.com>"},
		"Date":                      {"25 Jun 14 17:46 UTC"},
		"X-Add-Date":                {"25 Jun 14 17:46 UTC", "25 Jun 14 17:46 UTC"},
		"X-Add-Text":                {"cofee", "=?UTF-8?Q?caf=C3=A9?="},
		"Subject":                   {"=?UTF-8?Q?=C2=A1Hola,_se=C3=B1or!?="},
		"Content-Type":              {"text/plain; charset=UTF-8"},
		"Content-Transfer-Encoding": {"quoted-printable"},
	}

	testMessage(t, msg, header, "=C2=A1Hola, se=C3=B1or!")
}

func TestCustomMessage(t *testing.T) {
	msg := NewCustomMessage("ISO-8859-1", Base64)
	msg.AddHeader("Subject", "café")
	msg.SetBody("text/html", "¡Hola, señor!")

	header := mail.Header{
		"Mime-Version":              {"1.0"},
		"Date":                      {"25 Jun 14 17:46 UTC"},
		"Subject":                   {"=?ISO-8859-1?B?Y2Fmw6k=?="},
		"Content-Type":              {"text/html; charset=ISO-8859-1"},
		"Content-Transfer-Encoding": {"base64"},
	}

	testMessage(t, msg, header, "wqFIb2xhLCBzZcOxb3Ih")
}

func TestEmpty(t *testing.T) {
	msg := NewMessage()

	header := mail.Header{
		"Mime-Version": {"1.0"},
		"Date":         {"25 Jun 14 17:46 UTC"},
	}

	testMessage(t, msg, header, "")
}

func TestAlternative(t *testing.T) {
	msg := NewMessage()
	msg.SetBody("text/plain", "¡Hola, señor!")
	msg.AddAlternative("text/html", "¡<b>Hola</b>, <i>señor</i>!</h1>")

	boundary := getMainBoundary(t, msg)

	header := mail.Header{
		"Mime-Version": {"1.0"},
		"Date":         {"25 Jun 14 17:46 UTC"},
		"Content-Type": {"multipart/alternative; boundary=" + boundary},
	}
	body := "--" + boundary + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"\r\n" +
		"=C2=A1Hola, se=C3=B1or!\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"\r\n" +
		"=C2=A1<b>Hola</b>, <i>se=C3=B1or</i>!</h1>\r\n" +
		"--" + boundary + "--\r\n"

	testMessage(t, msg, header, body)
}

func TestAttachment(t *testing.T) {
	readFile = stubReadFile

	msg := NewMessage()
	msg.SetBody("text/plain", "Test")
	msg.Attach("/tmp/test.pdf")

	boundary := getMainBoundary(t, msg)

	header := mail.Header{
		"Mime-Version": {"1.0"},
		"Date":         {"25 Jun 14 17:46 UTC"},
		"Content-Type": {"multipart/mixed; boundary=" + boundary},
	}
	body := "--" + boundary + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"\r\n" +
		"Test\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Type: application/pdf; name=\"test.pdf\"\r\n" +
		"Content-Disposition: attachment; filename=\"test.pdf\"\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"\r\n" +
		base64.StdEncoding.EncodeToString([]byte("Content of test.pdf")) + "\r\n" +
		"--" + boundary + "--\r\n"

	testMessage(t, msg, header, body)
}

func TestAttachmentOnly(t *testing.T) {
	readFile = stubReadFile

	msg := NewMessage()
	msg.Attach("/tmp/test.pdf")

	header := mail.Header{
		"Mime-Version":              {"1.0"},
		"Date":                      {"25 Jun 14 17:46 UTC"},
		"Content-Type":              {"application/pdf; name=\"test.pdf\""},
		"Content-Disposition":       {"attachment; filename=\"test.pdf\""},
		"Content-Transfer-Encoding": {"base64"},
	}
	body := base64.StdEncoding.EncodeToString([]byte("Content of test.pdf"))

	testMessage(t, msg, header, body)
}

func TestMultipleAttachment(t *testing.T) {
	readFile = stubReadFile

	msg := NewMessage()
	msg.SetBody("text/plain", "Test")
	msg.Attach("/tmp/test.pdf")
	msg.Attach("/tmp/test.zip")

	boundary := getMainBoundary(t, msg)

	header := mail.Header{
		"Mime-Version": {"1.0"},
		"Date":         {"25 Jun 14 17:46 UTC"},
		"Content-Type": {"multipart/mixed; boundary=" + boundary},
	}
	body := "--" + boundary + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"\r\n" +
		"Test\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Type: application/pdf; name=\"test.pdf\"\r\n" +
		"Content-Disposition: attachment; filename=\"test.pdf\"\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"\r\n" +
		base64.StdEncoding.EncodeToString([]byte("Content of test.pdf")) + "\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Type: application/zip; name=\"test.zip\"\r\n" +
		"Content-Disposition: attachment; filename=\"test.zip\"\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"\r\n" +
		base64.StdEncoding.EncodeToString([]byte("Content of test.zip")) + "\r\n" +
		"--" + boundary + "--\r\n"

	testMessage(t, msg, header, body)
}

func TestMultipleAttachmentOnly(t *testing.T) {
	readFile = stubReadFile

	msg := NewMessage()
	msg.Attach("/tmp/test.pdf")
	msg.Attach("/tmp/test.zip")

	boundary := getMainBoundary(t, msg)

	header := mail.Header{
		"Mime-Version": {"1.0"},
		"Date":         {"25 Jun 14 17:46 UTC"},
		"Content-Type": {"multipart/mixed; boundary=" + boundary},
	}
	body := "--" + boundary + "\r\n" +
		"Content-Type: application/pdf; name=\"test.pdf\"\r\n" +
		"Content-Disposition: attachment; filename=\"test.pdf\"\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"\r\n" +
		base64.StdEncoding.EncodeToString([]byte("Content of test.pdf")) + "\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Type: application/zip; name=\"test.zip\"\r\n" +
		"Content-Disposition: attachment; filename=\"test.zip\"\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"\r\n" +
		base64.StdEncoding.EncodeToString([]byte("Content of test.zip")) + "\r\n" +
		"--" + boundary + "--\r\n"

	testMessage(t, msg, header, body)
}

func TestFullMessage(t *testing.T) {
	readFile = stubReadFile

	msg := NewMessage()
	msg.SetBody("text/plain", "¡Hola, señor!")
	msg.AddAlternative("text/html", "¡<b>Hola</b>, <i>señor</i>!</h1>")
	msg.Attach("/tmp/test.pdf")

	mainBoundary := getMainBoundary(t, msg)
	subBoundary := getBodyBoundary(t, msg)

	header := mail.Header{
		"Mime-Version": {"1.0"},
		"Date":         {"25 Jun 14 17:46 UTC"},
		"Content-Type": {"multipart/mixed; boundary=" + mainBoundary},
	}
	body := "--" + mainBoundary + "\r\n" +
		"Content-Type: multipart/alternative; boundary=" + subBoundary + "\r\n" +
		"\r\n" +
		"--" + subBoundary + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"\r\n" +
		"=C2=A1Hola, se=C3=B1or!\r\n" +
		"--" + subBoundary + "\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"\r\n" +
		"=C2=A1<b>Hola</b>, <i>se=C3=B1or</i>!</h1>\r\n" +
		"--" + subBoundary + "--\r\n" +
		"\r\n" +
		"--" + mainBoundary + "\r\n" +
		"Content-Type: application/pdf; name=\"test.pdf\"\r\n" +
		"Content-Disposition: attachment; filename=\"test.pdf\"\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"\r\n" +
		base64.StdEncoding.EncodeToString([]byte("Content of test.pdf")) + "\r\n" +
		"--" + mainBoundary + "--\r\n"

	testMessage(t, msg, header, body)
}

func TestQpLineLength(t *testing.T) {
	msg := NewMessage()
	msg.SetBody("text/plain",
		strings.Repeat("0", 79)+"\r\n"+
			strings.Repeat("0", 78)+"à\r\n"+
			strings.Repeat("0", 77)+"à\r\n"+
			strings.Repeat("0", 76)+"à\r\n"+
			strings.Repeat("0", 75)+"à\r\n"+
			strings.Repeat("0", 78)+"\r\n"+
			strings.Repeat("0", 79)+"\n")

	header := mail.Header{
		"Mime-Version":              {"1.0"},
		"Date":                      {"25 Jun 14 17:46 UTC"},
		"Content-Type":              {"text/plain; charset=UTF-8"},
		"Content-Transfer-Encoding": {"quoted-printable"},
	}
	body := strings.Repeat("0", 78) + "=\r\n0\r\n" +
		strings.Repeat("0", 78) + "=\r\n=C3=A0\r\n" +
		strings.Repeat("0", 77) + "=\r\n=C3=A0\r\n" +
		strings.Repeat("0", 76) + "=\r\n=C3=A0\r\n" +
		strings.Repeat("0", 75) + "=C3=\r\n=A0\r\n" +
		strings.Repeat("0", 78) + "\r\n" +
		strings.Repeat("0", 78) + "=\r\n0\n"

	testMessage(t, msg, header, body)
}

func TestBase64LineLength(t *testing.T) {
	msg := NewCustomMessage("UTF-8", Base64)
	msg.SetBody("text/plain", strings.Repeat("0", 58))

	header := mail.Header{
		"Mime-Version":              {"1.0"},
		"Date":                      {"25 Jun 14 17:46 UTC"},
		"Content-Type":              {"text/plain; charset=UTF-8"},
		"Content-Transfer-Encoding": {"base64"},
	}
	body := strings.Repeat("MDAw", 19) + "MA\r\n=="

	testMessage(t, msg, header, body)
}

func testMessage(t *testing.T, msg *Message, header mail.Header, body string) {
	m := export(t, msg)

	want := &mail.Message{Header: header, Body: strings.NewReader(body)}
	for key := range want.Header {
		if _, ok := m.Header[key]; !ok {
			t.Errorf("Missing header: %q", key)
		} else {
			gotHeader := strings.Join(m.Header[key], ", ")
			wantHeader := strings.Join(m.Header[key], ", ")
			if gotHeader != wantHeader {
				t.Errorf("Invalid header %q, got: %q, want: %q", key, gotHeader, wantHeader)
			}
		}
	}
	for key := range m.Header {
		if _, ok := want.Header[key]; !ok {
			t.Errorf("Header %q should not be set", key)
		}
	}

	var (
		got, expected []byte
		err           error
	)
	if got, err = ioutil.ReadAll(m.Body); err != nil {
		t.Error(err)
	}
	if expected, err = ioutil.ReadAll(want.Body); err != nil {
		t.Error(err)
	}
	if string(got) != string(expected) {
		t.Errorf("Message body is not valid, got:\n%s\nwant:\n%s", got, expected)
	}
	lastExportedMessage = nil
}

func export(t *testing.T, msg *Message) *mail.Message {
	if lastExportedMessage == nil {
		now = stubNow
		var err error
		if lastExportedMessage, err = msg.Export(); err != nil {
			t.Errorf("Export should not return an error, got error %v", err)
		}
	}

	return lastExportedMessage
}

var lastExportedMessage *mail.Message

func getMainBoundary(t *testing.T, msg *Message) string {
	m := export(t, msg)
	h := m.Header["Content-Type"]
	if h == nil {
		t.Fatal(`Message does not contain header "Content-Type"`)
		return ""
	}
	contentType := strings.Join(h, ", ")

	if matches := regexp.MustCompile("boundary=(\\w+)").FindStringSubmatch(contentType); matches != nil {
		return matches[1]
	}

	t.Fatalf("Boundary not found in: %s", contentType)
	return ""
}

func getBodyBoundary(t *testing.T, msg *Message) string {
	m := export(t, msg)
	body, _ := ioutil.ReadAll(m.Body)
	m.Body = bytes.NewBuffer(body)
	if matches := regexp.MustCompile("boundary=(\\w+)").FindSubmatch(body); matches != nil {
		return string(matches[1])
	}

	t.Fatal("Boundary not found in body")
	return ""
}

func stubReadFile(filename string) ([]byte, error) {
	return []byte("Content of " + filepath.Base(filename)), nil
}

func stubNow() time.Time {
	return time.Date(2014, 06, 25, 17, 46, 0, 0, time.UTC)
}
