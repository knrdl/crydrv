package main

import (
	"bytes"
	"testing"
)

func TestAppKey(t *testing.T) {
	appKey := Try(makeAppKey())
	if len(appKey) != APP_KEY_LENGTH {
		t.Errorf("app key has wrong length")
	}
}
func TestStrEncodeDecode(t *testing.T) {
	appKey := Try(makeAppKey())
	if !bytes.Equal(Try(strDecode(strEncode(appKey))), appKey) {
		t.Errorf("string encoding failed")
	}
}

func TestUsersalt(t *testing.T) {
	appKey := Try(makeAppKey())
	salt1a := makeUserSalt(appKey, Username("user1"))
	salt1b := makeUserSalt(appKey, Username("user1"))
	salt2 := makeUserSalt(appKey, Username("user2"))
	if len(salt1a) != USER_SALT_LENGTH {
		t.Errorf("wrong salt length")
	}
	if !bytes.Equal(salt1a, salt1b) || bytes.Equal(salt1a, salt2) {
		t.Errorf("salting failed")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	username := "user1"
	password := "pass1"
	appKey := Try(makeAppKey())
	userSalt := makeUserSalt(appKey, Username(username))
	userKey := Password(password).hash(userSalt)
	if len(userKey) != USER_KEY_LENGTH {
		t.Errorf("wrong userkey length")
	}

	cipher1a := Try(userKey.encrypt(Plaintext(appKey)))
	cipher1b := Try(userKey.encrypt(Plaintext(appKey)))
	if len(cipher1a) != len(appKey)+12+16 {
		t.Errorf("wrong ciphertext length: %d", len(cipher1a))
	}
	if bytes.Equal(cipher1a, cipher1b) {
		t.Errorf("nonce reused")
	}
	if !bytes.Equal(Try(userKey.decrypt(cipher1a)), appKey) {
		t.Errorf("encrypt/decrypt failed")
	}
}

func BenchmarkEncrypt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		appKey := []byte(Try(makeAppKey()))
		userKey := UserKey(appKey)
		plaintext := Plaintext(appKey)
		b.StartTimer()
		_ = Try(userKey.encrypt(plaintext))
	}
}
func BenchmarkDecrypt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		appKey := []byte(Try(makeAppKey()))
		userKey := UserKey(appKey)
		plaintext := Plaintext(appKey)
		ciphertext := Ciphertext(Try(userKey.encrypt(plaintext)))
		b.StartTimer()
		_ = Try(userKey.decrypt(ciphertext))
	}
}
