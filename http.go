package macaco

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/rainycape/otto"
)

type readerCloser struct {
	io.Reader
}

func (r *readerCloser) Close() error {
	return nil
}

func methodHasBody(m string) bool {
	return m == "POST" || m == "PUT" || m == "PATCH"
}

func (c *Context) responseError(err error) otto.Value {
	val := c.errObject(err)
	resp := c.mustCallValue("new M.http.Response", nil)
	resp.Set("error", val.val)
	return resp.val
}

func (c *Context) sendHttpRequest(method string, call otto.FunctionCall) otto.Value {
	idx := 0
	if method == "" {
		method = call.Argument(0).String()
		idx = 1
	}
	method = strings.ToUpper(method)
	u := call.Argument(idx).String()
	idx++
	var qs string
	data := call.Argument(idx)
	if data.IsObject() {
		values := make(url.Values)
		obj := data.Object()
		for _, k := range obj.Keys() {
			val, err := obj.Get(k)
			if err != nil {
				c.Errorf("error getting object key %q: %s", k, err)
				return otto.Value{}
			}
			values.Add(k, val.String())
			qs = values.Encode()
		}
	} else if !data.IsUndefined() && !data.IsNull() {
		qs = data.String()
	}
	if len(qs) > 0 && !methodHasBody(method) {
		sep := "?"
		if strings.IndexByte(u, '?') >= 0 {
			sep = "&"
		}
		u += sep + qs
	}
	c.Debugf("%s - %s\n", method, u)
	req, err := http.NewRequest(method, u, nil)
	if err != nil {
		return c.responseError(err)
	}
	if len(qs) > 0 && methodHasBody(method) {
		req.Body = &readerCloser{strings.NewReader(qs)}
		if method == "POST" {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return c.responseError(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return c.responseError(err)
	}
	respHeaders := make(map[string]string, len(resp.Header))
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}
	return c.mustCallValue("new M.http.Response", nil, resp.Request.URL.String(), string(body), resp.StatusCode, u, respHeaders).val
}

func (c *Context) makeHttpRequest(method string, call otto.FunctionCall) otto.Value {
	val := c.sendHttpRequest(method, call)
	callback := call.Argument(len(call.ArgumentList) - 1)
	if callback.IsFunction() {
		callback.Call(otto.Value{}, val)
	}
	return val
}

func (c *Context) httpRequest(call otto.FunctionCall) otto.Value {
	return c.makeHttpRequest("", call)
}

func (c *Context) httpGet(call otto.FunctionCall) otto.Value {
	return c.makeHttpRequest("GET", call)
}

func (c *Context) httpPost(call otto.FunctionCall) otto.Value {
	return c.makeHttpRequest("POST", call)
}

func (c *Context) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Context) loadHTTP(obj *otto.Object) {
	httpObj := c.newMacacoObject("http")
	httpObj.Set("request", c.httpRequest)
	httpObj.Set("get", c.httpGet)
	httpObj.Set("post", c.httpPost)
}

func validateHTTPResponse(resp *http.Response) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		// TODO: errors as json?
		return fmt.Errorf("%v (status code %v)", string(body), resp.StatusCode)
	}
	return nil
}
