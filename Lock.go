package part

import (
	"os"
    "sync"
)

type lock struct {
	sync.Mutex
}

var (
	lock_file *os.File
	lock_md5Key string
)

func Lock() *lock {
	return &lock{}
}

func (l *lock) Start(filePath string) int {
	l.Lock()
	defer l.Unlock()

	if l.State() {return 1}
	if Checkfile().IsOpen(lock_md5Key) {return 2}
	if !Checkfile().IsExist(filePath) {return 3}

	lock_md5Key = ".lock."+filePath
	lock_file, _ = os.Create(lock_md5Key)
	return 0
}

func (l *lock) Stop() int {
	l.Lock()
	defer l.Unlock()

	if !l.State() {return 1}
	if !Checkfile().IsExist(lock_md5Key) {return 2}

	lock_file.Close()
	os.Remove(lock_md5Key)
	lock_md5Key = ""
	return 0
}

func (*lock) State() bool {
	return lock_md5Key != ""
}