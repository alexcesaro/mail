package gomail

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"path/filepath"
	"time"

	"github.com/alexcesaro/mail/quotedprintable"
)

// Export converts the message into a net/mail.Message.
func (msg *Message) Export() (*mail.Message, error) {
	w := newMessageWriter(msg)

	if msg.isMixed() {
		w.openMultipart("mixed")
	}
	if msg.isAlternative() {
		w.openMultipart("alternative")
	}

	for _, part := range msg.parts {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", part.contentType+"; charset="+msg.charset)
		if msg.encoding == Base64 {
			h.Set("Content-Transfer-Encoding", Base64)
		} else {
			h.Set("Content-Transfer-Encoding", QuotedPrintable)
		}

		w.writeHeader(h)
		if err := w.writeBody(part.body.Bytes(), msg.encoding); err != nil {
			return nil, err
		}
	}
	if msg.isAlternative() {
		w.closeMultipart()
	}

	for _, attachment := range msg.attachments {
		mimeType := mime.TypeByExtension(filepath.Ext(attachment.name))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", mimeType+"; name=\""+attachment.name+"\"")
		h.Set("Content-Disposition", "attachment; filename=\""+attachment.name+"\"")
		h.Set("Content-Transfer-Encoding", Base64)

		w.writeHeader(h)
		if err := w.writeBody(attachment.content, Base64); err != nil {
			return nil, err
		}
	}
	if msg.isMixed() {
		w.closeMultipart()
	}

	return w.export(), nil
}

func (msg *Message) isMixed() bool {
	return (len(msg.parts) > 0 && len(msg.attachments) > 0) || len(msg.attachments) > 1
}

func (msg *Message) isAlternative() bool {
	return len(msg.parts) > 1
}

// messageWriter helps converting the message into a net/mail.Message
type messageWriter struct {
	header     mail.Header
	buf        *bytes.Buffer
	writers    [2]*multipart.Writer
	partWriter io.Writer
	depth      uint8
}

func newMessageWriter(msg *Message) *messageWriter {
	// We copy the header so Export does not modify the message
	header := make(mail.Header, len(msg.header))
	for k, v := range msg.header {
		header[k] = v
	}

	if _, ok := header["Mime-Version"]; !ok {
		header["Mime-Version"] = []string{"1.0"}
	}
	if _, ok := header["Date"]; !ok {
		header["Date"] = []string{buildDateHeader(now())}
	}

	return &messageWriter{header: header, buf: new(bytes.Buffer)}
}

// Stubbed out for testing.
var now = time.Now

func (w *messageWriter) openMultipart(mimeType string) {
	w.writers[w.depth] = multipart.NewWriter(w.buf)
	contentType := "multipart/" + mimeType + "; boundary=" + w.writers[w.depth].Boundary()

	if w.depth == 0 {
		w.header["Content-Type"] = []string{contentType}
	} else {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", contentType)
		w.createPart(h)
	}
	w.depth++
}

func (w *messageWriter) closeMultipart() {
	if w.depth > 0 {
		w.writers[w.depth-1].Close()
		w.depth--
	}
}

func (w *messageWriter) writeHeader(h textproto.MIMEHeader) {
	if w.depth == 0 {
		for field, value := range h {
			w.header[field] = value
		}
	} else {
		w.createPart(h)
	}
}

func (w *messageWriter) createPart(h textproto.MIMEHeader) {
	// No need to check the error since the underlying writer is a bytes.Buffer
	w.partWriter, _ = w.writers[w.depth-1].CreatePart(h)
}

func (w *messageWriter) writeBody(body []byte, encoding string) error {
	var subWriter io.Writer
	if w.depth == 0 {
		subWriter = w.buf
	} else {
		subWriter = w.partWriter
	}

	if encoding == Base64 {
		writer := base64.NewEncoder(base64.StdEncoding, newBase64LineWriter(subWriter))
		// No need to check the error since base64LineWriter never returns error
		if _, err := writer.Write(body); err != nil {
			return err
		}
		if err := writer.Close(); err != nil {
			return err
		}
	} else {
		writer := quotedprintable.NewEncoder(newQpLineWriter(subWriter))
		// No need to check the error since qpLineWriter never returns error
		if _, err := writer.Write(body); err != nil {
			return err
		}
	}

	return nil
}

func (w *messageWriter) export() *mail.Message {
	return &mail.Message{Header: w.header, Body: w.buf}
}

// As defined in RFC 5322, 2.1.1.
const maxLineLen = 78

// base64LineWriter limits text encoded in base64 to 78 characters per line
type base64LineWriter struct {
	w       io.Writer
	lineLen int
}

func newBase64LineWriter(w io.Writer) *base64LineWriter {
	return &base64LineWriter{w: w}
}

func (w *base64LineWriter) Write(p []byte) (int, error) {
	n := 0
	for len(p)+w.lineLen > maxLineLen {
		w.w.Write(p[:maxLineLen-w.lineLen])
		w.w.Write([]byte("\r\n"))
		p = p[maxLineLen-w.lineLen:]
		n += maxLineLen - w.lineLen
		w.lineLen = 0
	}

	w.w.Write(p)
	w.lineLen += len(p)

	return n + len(p), nil
}

// qpLineWriter limits text encoded in quoted-printable to 78 characters per
// line
type qpLineWriter struct {
	w       io.Writer
	lineLen int
}

func newQpLineWriter(w io.Writer) *qpLineWriter {
	return &qpLineWriter{w: w}
}

func (w *qpLineWriter) Write(p []byte) (int, error) {
	n := 0
	for len(p) > 0 {
		// If the text is not over the limit, write everything
		if len(p) < maxLineLen-w.lineLen {
			w.w.Write(p)
			w.lineLen += len(p)
			return n + len(p), nil
		}

		i := bytes.IndexAny(p[:maxLineLen-w.lineLen+2], "\n")
		// If there is a newline before the limit, write the end of the line
		if i != -1 && (i != maxLineLen-w.lineLen+1 || p[i-1] == '\r') {
			w.w.Write(p[:i+1])
			p = p[i+1:]
			n += i + 1
			w.lineLen = 0
			continue
		}

		// Quoted-printable text must not be cut between an equal sign and the
		// two following characters
		var toWrite int
		if maxLineLen-w.lineLen-2 >= 0 && p[maxLineLen-w.lineLen-2] == '=' {
			toWrite = maxLineLen - w.lineLen - 2
		} else if p[maxLineLen-w.lineLen-1] == '=' {
			toWrite = maxLineLen - w.lineLen - 1
		} else {
			toWrite = maxLineLen - w.lineLen
		}

		// Insert the newline where it is needed
		w.w.Write(p[:toWrite])
		w.w.Write([]byte("=\r\n"))
		p = p[toWrite:]
		n += toWrite
		w.lineLen = 0
	}

	return n, nil
}
