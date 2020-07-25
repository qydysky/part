// +build linux

package Ppart

import (
    "os"
    "os/exec"
    "strings"
    "strconv"
    "path/filepath"
)

func PCheck(pros []string) []int{
	res:=[]int{}
    _pros:=[][]byte{}

    for _,v:= range pros{
        if v=="" {return res}
        _pros=append(_pros,[]byte(v))
        res=append(res,0)
    }

    for j,i :=range _pros{
        cmd := exec.Command("pgrep","-c",string(i))
        output, _ := cmd.Output()
        outputt:=strings.Replace(string(output), "\n", "", -1)
        res[j],_=strconv.Atoi(outputt)
    }
    return res
}

func PStartf(pro []*exec.Cmd){
    for i := range pro {
        pro[i].Start()
    }
}

func PRun(hide bool,prog string,cmd ...string){
    p:=exec.Command(prog,cmd...)
    if hide {}
    p.Run()
}

func Cdir()string{
    dir, _ := os.Executable()
    exPath := filepath.Dir(dir)
    return exPath
}

func PProxy(s, pacUrl string){
    if s=="off" {
        PRun(true,"gsettings","set","org.gnome.system.proxy","mode","none");
        PRun(true,"kwriteconfig5","--file","kioslaverc","--group","'Proxy Settings'","--key","ProxyType","\"0\"");
    }else{
        PRun(true,"gsettings","set","org.gnome.system.proxy","autoconfig-url",pacUrl);
        PRun(true,"gsettings","set","org.gnome.system.proxy","mode","auto");
        PRun(true,"kwriteconfig5","--file","kioslaverc","--group","'Proxy Settings'","--key","ProxyType","\"2\"");
        PRun(true,"kwriteconfig5","--file","kioslaverc","--group","'Proxy Settings'","--key","Proxy Config Script","\""+pacUrl+"\"");
    }
}
