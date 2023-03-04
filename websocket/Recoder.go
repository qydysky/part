package part

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	file "github.com/qydysky/part/file"
	funcCtrl "github.com/qydysky/part/funcCtrl"
	signal "github.com/qydysky/part/signal"
)

var (
	ErrSerIsNil  = errors.New("ErrSerIsNil")
	ErrFileNoSet = errors.New("ErrFileNoSet")
	ErrhadStart  = errors.New("ErrhadStart")
	ErrIsExist   = errors.New("ErrIsExist")
)

type Recorder struct {
	Server   *Server
	onlyOnce funcCtrl.SkipFunc
	stopflag *signal.Signal
}

func (t *Recorder) Start(filePath string) error {
	if t.Server == nil {
		return ErrSerIsNil
	}
	if filePath == "" {
		return ErrFileNoSet
	}
	if t.onlyOnce.NeedSkip() {
		return ErrhadStart
	}

	f := file.New(filePath, 0, false)
	if f.IsExist() {
		return ErrIsExist
	}
	f.Create()

	go func() {
		defer f.Close()

		var startTimeStamp time.Time

		t.stopflag = signal.Init()
		if startTimeStamp.IsZero() {
			startTimeStamp = time.Now()
		}
		t.Server.Interface().Pull_tag(map[string]func(interface{}) bool{
			`send`: func(data interface{}) bool {
				if !t.stopflag.Islive() {
					return true
				}
				if tmp, ok := data.(Uinterface); ok {
					f.Write([]byte(fmt.Sprintf("%f,%d,%s\n", time.Since(startTimeStamp).Seconds(), tmp.Id, tmp.Data)), true)
					f.Sync()
				}
				return false
			},
		})
		t.stopflag.Wait()
	}()
	return nil
}

func (t *Recorder) Stop() {
	if t.stopflag.Islive() {
		t.stopflag.Done()
	}
	t.onlyOnce.UnSet()
}

func Play(filePath string) (s *Server, close func()) {
	sg := signal.Init()

	s = New_server()

	close = func() {
		s.Interface().Push_tag(`close`, uinterface{
			Id:   0,
			Data: `rev_close`,
		})
		sg.Done()
	}

	go func() {
		f := file.New(filePath, 0, false)
		defer f.Close()

		timer := time.NewTicker(time.Second)
		defer timer.Stop()

		var (
			cu   float64
			data []byte
			e    error
		)

		s.Interface().Pull_tag(map[string]func(any) (disable bool){
			`recv`: func(a any) (disable bool) {
				if d, ok := a.(Uinterface); ok {
					switch data := string(d.Data); data {
					case "pause":
						timer.Stop()
					case "play":
						timer.Reset(time.Second)
					}
				}
				return false
			},
		})

		for sg.Islive() {
			<-timer.C
			cu += 1

			for sg.Islive() {
				if data == nil {
					if data, e = f.ReadUntil('\n', 70, humanize.MByte); e != nil && !errors.Is(e, io.EOF) {
						panic(e)
					}
					if len(data) == 0 {
						return
					}
				}

				tIndex := bytes.Index(data, []byte{','})
				if d, _ := strconv.ParseFloat(string(data[:tIndex]), 64); d < cu {
					danmuIndex := tIndex + bytes.Index(data[tIndex+2:], []byte{','}) + 3
					s.Interface().Push_tag(`send`, Uinterface{
						Id:   0, //send to all
						Data: data[danmuIndex:],
					})
					data = nil
				}

				break
			}

		}
	}()

	return
}
