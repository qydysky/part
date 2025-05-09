package part

import (
	"encoding/json"
	"iter"
	"net"
	"net/http"
	"strings"
	"time"
)

type CookieJson struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`

	Path       string `json:"Path"`       // optional
	Domain     string `json:"Domain"`     // optional
	Expires    string `json:"Expires"`    // time.Time in RFC1123
	RawExpires string `json:"RawExpires"` // for reading cookies only

	// MaxAge=0 means no 'Max-Age' attribute specified.
	// MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'
	// MaxAge>0 means Max-Age attribute present and given in seconds
	MaxAge   int           `json:"MaxAge"`
	Secure   bool          `json:"Secure"`
	HttpOnly bool          `json:"HttpOnly"`
	SameSite http.SameSite `json:"SameSite"` // Go 1.11
	Raw      string        `json:"Raw"`
	Unparsed []string      `json:"Unparsed"` // Raw text of unparsed attribute-value pairs
}

func (t *CookieJson) ToJson() ([]byte, error) {
	return json.Marshal(t)
}

func (t *CookieJson) FromJson(cookie_in_json []byte) error {
	return json.Unmarshal(cookie_in_json, t)
}

func (t *CookieJson) ToCookie() (http.Cookie, error) {
	exp, e := time.Parse(time.RFC1123, t.Expires)
	if exp.IsZero() {
		exp = time.Now().Add(time.Duration(t.MaxAge) * time.Second)
	}
	return http.Cookie{
		Name:       t.Name,
		Value:      t.Value,
		Path:       t.Path,
		Domain:     t.Domain,
		Expires:    exp,
		RawExpires: t.RawExpires,
		MaxAge:     t.MaxAge,
		Secure:     t.Secure,
		HttpOnly:   t.HttpOnly,
		SameSite:   t.SameSite,
		Raw:        t.Raw,
		Unparsed:   t.Unparsed,
	}, e
}

func (t *CookieJson) FromCookie(cookie *http.Cookie) {
	exp_t := cookie.Expires
	if exp_t.IsZero() {
		exp_t = time.Now().Add(time.Duration(cookie.MaxAge) * time.Second)
	}
	exp := exp_t.Format(time.RFC1123)

	t.Name = cookie.Name
	t.Value = cookie.Value
	t.Path = cookie.Path
	t.Domain = cookie.Domain
	t.Expires = exp
	t.RawExpires = cookie.RawExpires
	t.MaxAge = cookie.MaxAge
	t.Secure = cookie.Secure
	t.HttpOnly = cookie.HttpOnly
	t.SameSite = cookie.SameSite
	t.Raw = cookie.Raw
	t.Unparsed = cookie.Unparsed
}

func Cookies_String_2_Map(Cookies string) (o map[string]string) {
	o = make(map[string]string)
	list := strings.Split(Cookies, `; `)
	for _, v := range list {
		s := strings.SplitN(v, "=", 2)
		if len(s) != 2 {
			continue
		}
		o[s[0]] = s[1]
	}
	return
}

func Iter_2_Cookies_String(Cookies iter.Seq2[string, string]) (o string) {
	for k, v := range Cookies {
		o += k + `=` + v + `; `
	}
	t := []rune(o)
	o = string(t[:len(t)-2])
	return
}

func Map_2_Cookies_String(Cookies map[string]string) (o string) {
	if len(Cookies) == 0 {
		return ""
	}
	for k, v := range Cookies {
		o += k + `=` + v + `; `
	}
	t := []rune(o)
	o = string(t[:len(t)-2])
	return
}

func Cookies_List_2_Map(Cookies []*http.Cookie) (o map[string]string) {
	o = make(map[string]string)
	for _, v := range Cookies {
		o[v.Name] = v.Value
	}
	return
}

func Cookies_String_2_List(Cookies string) (o []*http.Cookie) {
	list := strings.Split(Cookies, `; `)
	for _, v := range list {
		s := strings.SplitN(v, "=", 2)
		if len(s) != 2 {
			continue
		}
		o = append(o, &http.Cookie{
			Name:  s[0],
			Value: s[1],
		})
	}
	return
}

func Cookies_Map_2_List(Cookies map[string]string) (o []*http.Cookie) {
	for k, v := range Cookies {
		o = append(o, &http.Cookie{
			Name:  k,
			Value: v,
		})
	}
	return
}

func Cookies_List_2_String(Cookies []*http.Cookie) (o string) {
	if len(Cookies) == 0 {
		return ""
	}
	for _, v := range Cookies {
		o += v.Name + `=` + v.Value + `; `
	}
	t := []rune(o)
	o = string(t[:len(t)-2])
	return
}

func ValidCookieDomain(v string) bool {
	if isCookieDomainName(v) {
		return true
	}
	if net.ParseIP(v) != nil && !strings.Contains(v, ":") {
		return true
	}
	return false
}

func isCookieDomainName(s string) bool {
	if len(s) == 0 {
		return false
	}
	if len(s) > 255 {
		return false
	}

	if s[0] == '.' {
		// A cookie a domain attribute may start with a leading dot.
		s = s[1:]
	}
	last := byte('.')
	ok := false // Ok once we've seen a letter.
	partlen := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		default:
			return false
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
			// No '_' allowed here (in contrast to package net).
			ok = true
			partlen++
		case '0' <= c && c <= '9':
			// fine
			partlen++
		case c == '-':
			// Byte before dash cannot be dot.
			if last == '.' {
				return false
			}
			partlen++
		case c == '.':
			// Byte before dot cannot be dot, dash.
			if last == '.' || last == '-' {
				return false
			}
			if partlen > 63 || partlen == 0 {
				return false
			}
			partlen = 0
		}
		last = c
	}
	if last == '-' || partlen > 63 {
		return false
	}

	return ok
}
