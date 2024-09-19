package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
)

var hkdfHasher = sha512.New512_256

const USER_KEY_LENGTH = 32         // bytes
const USER_FINGERPRINT_LENGTH = 32 // bytes

func strEncode(value []byte) string {
	return base64.RawURLEncoding.EncodeToString(value)
}

func strDecode(value string) ([]byte, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func makeAppKey() (AppKey, error) {
	value := make(AppKey, hkdfHasher().Size())
	if _, err := rand.Read(value); err != nil {
		return nil, err
	}
	return value, nil
}

func makeUserSalt(appKey AppKey, username Username) UserSalt {
	hkdf := hkdf.New(hkdfHasher, appKey, []byte(username), nil)
	hash := make(UserSalt, hkdfHasher().Size())
	Try(io.ReadFull(hkdf, hash))
	return hash
}

func (userKey UserKey) hash(userSalt UserSalt) UserFingerprint {
	hkdf := hkdf.New(hkdfHasher, userKey, userSalt, nil)
	hash := make(UserFingerprint, hkdfHasher().Size())
	Try(io.ReadFull(hkdf, hash))
	return hash
}

func (crypath CryPath) hash(userKey UserKey, userSalt UserSalt) CryFilename {
	hkdf := hkdf.New(hkdfHasher, userKey, append(userSalt[:], []byte(crypath)[:]...), nil)
	hash := make([]byte, hkdfHasher().Size())
	Try(io.ReadFull(hkdf, hash))
	return CryFilename(strEncode(hash))
}

func (password Password) hash(userSalt UserSalt) UserKey {
	const iterations = 3
	const memory = 64 * 1024 // KiB
	const parallelism = 4
	return argon2.IDKey([]byte(password), userSalt, iterations, memory, parallelism, USER_KEY_LENGTH)
}

func (userKey UserKey) encrypt(plaintext Plaintext) (ciphertext Ciphertext, err error) {

	//Create a new Cipher Block from the key
	block, err := aes.NewCipher(userKey)
	if err != nil {
		return nil, err
	}

	//Create a new GCM - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	//https://golang.org/pkg/crypto/cipher/#NewGCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	//Create a nonce. Nonce should be from GCM
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	//Encrypt the data using aesGCM.Seal
	//Since we don't want to save the nonce somewhere else in this case, we add it as a prefix to the encrypted data. The first nonce argument in Seal is the prefix.
	return aesGCM.Seal(nonce, nonce, plaintext, nil), nil
}

func (userKey UserKey) decrypt(ciphertext Ciphertext) (plaintext Plaintext, err error) {

	//Create a new Cipher Block from the key
	block, err := aes.NewCipher(userKey)
	if err != nil {
		return nil, err
	}

	//Create a new GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	//Get the nonce size
	nonceSize := aesGCM.NonceSize()

	//Extract the nonce from the encrypted data
	nonce, ciphertextWithoutPrefix := ciphertext[:nonceSize], ciphertext[nonceSize:]

	//Decrypt the data
	plaintext, err = aesGCM.Open(nil, nonce, ciphertextWithoutPrefix, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
