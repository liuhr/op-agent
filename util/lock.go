package util

import (
	"errors"
	"os"
	"syscall"
)

type Flock struct {
	LockFile string
	lock     *os.File
}

// 创建文件锁，配合 defer f.Release() 来使用
func CreateLock(file string) (f *Flock, e error) {
	if file == "" {
		e = errors.New("cannot create flock on empty path")
		return
	}
	lock, e := os.Create(file)
	if e != nil {
		return
	}
	return &Flock{
		LockFile: file,
		lock:     lock,
	}, nil
}

// 释放文件锁
func (f *Flock) ReleaseLock() {
	if f != nil && f.lock != nil {
		f.lock.Close()
		os.Remove(f.LockFile)
	}
}

// 上锁，配合 defer f.Unlock() 来使用
func (f *Flock) Lock() (e error) {
	if f == nil {
		e = errors.New("cannot use lock on a nil flock")
		return
	}
	return syscall.Flock(int(f.lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

// 解锁
func (f *Flock) Unlock() {
	if f != nil {
		syscall.Flock(int(f.lock.Fd()), syscall.LOCK_UN)
	}
}


