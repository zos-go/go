// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ld

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// 7-bit ASCII printable and whitespace characters.
const ascii = "" +
	"\n!\"#$%&'()*+,-./0123456789:;<=>?@" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`" +
	"abcdefghijklmnopqrstuvwxyz{|}~\n"

// 8-bit IBM-1047 equivalent.
const ibm1047 = "\x25\x5a\x7f\x7b\x5b\x6c\x50" +
	"\x7d\x4d\x5d\x5c\x4e\x6b\x60\x4b\x61" +
	"\xf0\xf1\xf2\xf3\xf4\xf5\xf6\xf7\xf8" +
	"\xf9\x7a\x5e\x4c\x7e\x6e\x6f\x7c\xc1" +
	"\xc2\xc3\xc4\xc5\xc6\xc7\xc8\xc9\xd1" +
	"\xd2\xd3\xd4\xd5\xd6\xd7\xd8\xd9\xe2" +
	"\xe3\xe4\xe5\xe6\xe7\xe8\xe9\xad\xe0" +
	"\xbd\x5f\x6d\x79\x81\x82\x83\x84\x85" +
	"\x86\x87\x88\x89\x91\x92\x93\x94\x95" +
	"\x96\x97\x98\x99\xa2\xa3\xa4\xa5\xa6" +
	"\xa7\xa8\xa9\xc0\x4f\xd0\xa1\x25"

func TestUTF8ToEBCDICEncodeIconv(t *testing.T) {
	// Use iconv to generate the golden data.
	l, err := exec.Command("iconv", "-l").CombinedOutput()
	if err != nil {
		t.Skipf("cannot run iconv: %v", err)
	}
	const codePage = "IBM-1047"
	if !strings.Contains(string(l), codePage) {
		t.Skipf("iconv does not support %v", codePage)
	}

	strings := []string{
		ascii,
		"main·main",
	}
	for _, s := range strings {
		e, err := EncodeStringEBCDIC(s)
		if err != nil {
			t.Fatalf("%v", err)
		}
		cmd := exec.Command("iconv", "-f", codePage, "-t", "UTF-8")
		cmd.Stdin = bytes.NewReader(e)
		o, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v", err)
		}
		if string(o) != s {
			t.Errorf("wanted: %s\n", s)
			t.Errorf("got:    %s\n", o)
			t.Fatalf("output does not match input")
		}
	}
}

func TestASCIIToEBCDICEncodeGolden(t *testing.T) {
	e, err := EncodeStringEBCDIC(ascii)
	if err != nil {
		t.Fatalf("%v", err)
	}

	if string(e) != ibm1047 {
		t.Errorf("wanted: %v\n", []byte(ibm1047))
		t.Errorf("got:    %v\n", e)
		t.Fatalf("output does not match input")
	}
}

func TestEBCDICEncodeError(t *testing.T) {
	bad := []struct {
		start, end int
	}{
		{1, 8},
		{11, 31},
	}
	for _, x := range bad {
		for i := x.start; i <= x.end; i++ {
			c := []byte{byte(i)}
			_, err := EncodeStringEBCDIC(string(c))
			if err == nil {
				t.Fatalf("expected an error encoding %v", i)
			}
		}
	}
	utf8 := []string{
		"こんにちは",
		"Χαίρετε",
		"你好",
	}
	for _, x := range utf8 {
		_, err := EncodeStringEBCDIC(x)
		if err == nil {
			t.Fatalf("expected an error encoding %v", x)
		}
	}
}
