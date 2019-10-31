package accesslog

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasttemplate"
)

const (
	DefaultPattern = `%{2006-01-02T15:04:05.999-0700}t %a %A %{Host}i "%r" %s - %T "%{X-Real-IP}i" "%{X-Forwarded-For}i" %{Content-Length}i - %{Content-Length}o %b %{CDN}i`
	JSONPattern    = `{"category":"access","@timestamp":"%{2006-01-02T15:04:05.999-0700}t","remote_addr":"%a","server_addr":"%A","host":"%{Host}i","request":"%r","status":"%s","first_byte_commit_time":"%F","request_time":"%T","http_x_real_ip":"%{X-Real-IP}i","http_x_forwarded_for":"%{X-Forwarded-For}i","content_length":"%{Content-Length}i","sent_http_content_length":"%{Content-Length}o","body_bytes_sent":"%b","http_cdn":"%{CDN}i"}`
	newLine        = byte('\n')
	hostHeader     = "Host"
)

type Item struct {
	URL            *url.URL
	RequestHeader  http.Header
	ResponseHeader http.Header
	RemoteAddr     string
	Method         string
	Proto          string
	ReceivedAt     time.Time
	FirstByteTime  time.Time
	Latency        time.Duration
	BytesSent      int
	StatusCode     int
}

func (i *Item) Reset() {
	*i = Item{}
}

func (i Item) QueryString() string {
	if i.URL.RawQuery != "" {
		return "?" + i.URL.RawQuery
	}

	return ""
}

type AccessLogger struct {
	options
	localIP  string
	template *fasttemplate.Template
}

type options struct {
	pattern string
	output  io.Writer
}

// Option is a function that sets some option on the client.
type Option func(c *options)

// Pattern specifies the access log output pattern
// Pattern Options:
//  %a - Remote IP address
//  %A - Local IP address
//  %b - Bytes sent, excluding HTTP headers, or '-' if no bytes were sent
//  %B - Bytes sent, excluding HTTP headers
//  %H - Request protocol
//  %m - Request method
//  %q - Query string (prepended with a '?' if it exists, otherwise an empty string
//  %r - First line of the request
//  %s - HTTP status code of the response
//  %t - Time the request was received, in the format "18/Sep/2011:19:18:28 -0400".
//  %U - Requested URL path
//  %D - Time taken to process the request, in millis
//  %T - Time taken to process the request, in seconds
//  %F - Time taken to commit the response, in millis
//  %{xxx}i - Incoming request headers
//  %{xxx}o - Outgoing response headers
//  %{xxx}t - Time the request was received, in the format of xxx
//
// Default pattern is `%{2006-01-02T15:04:05.999-0700}t %a - %{Host}i "%r" %s - %T "%{X-Real-IP}i" "%{X-Forwarded-For}i" %{Content-Length}i - %{Content-Length}o %b %{CDN}i`
func Pattern(p string) Option {
	return func(opts *options) {
		opts.pattern = p
	}
}

// Output specifies the access log output writer
func Output(w io.Writer) Option {
	return func(opts *options) {
		opts.output = w
	}
}

func NewLogger(opts ...Option) (*AccessLogger, error) {
	logOpts := options{
		pattern: DefaultPattern,
		output:  os.Stdout,
	}

	for _, opt := range opts {
		opt(&logOpts)
	}

	logger := &AccessLogger{
		options: logOpts,
	}

	err := logger.buildTemplate()
	if err != nil {
		return nil, err
	}

	logger.localIP, err = localIP()
	if err != nil {
		return nil, err
	}

	return logger, nil
}

func (a *AccessLogger) buildTemplate() (err error) {
	templateText := strings.NewReplacer(
		"%a", "${RemoteIP}",
		"%A", "${LocalIP}",
		"%b", "${BytesSent|-}",
		"%B", "${BytesSent|0}",
		"%H", "${Proto}",
		"%m", "${Method}",
		"%q", "${QueryString}",
		"%r", "${Method} ${RequestURI} ${Proto}",
		"%s", "${StatusCode}",
		"%t", "${ReceivedAt|02/Jan/2006:15:04:05 -0700}",
		"%U", "${URLPath}",
		"%D", "${Latency|ms}",
		"%T", "${Latency|s}",
		"%F", "${FirstByteTime|ms}",
	).Replace(string(a.pattern))

	timeFormatPattern := regexp.MustCompile("%(\\{([^\\}]+)\\}){1}t")
	templateText = timeFormatPattern.ReplaceAllString(templateText, "${ReceivedAt|$2}")
	requestHeaderPattern := regexp.MustCompile("%(\\{([^\\}]+)\\}){1}i")
	templateText = requestHeaderPattern.ReplaceAllStringFunc(templateText, strings.ToLower)
	templateText = requestHeaderPattern.ReplaceAllString(templateText, "${RequestHeader|$2}")
	responseHeaderPattern := regexp.MustCompile("%(\\{([^\\}]+)\\}){1}o")
	templateText = responseHeaderPattern.ReplaceAllStringFunc(templateText, strings.ToLower)
	templateText = responseHeaderPattern.ReplaceAllString(templateText, "${ResponseHeader|$2}")
	a.template, err = fasttemplate.NewTemplate(templateText, "${", "}")
	return
}

