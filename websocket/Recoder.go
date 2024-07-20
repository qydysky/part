package part

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/dustin/go-humanize"
	ctx "github.com/qydysky/part/ctx"
	file "github.com/qydysky/part/file"
	funcCtrl "github.com/qydysky/part/funcCtrl"
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
	stopflag context.Context
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

		t.stopflag = ctx.CarryCancel(context.WithCancel(context.Background()))

		if startTimeStamp.IsZero() {
			startTimeStamp = time.Now()
		}
		t.Server.Interface().Pull_tag(map[string]func(interface{}) bool{
			`send`: func(data interface{}) bool {
				if ctx.Done(t.stopflag) {
					return true
				}
				if tmp, ok := data.(Uinterface); ok {
					f.Write([]byte(fmt.Sprintf("%f,%d,%s\n", time.Since(startTimeStamp).Seconds(), tmp.Id, tmp.Data)), true)
					f.Sync()
				}
				return false
			},
		})
		<-t.stopflag.Done()
	}()
	return nil
}

func (t *Recorder) Stop() {
	ctx.CallCancel(t.stopflag)
	t.onlyOnce.UnSet()
}

func Play(filePath string) (s *Server, close func()) {
	sg := ctx.CarryCancel(context.WithCancel(context.Background()))

	s = New_server()

	close = func() {
		s.Interface().Push_tag(`close`, uinterface{
			Id:   0,
			Data: `rev_close`,
		})
		_ = ctx.CallCancel(sg)
	}

	go func() {
		f := file.New(filePath, 0, false)
		defer f.Close()

		timer := time.NewTicker(time.Second)
		defer timer.Stop()

		var (
			cu   atomic.Int64
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
					default:
						if d, err := strconv.ParseFloat(data, 64); err == nil && d > 0 {
							cu.Store(int64(d))
							s.Interface().Push_tag(`send`, Uinterface{
								Id:   0, //send to all
								Data: []byte("ok"),
							})
						}
					}
				}
				return false
			},
		})

		for !ctx.Done(sg) {
			select {
			case <-timer.C:
			case <-sg.Done():
				return
			}

			cu.Add(1)

			for !ctx.Done(sg) {
				if data == nil {
					if data, e = f.ReadUntil([]byte{'\n'}, 70, humanize.MByte); e != nil && !errors.Is(e, io.EOF) {
						panic(e)
					}
					if len(data) == 0 {
						return
					}
				}

				tIndex := bytes.Index(data, []byte{','})
				if d, _ := strconv.ParseFloat(string(data[:tIndex]), 64); d < float64(cu.Load()) {
					danmuIndex := tIndex + bytes.Index(data[tIndex+2:], []byte{','}) + 3
					s.Interface().Push_tag(`send`, Uinterface{
						Id:   0, //send to all
						Data: data[danmuIndex:],
					})
					data = nil
				} else {
					break
				}
			}
		}
	}()

	return
}
