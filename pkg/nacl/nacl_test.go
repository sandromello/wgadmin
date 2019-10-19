package nacl

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEncryptionDecryption(t *testing.T) {
	key := []byte(`asecret`)
	rawMessage := []byte(`foo`)
	encmsg, err := EncryptMessage(key, rawMessage)
	if err != nil {
		t.Fatalf("failed encrypting message: %v", err)
	}
	decrypted, err := DecryptMessage(key, encmsg)
	if err != nil {
		t.Fatalf("failed decrypting message: %v", err)
	}
	if diff := cmp.Diff(string(rawMessage), string(decrypted)); diff != "" {
		t.Fatalf("unexpected public key (-want +got):\n%s", diff)
	}
}

func TestDecryptFail(t *testing.T) {
	key := []byte(`asecret`)
	rawMessage := []byte(`foo`)
	encmsg, err := EncryptMessage(key, rawMessage)
	if err != nil {
		t.Fatalf("failed encrypting message: %v", err)
	}
	key = []byte(`anewsecret`)
	decrypted, err := DecryptMessage(key, encmsg)
	if err != nil && err.Error() == "failed decrypting message" {
		return
	}
	t.Fatalf("unexpected decrypted key: %v, err: %v", string(decrypted), err)
}
