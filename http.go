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
	idx++
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
	opts := call.Argument(idx)
	idx++
	var cache bool
	if opts.IsObject() {
		obj := opts.Object()
		for _, k := range obj.Keys() {
			val, err := obj.Get(k)
			if err != nil {
				c.Errorf("error getting object key %q: %s", k, err)
				return otto.Value{}
			}
			switch strings.ToLower(k) {
			case "headers":
				if !val.IsObject() {
					c.Errorf("headers must be an object")
					return otto.Value{}
				}
				hobj := val.Object()
				for _, hk := range hobj.Keys() {
					hval, err := hobj.Get(hk)
					if err != nil {
						c.Errorf("error getting object key %q: %s", k, err)
						return otto.Value{}
					}
					req.Header.Add(hk, hval.String())
				}
			case "cache":
				cache, _ = val.ToBoolean()
			}
		}
	}
	if len(qs) > 0 && methodHasBody(method) {
		req.Body = &readerCloser{strings.NewReader(qs)}
		if method == "POST" {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	if cache && !methodHasBody(method) {
		// Try cache
		if entry, err := c.cache.cachedEntry(u); err == nil {
			c.Debugf("cached response from %s\n", u)
			return c.newHTTPResponse(u, entry.URL, entry.Data, entry.StatusCode, entry.Header)
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
	if cache && !methodHasBody(method) {
		// Save into cache
		if err := c.cache.cacheData(u, body, resp); err != nil {
			c.Debugf("error caching response from %s: %s\n", u, err)
		}
	}
	return c.newHTTPResponse(u, resp.Request.URL.String(), body, resp.StatusCode, resp.Header)
}

func (c *Context) newHTTPResponse(reqURL string, respURL string, body []byte, statusCode int, headers http.Header) otto.Value {
	respHeaders := make(map[string]string, len(headers))
	for k := range headers {
		respHeaders[k] = headers.Get(k)
	}
	return c.mustCallValue("new M.http.Response", nil, respURL, string(body), statusCode, reqURL, respHeaders).val
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
