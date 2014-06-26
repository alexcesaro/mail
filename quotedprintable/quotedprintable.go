// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file defines quoted-printable decoders and encoders, as specified in RFC
// 2045.
// Deviations:
// 1. in addition to "=\r\n", "=\n" is also treated as soft line break.
// 2. it will pass through a '\r' or '\n' not preceded by '=', consistent
//    with other broken QP encoders & decoders.

// Package quotedprintable implements quoted-printable and message header encoding as
// specified by RFC 2045 and RFC 2047.
package quotedprintable

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// Encode encodes src into at most MaxEncodedLen(len(src)) bytes to dst,
// returning the actual number of bytes written to dst.
func Encode(dst, src []byte) (n int) {
	for i := 0; i < len(src); i++ {
		switch c := src[i]; {
		case c != '=' && (isVchar(c) || isNewline(c)):
			dst[n] = c
			n++
		case isWSP(c):
			if isLastChar(i, src) {
				encodeByte(dst[n:], c)
				n += 3
			} else {
				dst[n] = c
				n++
			}
		default:
			encodeByte(dst[n:], c)
			n += 3
		}
	}

	return n
}

// isLastChar returns true if byte i is the last character of the line.
func isLastChar(i int, src []byte) bool {
	if i == len(src)-1 ||
		(i < len(src)-1 && src[i+1] == '\n') ||
		(i < len(src)-2 && src[i+1] == '\r' && src[i+2] == '\n') {
		return true
	}

	return false
}

// encodeByte encodes a byte using the quoted-printable encoding.
func encodeByte(dst []byte, b byte) {
	dst[0] = '='
	dst[1] = hextable[b>>4]
	dst[2] = hextable[b&0x0f]
}

const hextable = "0123456789ABCDEF"

// EncodeToString returns the quoted-printable encoding of src.
func EncodeToString(src []byte) string {
	dbuf := make([]byte, MaxEncodedLen(len(src)))
	n := Encode(dbuf, src)
	return string(dbuf[:n])
}

// MaxEncodedLen returns the maximum length of an encoding of n source bytes.
func MaxEncodedLen(n int) int { return 3 * n }

// NewEncoder returns a new quoted-printable stream encoder. Data written to the
// returned writer will be encoded and then written to w.
func NewEncoder(w io.Writer) io.Writer {
	return &encoder{w}
}

type encoder struct {
	w io.Writer
}

func (e *encoder) Write(p []byte) (int, error) {
	dbuf := make([]byte, MaxEncodedLen(len(p)))
	n := Encode(dbuf, p)
	n, err := e.w.Write(dbuf[:n])
	if err != nil {
		nn := 0
		for i := 0; i < n; i++ {
			if dbuf[i] == '=' {
				if i+2 >= n {
					break
				}
				i += 2
			}
			nn++
		}
		return nn, err
	}

	return len(p), nil
}

// Decode decodes src into at most MaxDecodedLen(len(src)) bytes to dst,
// returning the actual number of bytes written to dst.
func Decode(dst, src []byte) (n int, err error) {
	var eol, trimLen, eolLen int
	for i := 0; i < len(src); i++ {
		if i == eol {
			eol = bytes.IndexByte(src[i:], '\n') + i + 1
			if eol == i {
				eol = len(src)
				eolLen = 0
			} else if eol-2 >= i && src[eol-2] == '\r' {
				eolLen = 2
			} else {
				eolLen = 1
			}

			// Count the number of bytes to trim
			trimLen = 0
			for {
				if trimLen == eol-eolLen-i {
					break
				}

				switch src[eol-eolLen-trimLen-1] {
				case '\n', '\r', ' ', '\t':
					trimLen++
					continue
				case '=':
					if trimLen > 0 {
						trimLen += eolLen + 1
						eolLen = 0
						err = fmt.Errorf("quotedprintable: invalid bytes after =: %q", src[eol-trimLen+1:eol])
					} else {
						trimLen = eolLen + 1
						eolLen = 0
					}
				}
				break
			}
		}

		// Skip trimmable bytes
		if trimLen > 0 && i == eol-trimLen-eolLen {
			if err != nil {
				return n, err
			}

			i += trimLen - 1
			continue
		}

		switch c := src[i]; {
		case c == '=':
			if i+2 >= len(src) {
				return n, io.ErrUnexpectedEOF
			}
			b, convErr := readHexByte(src[i+1:])
			if convErr != nil {
				return n, convErr
			}
			dst[n] = b
			n++
			i += 2
		case (c >= ' ' && c <= '~') || c == '\n' || c == '\r' || c == '\t':
			dst[n] = c
			n++
		default:
			return n, fmt.Errorf("quotedprintable: invalid unescaped byte 0x%02x in quoted-printable body", c)
		}
	}

	return n, nil
}

// MaxDecodedLen returns the maximum length of a decoding of n source bytes.
func MaxDecodedLen(n int) int { return n }

// DecodeString returns the bytes represented by the quoted-printable string s.
func DecodeString(s string) ([]byte, error) {
	dbuf := make([]byte, MaxDecodedLen(len(s)))
	n, err := Decode(dbuf, []byte(s))
	return dbuf[:n], err
}

// NewDecoder returns a new quoted-printable stream decoder.
func NewDecoder(r io.Reader) io.Reader {
	return &qpReader{br: bufio.NewReader(r)}
}

type qpReader struct {
	br   *bufio.Reader
	line []byte
	eof  bool
	err  error
}

func (q *qpReader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		if len(q.line) == 0 {
			if q.err != nil {
				return n, q.err
			} else if q.eof {
				return n, io.EOF
			}

			q.line, q.err = q.br.ReadSlice('\n')
			if q.err == io.EOF {
				q.eof = true
			} else if q.err != nil {
				return n, q.err
			}

			var nn int
			nn, q.err = Decode(q.line, q.line)
			q.line = q.line[:nn]
		}

		nn := copy(p[n:], q.line)
		n += nn
		q.line = q.line[nn:]
	}

	return n, nil
}

func fromHex(b byte) (byte, error) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', nil
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10, nil
	}
	return 0, fmt.Errorf("quotedprintable: invalid quoted-printable hex byte 0x%02x", b)
}

func readHexByte(v []byte) (b byte, err error) {
	var hb, lb byte
	if hb, err = fromHex(v[0]); err != nil {
		return 0, err
	}
	if lb, err = fromHex(v[1]); err != nil {
		return 0, err
	}
	return hb<<4 | lb, nil
}

// isVchar returns true if c is an RFC 5322 VCHAR character.
func isVchar(c byte) bool {
	// Visible (printing) characters.
	return '!' <= c && c <= '~'
}

// isWSP returns true if c is a WSP (white space).
// WSP is a space or horizontal tab (RFC5234 Appendix B).
func isWSP(c byte) bool {
	return c == ' ' || c == '\t'
}

// isNewline returns true if c is a newline character.
func isNewline(c byte) bool {
	return c == '\n' || c == '\r'
}
