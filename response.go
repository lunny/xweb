package xweb

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

type ResponseType int

const (
	AutoResponse  = iota + 1
	JsonResponse
	XmlResponse
)

type ResponseWriter struct {
	resp       http.ResponseWriter
	buffer     []byte
	StatusCode int
	header     http.Header
}

func NewResponseWriter(resp http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		resp:       resp,
		buffer:     make([]byte, 0),
		StatusCode: 0,
		header:     make(map[string][]string),
	}
}

func (r *ResponseWriter) Header() http.Header {
	return r.header
}

func (r *ResponseWriter) Write(data []byte) (int, error) {
	if r.StatusCode == 0 {
		r.StatusCode = http.StatusOK
	}
	r.buffer = append(r.buffer, data...)
	return len(data), nil
}

func (r *ResponseWriter) Written() bool {
	return r.StatusCode != 0
}

func (r *ResponseWriter) WriteHeader(code int) {
	r.StatusCode = code
}

func (r *ResponseWriter) ServeFile(req *http.Request, path string) error {
	http.ServeFile(r, req, path)
	if r.StatusCode != http.StatusOK {
		return errors.New("serve file failed")
	}
	return nil
}

func (r *ResponseWriter) ServeReader(rd io.Reader) error {
	ln, err := io.Copy(r, rd)
	if err != nil {
		return err
	}
	r.Header().Set("Content-Length", strconv.Itoa(int(ln)))
	return nil
}

func (r *ResponseWriter) ServeXml(obj interface{}) error {
	dt, err := xml.Marshal(obj)
	if err != nil {
		return err
	}
	r.Header().Set("Content-Type", "application/xml")
	_, err = r.Write(dt)
	return err
}

func (r *ResponseWriter) ServeJson(obj interface{}) error {
	dt, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	r.Header().Set("Content-Type", "application/json")
	_, err = r.Write(dt)
	return err
}

func (r *ResponseWriter) Download(fpath string) error {
	f, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer f.Close()

	fName := filepath.Base(fpath)
	r.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%v\"", fName))
	_, err = io.Copy(r, f)
	return err
}

func redirect(w http.ResponseWriter, url string, status ...int) error {
	s := 302
	if len(status) > 0 {
		s = status[0]
	}
	w.Header().Set("Location", url)
	w.WriteHeader(s)
	_, err := w.Write([]byte("Redirecting to: " + url))
	return err
}

func (w *ResponseWriter) Redirect(url string, status ...int) error {
	return redirect(w, url, status...)
}

// Notmodified writes a 304 HTTP response
func (w *ResponseWriter) NotModified() {
	w.WriteHeader(http.StatusNotModified)
}

// NotFound writes a 404 HTTP response
func (w *ResponseWriter) NotFound(message ...string) error {
	if len(message) == 0 {
		return w.Abort(http.StatusNotFound, "Not Found")
	}
	return w.Abort(http.StatusNotFound, message[0])
}

// Abort is a helper method that sends an HTTP header and an optional
// body. It is useful for returning 4xx or 5xx errors.
// Once it has been called, any return value from the handler will
// not be written to the response.
func (w *ResponseWriter) Abort(status int, body string) error {
	w.WriteHeader(status)
	w.Write([]byte(body))
	return nil
}

// SetHeader sets a response header. the current value
// of that header will be overwritten .
func (w *ResponseWriter) SetHeader(key string, value string) {
	w.Header().Set(key, value)
}

func (r *ResponseWriter) Flush() error {
	//fmt.Println("responsewriter:", r)

	if r.StatusCode == 0 {
		r.StatusCode = http.StatusOK
	}
	r.resp.WriteHeader(r.StatusCode)
	for key, value := range r.header {
		//fmt.Println("=====", key, value)
		if len(value) == 1 {
			r.resp.Header().Set(key, value[0])
		} else {
			for _, v := range value {
				r.resp.Header().Add(key, v)
			}
		}
	}

	_, err := r.resp.Write(r.buffer)
	if err != nil {
		return err
	}

	if flusher, ok := r.resp.(http.Flusher); ok {
		//fmt.Println("flush------>")
		flusher.Flush()
	}
	return nil
}

type ResponseInterface interface {
	SetResponse(*ResponseWriter)
}

type HttpResponseInterface interface {
	SetResponse(http.ResponseWriter)
}

type Responses struct {
}

func (ii *Responses) Intercept(ctx *Context) {
	if action := ctx.Action(); action != nil {
		if s, ok := action.(HttpResponseInterface); ok {
			s.SetResponse(ctx.Resp())
		}

		if s, ok := action.(ResponseInterface); ok {
			s.SetResponse(ctx.Resp())
		}
	}

	ctx.Invoke()
}
