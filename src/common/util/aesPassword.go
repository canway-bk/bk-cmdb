package util

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"io"
	"crypto/rand"
	"encoding/base64"
)

// AesPasswrodKey 16/32位
const (
	AesPasswrodKey = "7GC8DTjhVmpGrLCx"
)

var (
	aesBlock cipher.Block
	ErrAESTextSize = errors.New("ciphertext is not a multiple of the block size")
	ErrAESPadding = errors.New("cipher padding size error")
)

func _init() {
	var err error
	aesBlock, err = aes.NewCipher([]byte(AesPasswrodKey))
	if err != nil {
		panic(err)
	}
}

// AES解密
func AesDecrypt(pwd string) (string, error) {
	_init()
	src, err := base64.StdEncoding.DecodeString(pwd)
	if nil != err {
		return "", err
	}
	if len(src) < aes.BlockSize * 2 || len(src) % aes.BlockSize != 0 {
		return "", ErrAESTextSize
	}
	srcLen := len(src) - aes.BlockSize
	decryptText := make([]byte, srcLen)
	iv := src[srcLen:]
	mode := cipher.NewCBCDecrypter(aesBlock, iv)
	mode.CryptBlocks(decryptText, src[:srcLen])
	paddingLen := int(decryptText[srcLen - 1])
	if paddingLen > 16 {
		return "", ErrAESPadding
	}
	return string(decryptText[:srcLen - paddingLen]), nil
}

// AES加密
func AesEncrypt(src []byte) (string, error) {
	_init()
	padLen := aes.BlockSize - (len(src) % aes.BlockSize)
	for i := 0; i < padLen; i++ {
		src = append(src, byte(padLen))
	}
	srcLen := len(src)
	encryptText := make([]byte, srcLen + aes.BlockSize)
	iv := encryptText[srcLen:]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}
	mode := cipher.NewCBCEncrypter(aesBlock, iv)
	mode.CryptBlocks(encryptText[:srcLen], src)
	pwd := base64.StdEncoding.EncodeToString(encryptText)
	return pwd, nil
}
