package mp4

import (
	"errors"
	"io"
	"testing"

	"github.com/dustin/go-humanize"
	. "github.com/qydysky/part/decoder/tool"
	pfile "github.com/qydysky/part/file"
	pslice "github.com/qydysky/part/slice"
)

func TestMain(t *testing.T) {
	f := pfile.New("0.mp4", 0, false)
	data, e := f.ReadAll(humanize.KByte, humanize.MByte*10)
	if !errors.Is(e, io.EOF) {
		t.Fatal(e)
	}

	fmp4decoder := NewFmp4Decoder()
	if _, e := fmp4decoder.Init_fmp4(data); e != nil {
		t.Fatal(e)
	}

	buf := pslice.New[byte]()
	if _, e = fmp4decoder.Search_stream_fmp4(data, buf); e != nil && !errors.Is(e, io.EOF) {
		t.Fatal(e)
	}

	t.Log(buf.Size())
}

func TestMain1(t *testing.T) {
	f := pfile.New("0.mp4", 0, false)
	data, e := f.ReadAll(humanize.KByte, humanize.MByte*10)
	if !errors.Is(e, io.EOF) {
		t.Fatal(e)
	}

	if boxs, e := decode(data, "ftyp"); e != nil {
		t.Fatal(e)
	} else {
		for i := 0; i < len(boxs); i++ {
			if boxs[i].n == "mdat" {
				mdat := boxs[i]
				t.Logf("box type: %s", data[mdat.i+4:mdat.i+8])
				t.Logf("box len: %d", Btoi32(data, mdat.i))
				t.Logf("box cont: %x ...", data[mdat.i+8:mdat.i+8+20])
				for i := mdat.i + 8; i < mdat.e; {
					nvlL := Btoi32(data, i)
					i += 4
					// t.Logf("nvl l: %d", nvlL)
					t.Logf("nvl h: %x %x %d", data[i], data[i]&0x1f, nvlL)
					if data[i]&0x1f == 5 || data[i]&0x1f == 1 {
						t.Logf("%0.8b", data[i+1:i+20])
						// 	r := NewBitsReader(data[i+1 : i+int(nvlL)])
						// 	first_mb_in_slice := r.UE()
						// t.Logf("first_mb_in_slice: %d", first_mb_in_slice)
						// 	return
					}
					i += int(nvlL)
				}
				return
			}
		}
	}
}
