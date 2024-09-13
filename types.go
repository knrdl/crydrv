package main

type AppKey []byte
type UserSalt []byte
type UserKey []byte
type UserKeyHash []byte

type Plaintext []byte
type Ciphertext []byte

type Username string
type Password string

type FsPath string
type FsFilename string
type CryPath string

func (s Password) String() string {
	return "*****"
}
