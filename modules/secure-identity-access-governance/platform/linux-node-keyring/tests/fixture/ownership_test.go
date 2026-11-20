//go:build linux && symphony_node_keyring_fixture

package main

import (
	"encoding/binary"
	"math"
	"strconv"
	"testing"
)

func TestParseFixtureID(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		value uint32
		valid bool
	}{
		{"1", 1, true},
		{"1001", 1001, true},
		{"2147483647", math.MaxInt32, true},
		{"2147483648", math.MaxInt32 + 1, strconv.IntSize == 64},
		{"4294967294", math.MaxUint32 - 1, strconv.IntSize == 64},
		{"4294967295", 0, false}, // POSIX ownership sentinel
		{"4294967296", 0, false}, // would wrap to root
		{"18446744073709551616", 0, false},
		{"", 0, false},
		{"0", 0, false},
		{"-1", 0, false},
		{"+1", 0, false},
		{"01", 0, false},
		{" 1", 0, false},
		{"1\n", 0, false},
		{"1x", 0, false},
	} {
		t.Run(strconv.Quote(tc.raw), func(t *testing.T) {
			got, err := parseFixtureID(tc.raw)
			if (err == nil) != tc.valid || (tc.valid && got != tc.value) {
				t.Fatalf("parseFixtureID(%q) returned %d, %v (want valid=%t, value=%d)", tc.raw, got, err, tc.valid, tc.value)
			}
		})
	}
}

func TestFixtureOwnershipReachesCredentialsAndACL(t *testing.T) {
	maxID := "4294967294"
	if strconv.IntSize == 32 {
		maxID = "2147483647"
	}
	for _, pair := range [][2]string{{maxID, "1001"}, {"1001", maxID}} {
		uid, err := parseFixtureID(pair[0])
		if err != nil {
			t.Fatal(err)
		}
		gid, err := parseFixtureID(pair[1])
		if err != nil {
			t.Fatal(err)
		}
		a := account{uid: uid, gid: gid}
		creds := a.credentials().Credential
		if creds == nil || creds.Uid != uid || creds.Gid != gid || len(creds.Groups) != 1 || creds.Groups[0] != gid {
			t.Fatalf("service IDs %d/%d did not reach the child credentials intact: %+v", uid, gid, creds)
		}

		// Linux encodes the named-user entry directly after the four-byte
		// header and the eight-byte owner entry, all in little-endian order.
		acl := credentialACL(a.uid, 0)
		if len(acl) < 20 || binary.LittleEndian.Uint16(acl[12:14]) != 2 ||
			binary.LittleEndian.Uint16(acl[14:16]) != 4 || binary.LittleEndian.Uint32(acl[16:20]) != uid {
			t.Fatalf("service UID %d did not reach the named-user read ACL", uid)
		}
	}
}
