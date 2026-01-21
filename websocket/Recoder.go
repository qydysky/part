package part

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	ctx "github.com/qydysky/part/ctx"
	pe "github.com/qydysky/part/errors"
	file "github.com/qydysky/part/file"
	funcCtrl "github.com/qydysky/part/funcCtrl"
	us "github.com/qydysky/part/unsafe"
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

	f := file.Open(filePath)
	if f.IsExist() {
		return ErrIsExist
	}
	f.Create()

	t.stopflag = ctx.CarryCancel(context.WithCancel(context.Background()))

	go func() {
		defer f.Close()

		var startTimeStamp time.Time

		if startTimeStamp.IsZero() {
			startTimeStamp = time.Now()
		}
		t.Server.Interface().Pull_tag(map[string]func(Uinterface) bool{
			`send`: func(tmp Uinterface) bool {
				if ctx.Done(t.stopflag) {
					return true
				}
				f.Write([]byte(fmt.Sprintf("%f,%d,%s\n", time.Since(startTimeStamp).Seconds(), tmp.Id, tmp.Data)))
				f.Sync()
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

// 在回放结束时，将会主动断开客户连接
func Play(filePath string) (s *Server, close func()) {
	return Plays(func(reg func(filepath string, start, dur time.Duration) error) {
		reg(filePath, 0, 0)
	})
}

var (
	ErrPlays       = pe.Action(`Plays`)
	ErrCallPlay    = ErrPlays.New(`ErrCallPlay`)
	ErrFileNoExist = ErrPlays.New(`ErrFileNoExist`)
)

// 当新连接建立时，总是从开头开始
//
// 当需要跳转到指定时刻时，需要向服务器发送需要跳转的时刻，单位秒
//
// 暂停向服务器发送pause,播放发送play
//
// reg filepath指向Recorder的文件路径，start是初始偏移值（默认0），dur是时长（默认0，当发送最后一条到客户端后关闭连接）
//
// 在回放到末尾时，将会主动断开客户连接
func Plays(regF func(reg func(filepath string, start, dur time.Duration) error)) (s *Server, close func()) {
	sg := ctx.CarryCancel(context.WithCancel(context.Background()))

	s = New_server()
	serMq := s.Interface()

	timer := time.NewTicker(time.Millisecond * 500)

	close = func() {
		// 退出断开所有连接
		serMq.Push_tag(`close`, Uinterface{
			Id: 0,
		})
		timer.Stop()
		_ = ctx.CallCancel(sg)
	}

	type rec struct {
		start, op, dur time.Duration
		file           *file.File
		next           *rec
	}

	var (
		rootRec  = &rec{}
		rangeRec = func(each func(*rec) (stop bool)) (stopRec *rec) {
			stopRec = rootRec
			for {
				if each(stopRec) || stopRec.next == nil || stopRec.next.file == nil {
					break
				}
				stopRec = stopRec.next
			}
			return
		}
	)

	// 构建播放序列
	{
		var tmp *rec = rootRec
		regF(func(filepath string, start, dur time.Duration) error {
			if !file.IsExist(filepath) {
				return ErrFileNoExist
			}
			_ = rangeRec(func(r *rec) (stop bool) {
				tmp.op += r.dur - r.start
				return false
			})
			tmp.start = start
			tmp.dur = dur
			tmp.file = file.Open(filepath)
			tmp.next = &rec{}
			tmp = tmp.next
			return nil
		})
		if rootRec.file == nil {
			return
		}
	}

	// 注册请求初始化事件
	serMq.Pull_tag_async_only(`init`, func(u Uinterface) (disable bool) {
		var (
			clientId = u.Id
			lock     sync.Mutex
			seed     bool
			paused   bool
			// 由于公共时钟0.5s触发一次，cu会在进入处理之前先+0.5,故此处-0.5以避免丢失开头0.5s的数据
			cu    float64 = -0.5
			cuRec *rec    = rootRec
			data  []byte
		)

		defer serMq.Push_tag(`close`, Uinterface{
			Id: clientId,
		})

		serMq.Pull_tag_only(`recv`, func(d Uinterface) (disable bool) {
			if d.Id != 0 && d.Id != clientId {
				return false
			}

			lock.Lock()
			defer lock.Unlock()

			switch data := us.B2S(d.Data); data {
			case "pause":
				paused = true
			case "play":
				paused = false
			default:
				// 上报时刻与当前时刻差超2s,调整时间
				if t, e := strconv.ParseFloat(data, 64); e == nil && math.Abs(cu-t) > 2 {
					cuRec = rangeRec(func(r *rec) (stop bool) {
						return r.op.Seconds() <= t && t <= r.op.Seconds()+r.dur.Seconds()
					})
					if cuRec.file != nil {
						_ = cuRec.file.SeekIndex(0, file.AtOrigin)
					}
					// 由于公共时钟0.5s触发一次，cu会在进入处理之前先+0.5,故此处-0.5以避免丢失开头0.5s的数据
					cu = t - 0.5
					seed = true
				}
			}
			return false
		})

		// 监听退出
		cancleFin, fin := serMq.Pull_tag_chan(`fin`, 1, sg)
		defer cancleFin()
		// 监听退出
		cancle, err := serMq.Pull_tag_chan(`error`, 1, sg)
		defer cancle()

		for !ctx.Done(sg) {
			select {
			case <-timer.C:
			case <-err:
				return
			case <-fin:
				return
			case <-sg.Done():
				return
			}

			func() {
				lock.Lock()
				defer lock.Unlock()
				if paused {
					return
				}

				cu += 0.5

				for !ctx.Done(sg) {
					// data中未存在未清除的数据
					if len(data) == 0 {
						// 读取一行
						if e := cuRec.file.ReadUntilV2(&data, []byte{'\n'}, 70, humanize.MByte); e != nil {
							if !errors.Is(e, io.EOF) {
								panic(e)
							} else if len(data) > 0 {
								//存在未处理数据
							} else if cuRec.dur == 0 || cuRec.next.file == nil {
								// io.EOF 并且是无限长（dur==0）or 最后一个文件
								// 退出
								cancleFin()
								return
							} else if cu < cuRec.op.Seconds()+cuRec.dur.Seconds() {
								// 当前时间仍位于本段设定的时间段，继续等待
								break
							} else {
								// 下一个文件
								cuRec = cuRec.next
								break
							}
						}
					}

					tIndex := bytes.Index(data, []byte{','})
					d, _ := strconv.ParseFloat(us.B2S(data[:tIndex]), 64)
					// 处理start
					if d < cuRec.start.Seconds() {
						// 时刻位于start之前，清空数据
						data = data[:0]
						continue
					}

					d = d + cuRec.op.Seconds() - cuRec.start.Seconds()

					// 处理seed
					if seed {
						seed = d < cu
						if seed {
							// 时间跳转，清空数据
							data = data[:0]
							continue
						} else {
							break
						}
					}

					if d < cu {
						// 读取到的行在当前时刻之前
						danmuIndex := tIndex + bytes.Index(data[tIndex+2:], []byte{','}) + 3
						serMq.Push_tag(`send`, Uinterface{
							Id:   clientId,
							Data: data[danmuIndex:],
						})
						// 清空数据
						data = data[:0]
					} else {
						// 读取到了时刻之后的行，保留数据
						break
					}
				}

			}()
		}

		return false
	})
	return
}
