// +build !linux


package Ppart

import (
    "syscall"
    "unsafe"
    "os"
    "os/exec"
    "path/filepath"
)

type ulong int32
type ulong_ptr uintptr

type PROCESSENTRY32 struct {
    dwSize ulong
    cntUsage ulong
    th32ProcessID ulong
    th32DefaultHeapID ulong_ptr
    th32ModuleID ulong
    cntThreads ulong
    th32ParentProcessID ulong
    pcPriClassBase ulong
    dwFlags ulong
    szExeFile [260]byte
}

func PCheck(pros []string) []int{
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
    pHandle,_,_  := kernel32.NewProc("CreateToolhelp32Snapshot").Call(uintptr(0x2),uintptr(0x0))
    res:=[]int{}
    _pros:=[][]byte{}
    if int(pHandle)==-1 {return res}
    for _,v:= range pros{
        if v=="" {return res}
        _pros=append(_pros,[]byte(v))
        res=append(res,0)
    }
    // fmt.Println(string(_pros[0]))
    pp:= kernel32.NewProc("Process32Next")
    var proc PROCESSENTRY32;
    var a [260]byte;
    proc.dwSize = ulong(unsafe.Sizeof(proc));
    
    for {
        proc.szExeFile=a;
        rt,_,_ := pp.Call(uintptr(pHandle),uintptr(unsafe.Pointer(&proc)))
        
        if int(rt)!=1 {break}
        // fmt.Println(string(proc.szExeFile[0:]))
        for j,i :=range _pros{
            // fmt.Println(string(proc.szExeFile[0:len(_pros[i])]))
            // if len(_pros[i])!=len(proc.szExeFile){continue}
            
            for q,v:=range i{
                if proc.szExeFile[q]!=v {break}
                if q+1==len(i) {res[j]+=1}
            }
            // fmt.Println("")

            
            // if proc.szExeFile[:len(_pros[i])]==_pros[i] {res[i]+=1}
        }
    }
    kernel32.NewProc("CloseHandle").Call(pHandle);
    // fmt.Println(time.Since(t))
	return res
}

func PStartf(pro []*exec.Cmd){
    for i := range pro {
        pro[i].SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
        pro[i].Start()
    }
}

func PRun(hide bool,prog string,cmd ...string){
    p:=exec.Command(prog,cmd...)
    if hide {p.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}}
    p.Run()
}

func PProxy(s, pacUrl string){
    if s=="off"{
        PRun(true,Cdir()+"/ref/sysproxy64.exe","off")
    }else{
        PRun(true,Cdir()+"/ref/sysproxy64.exe","pac",pacUrl)
    }
}
func Cdir()string{
    dir, _ := os.Executable()
    exPath := filepath.Dir(dir)
    return exPath
}
func PIsExist(f string) bool{
    _, err := os.Stat(f)
    return err == nil || os.IsExist(err)
}