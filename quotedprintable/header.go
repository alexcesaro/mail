// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file defines encoding and decoding functions for encoded-words
// as defined in RFC 2047.

package quotedprintable

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	// Q represents the Q-encoding defined in RFC 2047.
	Q = "Q"
	// B represents the Base64 encoding defined in RFC 2045.
	B = "B"
)

// HeaderEncoder in an encoder for encoded words.
type HeaderEncoder struct {
	charset    string
	encoding   string
	splitWords bool
}

const maxEncodedWordLen = 75 // As defined in RFC 2047, section 2

// StdHeaderEncoder is a RFC 2047 encoder for UTF-8 strings using Q encoding.
var StdHeaderEncoder = &HeaderEncoder{"UTF-8", Q, true}

// NewHeaderEncoder returns a new HeaderEncoder to encode strings in the
// specified charset using the encoding enc.
func NewHeaderEncoder(charset string, enc string) (*HeaderEncoder, error) {
	if strings.ToUpper(enc) != Q && strings.ToUpper(enc) != B {
		return nil, fmt.Errorf("quotedprintable: RFC 2047 encoding not supported: %q", enc)
	}

	// We automatically split encoded-words only when the charset is UTF-8
	// because since multi-octet character must not be split across adjacent
	// encoded-words (see RFC 2047, section 5) there is no way to split words
	// without knowing how the charset works.
	splitWords := strings.ToUpper(charset) == "UTF-8"

	return &HeaderEncoder{charset, enc, splitWords}, nil
}

// EncodeHeader encodes a string to be used as a MIME header value. It encodes
// the input only if it contains non-ASCII characters.
func (e *HeaderEncoder) EncodeHeader(s string) string {
	if !needsEncoding(s) {
		return s
	}

	return e.encodeWord(s)
}

func needsEncoding(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isVchar(s[i]) && !isWSP(s[i]) {
			return true
		}
	}

	return false
}

// encodeWord encodes a string into an encoded-word.
func (e *HeaderEncoder) encodeWord(s string) string {
	buf := new(bytes.Buffer)
	e.openWord(buf)
	if strings.ToUpper(e.encoding) == B {
		maxLen := maxEncodedWordLen - buf.Len() - 2
		if !e.splitWords || base64.StdEncoding.EncodedLen(len(s)) <= maxLen {
			buf.WriteString(base64.StdEncoding.EncodeToString([]byte(s)))
		} else {
			var n, last, runeSize int
			for i := 0; i < len(s); i += runeSize {
				runeSize = getRuneSize(s, i)

				if base64.StdEncoding.EncodedLen(n+runeSize) <= maxLen {
					n += runeSize
				} else {
					buf.WriteString(base64.StdEncoding.EncodeToString([]byte(s[last:i])))
					e.splitWord(buf)
					last = i
					n = runeSize
				}
			}
			buf.WriteString(base64.StdEncoding.EncodeToString([]byte(s[last:])))
		}
	} else {
		if !e.splitWords {
			for i := 0; i < len(s); i++ {
				writeQ(buf, s[i])
			}
		} else {
			var runeSize int
			n := buf.Len()
			for i := 0; i < len(s); i += runeSize {
				b := s[i]
				var encLen int
				if b == ' ' || (isVchar(b) && b != '=' && b != '?' && b != '_') {
					encLen, runeSize = 1, 1
				} else {
					runeSize = getRuneSize(s, i)
					encLen = 3 * runeSize
				}

				// We remove 2 to let spaces for closing chars "?="
				if n+encLen > maxEncodedWordLen-2 {
					n = e.splitWord(buf)
				}
				writeQString(buf, s[i:i+runeSize])
				n += encLen
			}
		}
	}
	e.closeWord(buf)

	return buf.String()
}

func (e *HeaderEncoder) openWord(buf *bytes.Buffer) int {
	buf.WriteString("=?")
	buf.WriteString(e.charset)
	buf.WriteByte('?')
	buf.WriteString(e.encoding)
	buf.WriteByte('?')

	return 4 + len(e.charset) + len(e.encoding)
}

func (e *HeaderEncoder) closeWord(buf *bytes.Buffer) {
	buf.WriteString("?=")
}

