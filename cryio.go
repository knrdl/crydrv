package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const BLOCK_SIZE_UNENCRYPTED = 4 * 1024 * 1024                // 4 MiB
const BLOCK_SIZE_ENCRYPTED = BLOCK_SIZE_UNENCRYPTED + 12 + 16 // 4 MiB + AES nonce + PKCS#7 padding

var cipherReadBufferPool = sync.Pool{
	New: func() any {
		return make(Ciphertext, BLOCK_SIZE_ENCRYPTED)
	},
}

type BlockCache struct {
	sync.Mutex

	index int64
	data  Plaintext
}

type CryFileReader struct {
	io.ReadSeeker

	filepath FsFilepath
	datasize int64
	blocks   int64
	position int64
	userKey  UserKey
	modTime  time.Time

	blockCache *BlockCache

	file *os.File
}

func NewCryFileReader(filepath FsFilepath, userKey UserKey) (*CryFileReader, error) {
	f := new(CryFileReader)
	f.userKey = userKey
	f.filepath = filepath
	f.position = 0

	stat, err := os.Stat(string(f.filepath))
	if err != nil {
		return nil, err
	}

	f.blocks = (stat.Size() + BLOCK_SIZE_ENCRYPTED - 1) / BLOCK_SIZE_ENCRYPTED
	f.modTime = stat.ModTime()

	f.file, err = os.Open(string(f.filepath))
	if err != nil {
		return nil, err
	}

	if f.blocks > 0 {
		lastBlockIndex := int64(f.blocks - 1)
		buf := make(Ciphertext, BLOCK_SIZE_ENCRYPTED)
		_, err = f.file.Seek((lastBlockIndex * int64(BLOCK_SIZE_ENCRYPTED)), io.SeekStart)
		if err != nil {
			defer IgnoreErrFunc(f.file.Close)
			return nil, err
		}
		n, err := f.file.Read(buf)
		if err != nil {
			defer IgnoreErrFunc(f.file.Close)
			return nil, err
		}
		decrypted, err := f.userKey.decrypt(buf[:n])
		if err != nil {
			defer IgnoreErrFunc(f.file.Close)
			return nil, err
		}
		f.blockCache = &BlockCache{index: lastBlockIndex, data: decrypted}
		f.datasize = (lastBlockIndex * int64(BLOCK_SIZE_UNENCRYPTED)) + int64(len(decrypted))
	} else { // empty file
		f.blockCache = &BlockCache{index: 0, data: []byte{}}
		f.datasize = 0
	}
	return f, nil
}

func (f *CryFileReader) Read(p []byte) (int, error) {

	if f.position >= f.datasize {
		return 0, io.EOF
	}

	blockIndex := int64(f.position) / int64(BLOCK_SIZE_UNENCRYPTED)
	blockOffset := f.position - blockIndex*int64(BLOCK_SIZE_UNENCRYPTED)

	f.blockCache.Lock()
	cacheIndex := f.blockCache.index
	f.blockCache.Unlock()
	if cacheIndex != blockIndex {

		_, err := f.file.Seek(blockIndex*int64(BLOCK_SIZE_ENCRYPTED), io.SeekStart)
		if err != nil {
			defer IgnoreErrFunc(f.file.Close)
			return 0, err
		}

		buf := cipherReadBufferPool.Get().(Ciphertext)
		defer cipherReadBufferPool.Put(buf)
		n, err := f.file.Read(buf)
		if err != nil {
			return 0, err
		}
		decrypted, err := f.userKey.decrypt(buf[:n])
		if err != nil {
			return 0, err
		}
		f.blockCache.Lock()
		f.blockCache.index = blockIndex
		f.blockCache.data = decrypted
		f.blockCache.Unlock()
	}
	f.blockCache.Lock()
	decrypted := f.blockCache.data
	f.blockCache.Unlock()

	remainingBlockSize := int64(len(decrypted)) - blockOffset
	readableByteSize := Min(int64(len(p)), remainingBlockSize)

	if readableByteSize > 0 {
		f.position += readableByteSize

		copy(p, decrypted[blockOffset:blockOffset+readableByteSize])
		return int(readableByteSize), nil
	} else {
		return 0, io.EOF
	}
}

func (f *CryFileReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		f.position = offset
	case io.SeekCurrent:
		f.position += offset
	case io.SeekEnd:
		f.position = f.datasize + offset
	default:
		return 0, errors.New("invalid whence")
	}
	f.position = Clamp(0, f.position, f.datasize)
	return f.position, nil
}

func (f *CryFileReader) Close() error {
	return f.file.Close()
}

func WriteCryFile(outFilepath FsFilepath, inFile io.Reader, inFileSize int64, userKey UserKey) error {

	outDir := filepath.Dir(string(outFilepath))
	if err := os.MkdirAll(outDir, 0700); err != nil {
		return err
	}

	lock := outFilepath.WriteLock()
	defer outFilepath.WriteUnlock(lock)

	outFile, err := os.Create(string(outFilepath))
	if err != nil {
		return err
	}
	defer IgnoreErrFunc(outFile.Close)

	buf := make(Plaintext, Min(int64(BLOCK_SIZE_UNENCRYPTED), inFileSize))
	for {
		n, err := inFile.Read(buf)

		if n > 0 {
			encrypted, err := userKey.encrypt(buf[:n])
			if err != nil {
				return err
			}

			if _, err := outFile.Write(encrypted); err != nil {
				return err
			}
		}

		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
	}

	if err := outFile.Close(); err != nil {
		return err
	}

	return nil
}

func (cryFilename *CryFilename) toFilepath(basedir string) FsFilepath {
	str := string(*cryFilename)
	return FsFilepath(filepath.Join(basedir, str[0:2], str[2:]))
}
