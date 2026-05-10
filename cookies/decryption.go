package cookies

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"log"
)

func decryptCookie(encrypted []byte, key []byte) (string, error) {
	// plaintext cookie, no prefix
	if len(encrypted) == 0 {
		return "", nil
	}

	// not encrypted at all (very old cookies)
	if len(encrypted) < 3 || (string(encrypted[:3]) != "v10" && string(encrypted[:3]) != "v11") {
		return string(encrypted), nil
	}

	// strip the v10/v11 prefix
	encrypted = encrypted[3:]

	// IV is always 16 space characters on Linux
	iv := []byte("                ") // 16 spaces, 0x20

	// create AES block cipher with our derived key
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// CBC mode decryptor
	mode := cipher.NewCBCDecrypter(block, iv)

	// CBC requires data length to be multiple of block size (16)
	if len(encrypted)%aes.BlockSize != 0 {
		log.Fatal("encrypted data is not block aligned")
		return "", errors.New("encrypted data is not block aligned")
	}

	// decrypt in place
	plaintext := make([]byte, len(encrypted))
	mode.CryptBlocks(plaintext, encrypted)

	// strip PKCS7 padding
	// last byte tells you how many padding bytes were added
	padLen := int(plaintext[len(plaintext)-1])
	if padLen > aes.BlockSize || padLen == 0 {
		return "", errors.New("invalid padding")
	}
	plaintext = plaintext[:len(plaintext)-padLen]

	return string(plaintext), nil
}
