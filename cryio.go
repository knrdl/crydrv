package main

import (
	"io"
	"math"
	"os"
	"path"
	"sync"
	"time"
)

const BLOCK_SIZE_UNENCRYPTED int = 4 * 1024 * 1024                // 4 MiB
const BLOCK_SIZE_ENCRYPTED int = BLOCK_SIZE_UNENCRYPTED + 12 + 16 // 4 MiB + AES nonce + PKCS#7 padding

var accessLocks sync.Map = sync.Map{}

func (filepath FsFilepath) ReadLock() {
	mutex, _ := accessLocks.LoadOrStore(filepath, new(sync.RWMutex))
	mutex.(*sync.RWMutex).RLock()
}
func (filepath FsFilepath) ReadUnlock() {
	if mutex, ok := accessLocks.Load(filepath); ok {
		mutex.(*sync.RWMutex).RUnlock()
	}
}
func (filepath FsFilepath) WriteLock() {
	mutex, _ := accessLocks.LoadOrStore(filepath, new(sync.RWMutex))
	mutex.(*sync.RWMutex).Lock()
}
func (filepath FsFilepath) WriteUnlock() {
	if mutex, ok := accessLocks.Load(filepath); ok {
		mutex.(*sync.RWMutex).Unlock()
	}
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

	f.filepath.ReadLock()
	defer f.filepath.ReadUnlock()

	stat, err := os.Stat(string(f.filepath))
	if err != nil {
		return nil, err
	}

	f.blocks = int64(math.Ceil(float64(stat.Size()) / float64(BLOCK_SIZE_ENCRYPTED)))
	f.modTime = stat.ModTime()

	f.file, err = os.Open(string(f.filepath))
	if err != nil {
		return nil, err
	}

	lastBlockIndex := int64(f.blocks - 1)
	buf := make(Ciphertext, BLOCK_SIZE_ENCRYPTED)
	f.file.Seek((lastBlockIndex * int64(BLOCK_SIZE_ENCRYPTED)), io.SeekStart)
	n, err := f.file.Read(buf)
	if err != nil {
		defer f.file.Close()
		return nil, err
	}
	decrypted, err := f.userKey.decrypt(buf[:n])
	if err != nil {
		defer f.file.Close()
		return nil, err
	}
	f.blockCache = &BlockCache{index: lastBlockIndex, data: decrypted}
	f.datasize = (lastBlockIndex * int64(BLOCK_SIZE_UNENCRYPTED)) + int64(len(decrypted))

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
		f.filepath.ReadLock()
		defer f.filepath.ReadUnlock()

		f.file.Seek(blockIndex*int64(BLOCK_SIZE_ENCRYPTED), io.SeekStart)
		buf := make(Ciphertext, BLOCK_SIZE_ENCRYPTED)
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

	f.position += readableByteSize

	copy(p, decrypted[blockOffset:blockOffset+readableByteSize])
	return int(readableByteSize), nil
}

func (f *CryFileReader) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekStart {
		f.position = offset
	}
	if whence == io.SeekCurrent {
		f.position += offset
	}
	if whence == io.SeekEnd {
		f.position = f.datasize + offset
	}
	f.position = Clamp(0, f.position, f.datasize)
	return f.position, nil
}

func (f *CryFileReader) Close() error {
	return f.file.Close()
}

func WriteCryFile(outFilepath FsFilepath, inFile io.Reader, inFileSize int64, userKey UserKey) error {

	outDir := path.Dir(string(outFilepath))
	if err := os.MkdirAll(outDir, 0700); err != nil {
		return err
	}

	outFilepath.WriteLock()
	defer outFilepath.WriteUnlock()

	outFile, err := os.Create(string(outFilepath))
	if err != nil {
		return err
	}
	defer outFile.Close()

	buf := make(Plaintext, Min(int64(BLOCK_SIZE_UNENCRYPTED), inFileSize))
	for {
		n, err := inFile.Read(buf)
		if err != io.EOF {
			if err != nil {
				return err
			}

			encrypted, err := userKey.encrypt(buf[:n])
			if err != nil {
				return err
			}
			outFile.Write(encrypted)
		} else {
			break
		}
	}

	if err := outFile.Close(); err != nil {
		return err
	}

	return nil
}

func (cryFilename *CryFilename) toFilepath(basedir string) FsFilepath {
	str := string(*cryFilename)
	return FsFilepath(path.Join(basedir, str[0:2], str[2:]))
}
