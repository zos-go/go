// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ld

import (
	"fmt"
)

// Convert from UTF-8 to 8-bit EBCDIC (IBM-1047).
// Only supports the subset of characters needed to encode linker symbols.
// Encodings from: http://www.ibm.com/support/knowledgecenter/SSEQ5Y_12.0.0/com.ibm.pcomm.doc/reference/html/hcp_reference27.htm
// Wikipedia version: https://en.wikipedia.org/wiki/EBCDIC_1047

var codePage1047 map[rune]byte

func init() {
	codePage1047 = make(map[rune]byte)

	// Control characters.
	codePage1047[0] = 0x00 // null

	// Whitespace.
	codePage1047[' '] = 0x40  // space
	codePage1047['\n'] = 0x25 // line feed
	codePage1047['\t'] = 0x05 // horizontal tab

	// Punctuation.
	codePage1047['!'] = 0x5A
	codePage1047['"'] = 0x7F
	codePage1047['#'] = 0x7B
	codePage1047['$'] = 0x5B
	codePage1047['%'] = 0x6C
	codePage1047['&'] = 0x50
	codePage1047['\''] = 0x7D
	codePage1047['('] = 0x4D
	codePage1047[')'] = 0x5D
	codePage1047['*'] = 0x5C
	codePage1047['+'] = 0x4E
	codePage1047[','] = 0x6B
	codePage1047['-'] = 0x60
	codePage1047['.'] = 0x4B
	codePage1047['/'] = 0x61
	codePage1047[':'] = 0x7A
	codePage1047[';'] = 0x5E
	codePage1047['<'] = 0x4C
	codePage1047['='] = 0x7E
	codePage1047['>'] = 0x6E
	codePage1047['?'] = 0x6F
	codePage1047['@'] = 0x7C
	codePage1047['['] = 0xAD
	codePage1047['\\'] = 0xE0
	codePage1047[']'] = 0xBD
	codePage1047['^'] = 0x5F
	codePage1047['_'] = 0x6D
	codePage1047['`'] = 0x79
	codePage1047['{'] = 0xC0
	codePage1047['|'] = 0x4F
	codePage1047['}'] = 0xD0
	codePage1047['~'] = 0xA1

	// Alphanumeric characters.
	regions := []struct {
		start rune
		end   rune
		point byte
	}{
		{'a', 'i', 0x81},
		{'j', 'r', 0x91},
		{'s', 'z', 0xA2},
		{'A', 'I', 0xC1},
		{'J', 'R', 0xD1},
		{'S', 'Z', 0xE2},
		{'0', '9', 0xF0},
	}
	for _, x := range regions {
		for i := x.start; i <= x.end; i++ {
			codePage1047[i] = byte(i - x.start + rune(x.point))
		}
	}

	// Extended characters.
	codePage1047['·'] = 0xB3

	// Fake characters.
	// The binder doesn't like these characters.
	// TODO(mundaym): these should just be hashed if they are encountered.
	codePage1047[0x394] = 0xB4
	codePage1047[' '] = 0xB5
	codePage1047['\t'] = 0xB6
}

func EncodeStringEBCDIC(s string) ([]byte, error) {
	if len(s) == 0 {
		return nil, nil
	}
	enc := make([]byte, 0)
	for _, c := range s {
		e, ok := codePage1047[c]
		if !ok {
			return nil, fmt.Errorf("unknown rune '%c' (UTF-8 code point: %#x)", c, c)
		}
		enc = append(enc, e)
	}
	return enc, nil
}
