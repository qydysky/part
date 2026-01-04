package xml

import (
	"bytes"
	"fmt"
	"iter"
	"strings"
	"testing"
)

func Test1(t *testing.T) {
	text := `
<?xml version="1.0" encoding="utf-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN"
  "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">

<html xmlns="http://www.w3.org/1999/xhtml">
<head>
    <title>Chapter</title>
    <link href="../Styles/fonts.css" type="text/css" rel="stylesheet"/>
    <link href="../Styles/main.css" type="text/css" rel="stylesheet"/>
</head>
<body>
<h2 class="head"><span  class="chapter-sequence-number" >第一章</span><br/>xxx</h2>
<p>1</p>
<p>2</p>
</body>
</html>
	`
	o := NewDecoder(strings.NewReader(text))
	fmt.Println(getHeader(o))
	fmt.Println(getP(o))
	// var title byte
	// for _, line := range NewDecoder(strings.NewReader(text)) {
	// 	if len(line.Name) == 2 && line.Name[0] == 'h' && line.Name[1] >= '1' && line.Name[1] <= '9' {
	// 		title = line.Name[1]
	// 	}
	// 	if len(line.Name) == 3 && line.Name[0] == '/' && line.Name[1] == 'h' && line.Name[2] == title {
	// 		title = 0
	// 	}
	// 	if title != 0 {
	// 		fmt.Printf("%s", line.Inner)
	// 	}
	// }
}

func getHeader(i iter.Seq[*Node]) (header string) {
	var title byte
	for line := range i {
		if len(line.Name) == 3 && line.Name[0] == '/' && line.Name[1] == 'h' && line.Name[2] == title {
			break
		}
		if len(line.Name) == 2 && line.Name[0] == 'h' && line.Name[1] >= '1' && line.Name[1] <= '9' {
			title = line.Name[1]
		}
		if title != 0 {
			if b := bytes.TrimSpace(line.Inner); len(b) == 0 {
				if len(header) != 0 && !strings.HasSuffix(header, " ") {
					header += " "
				}
			} else {
				header += string(b)
			}
		}
	}
	return
}

func getP(i iter.Seq[*Node]) (body string) {
	for line := range i {
		if len(line.Name) == 1 && line.Name[0] == 'p' {
			body += "\t" + string(line.Inner) + "\n"
		}
	}
	return
}
