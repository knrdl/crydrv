package main

import (
	"bytes"
	"fmt"
	"regexp"
)

type AppKey []byte
type UserSalt []byte
type UserKey []byte
type UserFingerprint []byte
type UserFingerprints []UserFingerprint

type Plaintext []byte
type Ciphertext []byte

type Username string
type Password string

type FsFilepath string
type CryFilename string
type CryPath string

func (s Password) String() string {
	return "*****"
}

func (userFingerprints *UserFingerprints) Contains(userFingerprint UserFingerprint) bool {
	for _, fp := range *userFingerprints {
		if bytes.Equal(fp, userFingerprint) {
			return true
		}
	}
	return false
}

func (userFingerprints *UserFingerprints) Load(config string) error {
	pattern := regexp.MustCompile(`[^a-zA-Z0-9\-_]+`) // base64url-encoded
	split := pattern.Split(config, -1)
	for i, fpStr := range split {
		if fpStr == "" {
			continue
		}
		fp, err := strDecode(fpStr)
		if err != nil {
			return err
		}
		if len(fp) != USER_FINGERPRINT_LENGTH {
			return fmt.Errorf("invalid fingerprint length for record %d", i)
		}
		*userFingerprints = append(*userFingerprints, fp)
	}
	return nil
}
