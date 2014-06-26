package mailer

import (
	"net/mail"
	"net/smtp"
	"strings"
	"testing"
)

var (
	testMailer = NewMailer("host", "username", "password", 25)
	testHeader = map[string][]string{
		"Mime-Version": {"1.0"},
		"Date":         {"25 Jun 14 17:46 UTC"},
		"From":         {"From <from@example.com>"},
		"To":           {"To <to@example.com>"},
		"Cc":           {"Cc <cc@example.com>"},
		"Bcc":          {"Bcc <bcc@example.com>", "Bcc2 <bcc2@example.com>"},
		"Content-Type": {"text/plain"},
		"Subject":      {"Hello!"},
	}
	testBody = strings.NewReader("This is a test message.")
	expected = []struct {
		addr string
		from string
		to   []string
		msg  string
	}{
		{
			"host:25",
			"from@example.com",
			[]string{"to@example.com", "cc@example.com"},

			"Mime-Version: 1.0\r\n" +
				"Date: 25 Jun 14 17:46 UTC\r\n" +
				"From: From <from@example.com>\r\n" +
				"To: To <to@example.com>\r\n" +
				"Cc: Cc <cc@example.com>\r\n" +
				"Content-Type: text/plain\r\n" +
				"Subject: Hello!\r\n" +
				"\r\n" +
				"This is a test message.",
		},
		{
			"host:25",
			"from@example.com",
			[]string{"bcc@example.com"},

			"Mime-Version: 1.0\r\n" +
				"Date: 25 Jun 14 17:46 UTC\r\n" +
				"From: From <from@example.com>\r\n" +
				"To: To <to@example.com>\r\n" +
				"Cc: Cc <cc@example.com>\r\n" +
				"Bcc: Bcc <bcc@example.com>\r\n" +
				"Content-Type: text/plain\r\n" +
				"Subject: Hello!\r\n" +
				"\r\n" +
				"This is a test message.",
		},
		{
			"host:25",
			"from@example.com",
			[]string{"bcc2@example.com"},

			"Mime-Version: 1.0\r\n" +
				"Date: 25 Jun 14 17:46 UTC\r\n" +
				"From: From <from@example.com>\r\n" +
				"To: To <to@example.com>\r\n" +
				"Cc: Cc <cc@example.com>\r\n" +
				"Bcc: Bcc2 <bcc2@example.com>\r\n" +
				"Content-Type: text/plain\r\n" +
				"Subject: Hello!\r\n" +
				"\r\n" +
				"This is a test message.",
		},
	}
)

func TestMessage(t *testing.T) {
	i := 0
	sendMail = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		if i > len(expected) {
			t.Fatalf("Only %d mails should be sent", len(expected))
		}
		want := expected[i]
		if addr != want.addr {
			t.Errorf("Invalid address, got %q, want %q", addr, want.addr)
		}
		if from != want.from {
			t.Errorf("Invalid from, got %q, want %q", from, want.from)
		}
		gotTo := strings.Join(to, ", ")
		wantTo := strings.Join(want.to, ", ")
		if gotTo != wantTo {
			t.Errorf("Invalid recipient, got %q, want %q", gotTo, wantTo)
		}
		gotMsg := string(msg)
		if gotMsg != want.msg {
			t.Errorf("Invalid message body, got:\r\n%s\r\nwant:\r\n%s\r\n", gotMsg, want.msg)
		}
		i++

		return nil
	}
	err := testMailer.Send(&mail.Message{Header: testHeader, Body: testBody})
	if err != nil {
		t.Error(err)
	}
}
