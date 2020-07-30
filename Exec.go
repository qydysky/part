package part

import (
	"os"
	"bufio"
	// "fmt"
	Ppart "github.com/qydysky/part/linuxwin"
	"io"
	"os/exec"
	"errors"
	"strings"
)

func Log (cmd *exec.Cmd,filename string) error {
	var newpath func(string) error = func (filename string)error{
		/*
			如果filename路径不存在，就新建它
		*/	
		var exist func(string) bool = func (s string) bool {
			_, err := os.Stat(s)
			return err == nil || os.IsExist(err)
		}
	
		for i:=0;true;{
			a := strings.Index(filename[i:],"/")
			if a == -1 {break}
			if a == 0 {a = 1}//bug fix 当绝对路径时开头的/导致问题
			i=i+a+1
			if !exist(filename[:i-1]) {
				err := os.Mkdir(filename[:i-1], os.ModePerm)
				if err != nil {return err}
			}
		}
		
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			fd,err:=os.Create(filename)
			if err != nil {
				return err
			}else{
				fd.Close()
			}
		}
		return nil
	}
	if err:=newpath(filename);err != nil {return err}

	fd, err := os.OpenFile(filename, os.O_RDWR | os.O_EXCL, 0755)
	if err != nil {return err}

	wb := bufio.NewWriter(fd)
	
	var w func(string) = func(s string) {
		_, err2 := wb.WriteString(s+"\n")
		if err2 != nil {
			Logf().E(err2.Error())
		}
	}
	defer func(){
		w("Log stop!")
		wb.Flush()
		fd.Sync()
		fd.Close()
	}()

	w("Log start!")
	var s string
	for _,v:= range cmd.Args {
		s+=v+" ";
	}
	w("cmd: "+s)

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		w(err.Error())
		return errors.New(err.Error())
	}

	Startf(cmd)

	reader := bufio.NewReader(stdout)

	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		w(line)
	}

	if err:=cmd.Wait();err !=nil{return err}
	return nil
}

func Run(hide bool,prog string,cmd ...string){
    Ppart.PRun(hide,prog,cmd ...)
}

func Startf(pro ...*exec.Cmd){
    Ppart.PStartf(pro)
}

func Stop(pro ...*exec.Cmd){
    for i := range pro {
        pro[i].Process.Kill()
    }
}

func Cmd () {

}