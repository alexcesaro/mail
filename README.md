# Deprecated

This package is deprecated, use these new repositories instead:
 * Gomail: https://github.com/go-gomail/gomail
 * quotedprintable: https://github.com/alexcesaro/quotedprintable
 * mailer has been merged into Gomail

The new version of Gomail has new features, better performances and a simpler
API!

Here is the list of breaking changes and how to update your code:

## Import

Before:

    import "github.com/alexcesaro/mail/gomail"

After:

    import "gopkg.in/gomail.v1"


## Add*Header functions

Functions AddHeader, AddAddressHeader and AddDateHeader were removed.

Before:

    msg.SetHeader("To", "alex@example.com")
    msg.AddHeader("To", "bob@example.com")

After:

    msg.SetHeader("From", "alex@example.com", "bob@example.com")

To replace AddAddressHeader and AddDateHeader you can use the new functions FormatAddress and FormatDate with SetHeader or SetHeaders.


## Attach a file

Before:

    if err := msg.Attach("/home/Alex/lolcat.jpg"); err != nil {
        panic(err)
    }

After:

    f, err := gomail.OpenFile("/home/Alex/lolcat.jpg")
    if err != nil {
        panic(err)
    }
    msg.Attach(f)


## NewCustomMessage

Before:

    msg := NewMessage("ISO-8859-1", Base64)

After:

    msg := NewMessage(SetCharset("ISO-8859-1"), SetEncoding(Base64))


## Export

Before:

    m, err := msg.Export()
    if err != nil {
        panic(err)
    }

After:

    m := msg.Export()


## Constants

Constants `QuotedPrintable` and `Base64` had type `string` and now have type
`Encoding`. If you just used these constants in NewCustomMessage there is
nothing to do.


----

This repository contains mail packages for Go:
 - [gomail](#gomail) is the main package of this repository, it provides a
   simple interface to easily write and send emails.
 - [mailer](#mailer) provides functions to easily send emails. It should be used
   with or inside a package that helps writing emails like it is done in gomail.
 - [quotedprintable](#quotedprintable) is a package that implements
   quoted-printable and message header encoding. Someday, it might enter the Go
   standard library.

You are more than welcome to ask questions on the [Go mailing-list](https://groups.google.com/d/topic/golang-nuts/ywPpNlmSt6U/discussion) and open issues here if you find bugs.


# gomail

[Documentation](http://godoc.org/github.com/alexcesaro/mail/gomail)

Package gomail provides a simple interface to easily write and send emails.

Example:

    package main

    import (
        "log"

        "github.com/alexcesaro/mail/gomail"
    )

    func main() {
        msg := gomail.NewMessage()
        msg.SetAddressHeader("From", "alex@example.com", "Alex")
        msg.SetHeader("To", "bob@example.com")
        msg.AddHeader("To", "cora@example.com")
        msg.SetHeader("Subject", "Hello!")
        msg.SetBody("text/plain", "Hello Bob and Cora!")
        msg.AddAlternative("text/html", "Hello <b>Bob</b> and <i>Cora</i>!")
        if err := msg.Attach("/home/Alex/lolcat.jpg"); err != nil {
            log.Println(err)
            return
        }

        m := gomail.NewMailer("smtp.example.com", "user", "123456", 25)
        if err := m.Send(msg); err != nil { // This will send the email to Bob and Cora
            log.Println(err)
        }
    }


# mailer

[Documentation](http://godoc.org/github.com/alexcesaro/mail/mailer)

Package mailer provides functions to easily send emails.

This package can be used as a standalone but if you want to send emails with
non-ASCII characters or with attachment you should use it with or inside a
package that helps writing emails like it is done in gomail.

    package main

    import (
        "log"
        "net/mail"
        "strings"

        "github.com/alexcesaro/mail/mailer"
    )

    func main() {
        msg := &mail.Message{
            mail.Header{
                "From":         {"alex@example.com"},
                "To":           {"bob@example.com", "cora@example.com"},
                "Subject":      {"Hello!"},
                "Content-Type": {"text/plain"},
            },
            strings.NewReader("Hello, how are you ?"),
        }

        m := mailer.NewMailer("smtp.example.com", "user", "123456", 25)
        if err := m.Send(msg); err != nil { // This will send the email to Bob and Cora
            log.Println(err)
        }
    }


# quotedprintable

[Documentation](http://godoc.org/github.com/alexcesaro/mail/quotedprintable)

Package quotedprintable implements quoted-printable and message header encoding
as specified by RFC 2045 and RFC 2047.

Someday, it might enter the Go standard library. See
[this post](https://groups.google.com/d/topic/golang-dev/PK_ICQNJTmg/discussion)
on the golang-dev mailing-list or
[this code review](https://codereview.appspot.com/101330049/) or
[issue 4943](https://code.google.com/p/go/issues/detail?id=4943) of the Go bug
tracker.
