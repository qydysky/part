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

		if data, e := f.ReadAll(humanize.KByte, humanize.MByte); e != nil && !errors.Is(e, io.EOF) {
			panic(e)
		} else {
			var (
				cu    float64
				index int
			)

			sdata := bytes.Split(data, []byte{'\n'})

			s.Interface().Pull_tag(map[string]func(any) (disable bool){
				`recv`: func(a any) (disable bool) {
					if d, ok := a.(Uinterface); ok {
						switch string(d.Data) {
						case "pause":
							timer.Stop()
						case "play":
							timer.Reset(time.Second)
						default:
							tmp, _ := strconv.ParseFloat(string(d.Data), 64)
							if tmp < cu {
								for index > 0 && index < len(sdata) {
									tIndex := bytes.Index(sdata[index], []byte{','})
									if d, _ := strconv.ParseFloat(string(sdata[index][:tIndex]), 64); d > cu {
										index -= 1
										continue
									} else if d < cu {
										break
									}
								}
							}
							cu = tmp
						}
					}
					return false
				},
			})

			for sg.Islive() {
				<-timer.C
				cu += 1

				for index > 0 && index < len(sdata) {
					tIndex := bytes.Index(sdata[index], []byte{','})
					if d, _ := strconv.ParseFloat(string(sdata[index][:tIndex]), 64); d > cu+1 {
						break
					} else if d < cu {
						index += 1
						continue
					}
					index += 1

					danmuIndex := tIndex + bytes.Index(sdata[index][tIndex+2:], []byte{','}) + 3
					s.Interface().Push_tag(`send`, Uinterface{
						Id:   0, //send to all
						Data: sdata[index][danmuIndex:],
					})
				}

			}
		}
	}()

	return
}
