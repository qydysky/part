package xml

import (
	"bytes"
	"io"
	"iter"
	"strings"
)

type nodeParseState int

const (
	unname nodeParseState = iota
	nameing
	unattr
	attring
	innering
)

type Node struct {
	Name  []byte
	Attr  []byte
	Inner []byte
	state nodeParseState
}

func (t *Node) clear() {
	t.Name = t.Name[:0]
	t.Attr = t.Attr[:0]
	t.Inner = t.Inner[:0]
}

func (t *Node) ToString() string {
	var b strings.Builder
	b.WriteByte('<')
	b.Write(t.Name)
	if len(t.Attr) > 0 {
		b.WriteByte(' ')
	}
	b.Write(t.Attr)
	b.WriteByte('>')
	b.Write(t.Inner)
	return b.String()
}

func NewDecoder(r io.Reader) iter.Seq[*Node] {
	node := Node{}
	return func(yield func(*Node) bool) {
		buf := make([]byte, 1)
		for {
			if n, e := r.Read(buf); n > 0 {
				switch node.state {
				case unname:
					if buf[0] == '<' {
						node.state = nameing
					}
				case nameing:
					switch buf[0] {
					case ' ':
						node.state = attring
					case '>':
						node.state = innering
					default:
						node.Name = append(node.Name, buf[0])
					}
				case attring:
					switch buf[0] {
					case '>':
						node.state = innering
					default:
						node.Attr = append(node.Attr, buf[0])
					}
				case innering:
					switch buf[0] {
					case '<':
						node.Attr = bytes.TrimSpace(node.Attr)
						if !yield(&node) {
							node.clear()
							node.state = nameing
							return
						}
						node.clear()
						node.state = nameing
					default:
						node.Inner = append(node.Inner, buf[0])
					}
				}
			} else if e == io.EOF {
				break
			} else {
				panic(e)
			}
		}
	}
}
