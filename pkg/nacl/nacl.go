package nacl

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/secretbox"
)

const encryptKeyName string = "ENCRYPT_KEY"

func keyToSecretBox(key []byte) (*[32]byte, error) {
	// Load your secret key from a safe place and
	// reuse it across multiple Seal calls.
	encryptKeyBytes, err := hex.DecodeString(hex.EncodeToString(key))
	if err != nil {
		return nil, err
	}
	var encryptKey [32]byte
	copy(encryptKey[:], encryptKeyBytes)
	return &encryptKey, nil
}

// EncryptMessage encrypts a message with an encrypt key
func EncryptMessage(key, msg []byte) ([]byte, error) {
	encryptKey, err := keyToSecretBox(key)
	if err != nil {
		return nil, err
	}
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, err
	}
	return secretbox.Seal(nonce[:], msg, &nonce, encryptKey), nil
}

// DecryptMessage decrypts a message with an encrypt key
func DecryptMessage(key, encryptedMessage []byte) ([]byte, error) {
	encryptKey, err := keyToSecretBox(key)
	if err != nil {
		return nil, err
	}
	// When you decrypt, you must use the same nonce and key you used to
	// encrypt the message. One way to achieve this is to store the nonce
	// alongside the encrypted message. Above, we stored the nonce in the first
	// 24 bytes of the encrypted text.
	var decryptNonce [24]byte
	copy(decryptNonce[:], encryptedMessage[:24])
	decrypted, ok := secretbox.Open(nil, encryptedMessage[24:], &decryptNonce, encryptKey)
	if !ok {
		return nil, fmt.Errorf("failed decrypting message")
	}
	return decrypted, nil
}