var dash = []byte("-")

var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 256))
	},
}

// log Execute the text template with the data derived from the request, and write to output.
func (a *AccessLogger) log(item *Item) error {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	_, err := a.template.ExecuteFunc(buf,
		func(w io.Writer, tag string) (int, error) {
			switch tag {
			case "RemoteIP":
				return w.Write([]byte(getRemoteIP(item.RemoteAddr)))
			case "LocalIP":
				return w.Write([]byte(a.localIP))
			case "BytesSent|-":
				if item.BytesSent == 0 {
					return w.Write(dash)
				}
				return w.Write([]byte(strconv.Itoa(item.BytesSent)))
			case "BytesSent|0":
				return w.Write([]byte(strconv.Itoa(item.BytesSent)))
			case "Proto":
				return w.Write([]byte(item.Proto))
			case "Method":
				return w.Write([]byte(item.Method))
			case "QueryString":
				return w.Write([]byte(item.QueryString()))
			case "RequestURI":
				return w.Write([]byte(item.URL.RequestURI()))
			case "URLPath":
				return w.Write([]byte(item.URL.Path))
			case "StatusCode":
				return w.Write([]byte(strconv.Itoa(item.StatusCode)))
			case "Latency|ms":
				return w.Write([]byte(strconv.FormatInt(item.Latency.Nanoseconds()/1000000, 10)))
			case "Latency|s":
				return w.Write([]byte(strconv.FormatFloat(item.Latency.Seconds(), 'f', 3, 64)))
			case "FirstByteTime|ms":
				if item.FirstByteTime.IsZero() {
					return w.Write(dash)
				}
				return w.Write([]byte(strconv.FormatInt(item.FirstByteTime.Sub(item.ReceivedAt).Nanoseconds()/1000000, 10)))
			default:
				if i := strings.Index(tag, "|"); i > 0 {
					switch string([]byte(tag)[:i]) {
					case "ReceivedAt":
						return w.Write([]byte(item.ReceivedAt.Format(string([]byte(tag)[i+1:]))))
					case "RequestHeader":
						hv := item.RequestHeader.Get(string([]byte(tag)[i+1:]))
						if hv == "" {
							return w.Write(dash)
						}
						return w.Write([]byte(hv))
					case "ResponseHeader":
						hv := item.ResponseHeader.Get(string([]byte(tag)[i+1:]))
						if hv == "" {
							return w.Write(dash)
						}
						return w.Write([]byte(hv))
					}
				}
			}
			return 0, nil
		})
	if err != nil {
		return err
	}

	buf.WriteByte(newLine)
	_, err = a.output.Write(buf.Bytes())
	return err
}

var itemPool = sync.Pool{
	New: func() interface{} {
		return &Item{}
	},
}

// Log write http access log to output.
func (a *AccessLogger) Log(w ResponseWriter, r *http.Request, t time.Time, d time.Duration) error {
	item := itemPool.Get().(*Item)
	defer itemPool.Put(item)
	item.Reset()

	// add Host header that deleted by net/http package
	// https://github.com/golang/go/commit/6e11f45ebdbc7b0ee1367c80ea0a0c0ec52d6db5
	// https://github.com/golang/go/issues/13134
	r.Header.Set(hostHeader, r.Host)

	item.URL = r.URL
	item.RequestHeader = r.Header
	item.RemoteAddr = r.RemoteAddr
	item.Method = r.Method
	item.Proto = r.Proto
	item.ReceivedAt = t
	item.Latency = d
	item.ResponseHeader = w.Header()
	item.BytesSent = w.Size()
	item.StatusCode = w.Status()
	item.FirstByteTime = w.FirstByteTime()

	return a.log(item)
}

func getRemoteIP(addr string) string {
	if remoteIP := strings.TrimSpace(addr); len(remoteIP) > 0 {
		return strings.SplitN(remoteIP, ":", 2)[0]
	}
	return ""
}

func localIP() (string, error) {
	addr, err := net.ResolveUDPAddr("udp", "1.2.3.4:1")
	if err != nil {
		return "", err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return "", err
	}

	defer conn.Close()

	host, _, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		return "", err
	}

	return host, nil
}
