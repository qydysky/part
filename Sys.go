package part

import (
	"sync"
    "path/filepath"
	"os"
	"runtime"
	"time"
	"strconv"
)

type sys struct {sync.Mutex}

func Sys () *sys {
	return &sys{}
}

func (this *sys) Cdir()string{
	this.Lock()
	defer this.Unlock()

    dir, _ := os.Executable()
    exPath := filepath.Dir(dir)
    return exPath
}

func (this *sys) Timeoutf(Timeout int) {
	this.Lock()
	defer this.Unlock()
	
    time.Sleep(time.Duration(Timeout)*time.Second)
}

func (this *sys) GetSys(sys string)bool{
	this.Lock()
	defer this.Unlock()

    return runtime.GOOS==sys
}

func (this *sys) GetTime() string {
	now := strconv.FormatInt(time.Now().Unix(),10)
	return now[len(now)-4:]
}