func (e *HeaderEncoder) splitWord(buf *bytes.Buffer) int {
	e.closeWord(buf)
	buf.WriteString("\r\n ")
	return e.openWord(buf)
}

func getRuneSize(s string, i int) int {
	runeSize := 1
	for i+runeSize < len(s) && !utf8.RuneStart(s[i+runeSize]) {
		runeSize++
	}

	return runeSize
}

func writeQString(buf *bytes.Buffer, s string) {
	for i := 0; i < len(s); i++ {
		writeQ(buf, s[i])
	}
}

func writeQ(buf *bytes.Buffer, b byte) {
	switch {
	case b == ' ':
		buf.WriteByte('_')
	case isVchar(b) && b != '=' && b != '?' && b != '_':
		buf.WriteByte(b)
	default:
		enc := make([]byte, 3)
		encodeByte(enc[0:3], b)
		buf.Write(enc)
	}
}

// DecodeHeader decodes a MIME header by decoding all encoded-words of the
// header. This function does not do any charset conversion, the returned text
// is encoded in the returned charset. So text is not necessarily encoded in
// UTF-8. As such, this function does not support decoding headers with multiple
// encoded-words using different charsets.
func DecodeHeader(header string) (text string, charset string, err error) {
	var buf bytes.Buffer
	for {
		i := strings.IndexByte(header, '=')
		if i == -1 {
			break
		}
		if i > 0 {
			buf.WriteString(header[:i])
			header = header[i:]
		}

		word := rfc2047.FindString(header)
		if word == "" {
			buf.WriteByte('=')
			header = header[1:]
			continue
		}

		for {
			dec, wordCharset, err := decodeWord(word)
			if err != nil {
				buf.WriteString(word)
				header = header[len(word):]
				break
			}
			if charset == "" {
				charset = wordCharset
			} else if charset != wordCharset {
				return "", "", fmt.Errorf("quotedprintable: multiple charsets in header are not supported: %q and %q used", charset, wordCharset)
			}
			buf.Write(dec)
			header = header[len(word):]

			// White-space and newline characters separating two encoded-words
			// must be deleted.
			var j int
			for j = 0; j < len(header) && (isWSP(header[j]) || isNewline(header[j])); j++ {
			}
			if j == 0 {
				// If there are no white-space characters following the current
				// encoded-word there is nothing special to do.
				break
			}
			word = rfc2047.FindString(header[j:])
			if word == "" {
				break
			}
			header = header[j:]
		}
	}
	buf.WriteString(header)

	return buf.String(), charset, nil
}

var rfc2047 = regexp.MustCompile(`^=\?[\w\-]+\?[bBqQ]\?[^?]+\?=`)

func decodeWord(s string) (text []byte, charset string, err error) {
	fields := strings.Split(s, "?")
	if len(fields) != 5 || fields[0] != "=" || fields[4] != "=" || len(fields[2]) != 1 {
		return []byte(s), "", nil
	}

	charset, enc, src := fields[1], fields[2], fields[3]

	var dec []byte
	switch strings.ToUpper(enc) {
	case B:
		if dec, err = base64.StdEncoding.DecodeString(src); err != nil {
			return dec, charset, err
		}
	case Q:
		if dec, err = qDecode(src); err != nil {
			return dec, charset, err
		}
	default:
		return []byte(""), charset, fmt.Errorf("quotedprintable: RFC 2047 encoding not supported: %q", enc)
	}

	return dec, charset, nil
}

// qDecode decodes a Q encoded string.
func qDecode(s string) ([]byte, error) {
	dec := make([]byte, MaxDecodedLen(len(s)))

	n := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '_':
			dec[n] = ' '
		case c == '=':
			if i+2 >= len(s) {
				return dec[:n], io.ErrUnexpectedEOF
			}
			buf, err := readHexByte([]byte(s[i+1:]))
			if err != nil {
				return dec[:n], err
			}
			dec[n] = buf
			i += 2
		case isVchar(c) || c == ' ' || c == '\n' || c == '\r' || c == '\t':
			dec[n] = c
		default:
			return dec[:n], fmt.Errorf("quotedprintable: invalid unescaped byte 0x%02x in Q encoded string", c)
		}
		n++
	}

	return dec[:n], nil
}
