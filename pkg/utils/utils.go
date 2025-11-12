package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func IsValidEmail(email string) bool {
	return strings.Contains(email, "@")
}

func GetCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// timestamp in milliseconds
func GetCurrentTimestampMilliseconds() int64 {
	return time.Now().UnixMilli()
}

func GenerateUUID() string {
	return uuid.New().String()
}

func ContainsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}

// EncryptToken encrypts a token using AES encryption (compatible with CryptoJS.AES.encrypt)
// OpenSSL key derivation (EVP_BytesToKey)
func evpBytesToKey(password, salt []byte, keyLen, ivLen int) (key, iv []byte) {
	hash := []byte{}
	var prev []byte
	for len(hash) < keyLen+ivLen {
		data := append(prev, password...)
		data = append(data, salt...)
		sum := md5.Sum(data)
		prev = sum[:]
		hash = append(hash, prev...)
	}
	key = hash[:keyLen]
	iv = hash[keyLen : keyLen+ivLen]
	return
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

func EncryptToken(plaintext string) (string, error) {
	salt := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	encryptionKey := fmt.Sprintf("%d", GetCurrentTimestampMilliseconds())

	key, iv := evpBytesToKey([]byte(encryptionKey), salt, 32, 16)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	plainPadded := pkcs7Pad([]byte(plaintext), block.BlockSize())
	ciphertext := make([]byte, len(plainPadded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plainPadded)

	// prepend "Salted__" + salt
	final := append([]byte("Salted__"), salt...)
	final = append(final, ciphertext...)

	// Base64 encode
	return base64.StdEncoding.EncodeToString(final), nil
}

// Check for string contains a specific strings array
func ContainsString(str string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(str, substr) {
			return true
		}
	}
	return false
}

// StringToInt converts a string to an integer with a default value if conversion fails
func StringToInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return val
}
