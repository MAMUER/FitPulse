package crypto

import (
	"testing"
)

func FuzzAESGCMEncryptor(f *testing.F) {
	validKey := "123456789012345678901234567890 2"
	enc, err := NewAESGCMEncryptor(validKey)
	if err != nil {
		f.Fatal(err)
	}

	f.Add("")
	f.Add("hello world")
	f.Add("Привет мир")
	f.Add(`<script>alert("xss")</script>`)
	f.Add(`'; DROP TABLE users; --`)
	f.Add("https://example.com?q=test&token=abc")
	f.Add(string([]byte{0x00, 0x01, 0x02, 0x03}))
	f.Add("a")
	f.Add("long boring text that should still survive encryption and decryption without panicking")

	f.Fuzz(func(t *testing.T, input string) {
		if input == "" {
			t.Skip("empty input is covered by unit tests")
		}

		plaintext := []byte(input)
		ciphertext, err := enc.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("encrypt failed: %v", err)
		}

		got, err := enc.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("decrypt failed: %v", err)
		}

		if string(got) != input {
			t.Errorf("roundtrip mismatch: got %q, want %q", got, input)
		}
	})
}
