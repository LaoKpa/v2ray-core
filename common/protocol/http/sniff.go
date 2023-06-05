package http

import (
	"bytes"
	"context"
	"errors"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"strings"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/net"
)

type version byte

const (
	HTTP1 version = iota
	HTTP2
)

type SniffHeader struct {
	version version
	host    string
	method  string
	path    string
}

func (h *SniffHeader) Protocol() string {
	switch h.version {
	case HTTP1:
		return "http1"
	case HTTP2:
		return "http2"
	default:
		return "unknown"
	}
}

func (h *SniffHeader) Domain() string {
	return h.host
}

var (
	// refer to https://pkg.go.dev/net/http@master#pkg-constants
	methods = [...]string{"get", "post", "head", "put", "delete", "options", "connect", "patch", "trace"}

	errNotHTTPMethod = errors.New("not an HTTP method")
)

func beginWithHTTPMethod(b []byte) error {
	for _, m := range &methods {
		if len(b) >= len(m) && strings.EqualFold(string(b[:len(m)]), m) {
			return nil
		}

		if len(b) < len(m) {
			return common.ErrNoClue
		}
	}

	return errNotHTTPMethod
}

func SniffHTTP(c context.Context, b []byte) (*SniffHeader, error) {
	if err := beginWithHTTPMethod(b); err != nil {
		return nil, err
	}
	sh := &SniffHeader{
		version: HTTP1,
	}

	//method string,path string,pototol string

	headers := bytes.Split(b, []byte{'\n'})
	header0 := headers[0]
	parts := bytes.SplitN(header0, []byte{' '}, 3)
	if len(parts) >= 2 {
		sh.method = strings.ToLower(string(parts[0]))
		sh.path = strings.ToLower(string(parts[1]))
		//pototol := strings.ToLower(string(parts[2]))
	}

	content := session.ContentFromContext(c)
	if content == nil {
		content = new(session.Content)
		c = session.ContextWithContent(c, content)
	}
	content.SetAttribute(":method", strings.ToUpper(sh.method))
	content.SetAttribute(":path", sh.path)
	//for key := range request.Header {
	//	value := request.Header.Get(key)
	//	content.SetAttribute(strings.ToLower(key), value)
	//}

	//c = session.ContextWithContent(c, content)

	for i := 1; i < len(headers); i++ {
		header := headers[i]
		if len(header) == 0 {
			break
		}
		parts := bytes.SplitN(header, []byte{':'}, 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(string(parts[0]))
		if key == "host" {
			rawHost := strings.ToLower(string(bytes.TrimSpace(parts[1])))
			dest, err := ParseHost(rawHost, net.Port(80))
			if err != nil {
				return nil, err
			}
			sh.host = dest.Address.String()
		}
	}

	if len(sh.host) > 0 {
		return sh, nil
	}

	return nil, common.ErrNoClue
}
