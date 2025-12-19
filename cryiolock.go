package main

import (
	"sync"
	"time"
)

type FileLock struct {
	sync.RWMutex
	refs int64
}

type FileLocker struct {
	sync.Mutex
	locks       map[FsFilepath]*FileLock
	lastRebuild time.Time
}

var fileLocker = &FileLocker{
	locks:       make(map[FsFilepath]*FileLock),
	lastRebuild: time.Now(),
}

func (locker *FileLocker) acquire(path FsFilepath) *FileLock {
	locker.Lock()
	defer locker.Unlock()

	l := locker.locks[path]
	if l == nil {
		l = &FileLock{}
		locker.locks[path] = l
	}
	l.refs++
	return l
}

func (locker *FileLocker) release(path FsFilepath) {
	locker.Lock()
	defer locker.Unlock()

	l := locker.locks[path]
	if l == nil {
		panic("release called for unknown path: unlock without lock")
	}
	l.refs--
	if l.refs < 0 {
		panic("file lock refs went negative")
	}
	if l.refs == 0 {
		delete(locker.locks, path)
	}

	if len(locker.locks) == 0 && time.Since(locker.lastRebuild) > time.Hour {
		locker.lastRebuild = time.Now()
		locker.locks = make(map[FsFilepath]*FileLock)
	}
}

func (filepath FsFilepath) ReadLock() *FileLock {
	l := fileLocker.acquire(filepath)
	l.RLock()
	return l
}

func (filepath FsFilepath) ReadUnlock(l *FileLock) {
	l.RUnlock()
	fileLocker.release(filepath)
}

func (filepath FsFilepath) WriteLock() *FileLock {
	l := fileLocker.acquire(filepath)
	l.Lock()
	return l
}

func (filepath FsFilepath) WriteUnlock(l *FileLock) {
	l.Unlock()
	fileLocker.release(filepath)
}
