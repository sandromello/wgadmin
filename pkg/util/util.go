package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"time"
)

type CipherKey struct {
	Key []byte
}

func pad(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func unpad(src []byte) ([]byte, error) {
	length := len(src)
	unpadding := int(src[length-1])

	if unpadding > length {
		return nil, errors.New("unpad error. This could happen when incorrect encryption key is used")
	}

	return src[:(length - unpadding)], nil
}

// NewAESCipherKey will generate a random safe string if the key is empty,
// otherwise it will as the cipher key
func NewAESCipherKey(base64Key string) (*CipherKey, error) {
	var cipherKey [sha256.Size]byte
	if base64Key == "" {
		randomSafeKey, err := GenerateRandomString(32)
		if err != nil {
			return nil, err
		}
		cipherKey = sha256.Sum256([]byte(randomSafeKey))

	} else {
		key, err := base64.StdEncoding.DecodeString(base64Key)
		if err != nil {
			return nil, err
		}
		copy(cipherKey[:], key)
	}

	ck := &CipherKey{
		Key: []byte(cipherKey[:]),
	}
	return ck, nil
}

func (k *CipherKey) EncryptMessage(rawText string) (string, error) {
	block, err := aes.NewCipher(k.Key)
	if err != nil {
		return "", err
	}

	msg := pad([]byte(rawText))
	ciphertext := make([]byte, aes.BlockSize+len(msg))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(msg))
	finalMsg := base64.StdEncoding.EncodeToString(ciphertext)
	return finalMsg, nil
}

func (k *CipherKey) DecryptMessage(encryptedText string) (string, error) {
	block, err := aes.NewCipher(k.Key)
	if err != nil {
		return "", err
	}

	decodedMsg, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	if (len(decodedMsg) % aes.BlockSize) != 0 {
		return "", errors.New("blocksize must be multipe of decoded message length")
	}

	iv := decodedMsg[:aes.BlockSize]
	msg := decodedMsg[aes.BlockSize:]

	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(msg, msg)

	unpadMsg, err := unpad(msg)
	if err != nil {
		return "", err
	}

	return string(unpadMsg), nil
}

func (k *CipherKey) String() string {
	return base64.StdEncoding.EncodeToString(k.Key)
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomString(n int) (string, error) {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return "", err
	}
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	for i, b := range bytes {
		bytes[i] = letters[b%byte(len(letters))]
	}
	return string(bytes), err
}

// RoundTime round time based on a resolution (r) from a given a duration (d)
func RoundTime(d, r time.Duration) time.Duration {
	if r <= 0 {
		return d
	}
	neg := d < 0
	if neg {
		d = -d
	}
	if m := d % r; m+m < r {
		d = d - m
	} else {
		d = d + r - m
	}
	if neg {
		return -d
	}
	return d
}

// GetDeltaDuration computes the delta between two string RFC3339 dates
func GetDeltaDuration(startTime, endTime string) string {
	start, _ := time.Parse(time.RFC3339, startTime)
	end, _ := time.Parse(time.RFC3339, endTime)
	delta := end.Sub(start)
	var d time.Duration
	if endTime != "" {
		d = RoundTime(delta, time.Second)
	} else {
		d = RoundTime(time.Since(start), time.Second)
	}
	switch {
	case d.Hours() >= 24: // day resolution
		return fmt.Sprintf("%.fd", math.Floor(d.Hours()/24))
	case d.Hours() >= 8760: // year resolution
		return fmt.Sprintf("%.fd", math.Floor(d.Hours()/8760))
	}
	return d.String()
}
