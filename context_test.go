package macaco

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
)

const (
	remoteScript = "http://cdnjs.cloudflare.com/ajax/libs/Base64/0.3.0/base64.min.js"
)

func newTestingContext(t testing.TB) *Context {
	ctx, err := NewContext()
	if err != nil {
		t.Fatal(err)
	}
	if testing.Verbose() {
		ctx.verbose = true
	}
	return ctx
}

func TestLogging(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	ctx := newTestingContext(t)
	ctx.Stdout = &stdout
	ctx.Stderr = &stderr
	if _, err := ctx.Run("M.logf('%s', 'foo'); M.errorf('%s', 'bar')"); err != nil {
		t.Fatal(err)
	}
	if s1 := stdout.String(); s1 != "foo" {
		t.Errorf("expecting stdout = \"foo\", got %q", s1)
	}
	if s2 := stderr.String(); s2 != "bar" {
		t.Errorf("expecting stderr = \"bar\", got %q", s2)
	}
}

func TestJSON(t *testing.T) {
	ctx := newTestingContext(t)
	// Make number float64, so its type is not altered.
	obj := map[string]interface{}{"a": float64(1), "b": true, "c": "12345"}
	val, err := ctx.Call("JSON.stringify", nil, obj)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("JSON is %s", val.String())
	res, err := ctx.Call("JSON.parse", nil, val.String())
	if err != nil {
		t.Fatal(err)
	}
	obj2 := res.Interface()
	if !reflect.DeepEqual(obj, obj2.(map[string]interface{})) {
		t.Errorf("JSON trip altered object from %v to %v", obj, obj2)
	}
}

func testResponseURL(t *testing.T, res *Value, expected string) {
	url, err := res.Get("url")
	if err != nil {
		t.Fatal(err)
	}
	if url.String() != expected {
		t.Errorf("expecting URL %q, got %v", expected, url)
	}
	jsonValue, err := res.Method("toJSON")
	if err != nil {
		t.Fatal(err)
	}
	m := jsonValue.Interface()
	if err != nil {
		t.Fatal(err)
	}
	if ru := m.(map[string]interface{})["url"].(string); ru != expected {
		t.Errorf("expecting response URL %q, got %q", expected, ru)
	}
}

func TestHTTP(t *testing.T) {
	const (
		getURL  = "http://httpbin.org/get"
		postURL = "http://httpbin.org/post"
		qs      = "foo=bar&foo2=bar2"
	)
	var stdout bytes.Buffer
	ctx := newTestingContext(t)
	ctx.Stdout = &stdout
	ctx.verbose = false
	// Sync calls
	res1, err := ctx.Call("M.http.get", nil, getURL)
	if err != nil {
		t.Fatal(err)
	}
	testResponseURL(t, res1, getURL)
	res2, err := ctx.Call("M.http.get", nil, getURL, qs)
	if err != nil {
		t.Fatal(err)
	}
	testResponseURL(t, res2, getURL+"?"+qs)
	res3, err := ctx.Call("M.http.get", nil, getURL+"?a=b", qs)
	if err != nil {
		t.Fatal(err)
	}
	testResponseURL(t, res3, getURL+"?"+"a=b&"+qs)
	res4, err := ctx.Call("M.http.get", nil, getURL, map[string]string{"foo": "bar", "foo2": "bar2"})
	if err != nil {
		t.Fatal(err)
	}
	testResponseURL(t, res4, getURL+"?"+qs)
	// AsyncCall
	stdout.Reset()
	_, err = ctx.Call(`(function(url) {
            M.http.post(url, function(resp) {
                var val = resp.toJSON();
                console.logf('%s', val.url);
            });
        })`, nil, postURL)
	if err != nil {
		t.Fatal(err)
	}
	if stdout.String() != postURL {
		t.Errorf("expecting post URL %q, got %q", postURL, stdout.String())
	}
}

func TestHTTPError(t *testing.T) {
	const invalidURL = "http://this-domain-does-not-exist-because-whatever-lets-hope-no-one-registers-it.foobar"
	ctx := newTestingContext(t)
	res, err := ctx.Call("M.http.get", nil, invalidURL)
	if err != nil {
		t.Fatal(err)
	}
	e, err := res.Get("error")
	if err != nil {
		t.Fatal(err)
	}
	msg, err := e.Method("message")
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := msg.Interface().(string); !ok || s == "" {
		t.Error("expecting non-empty error")
	}
}

func TestGlobals(t *testing.T) {
	ctx := newTestingContext(t)
	hasMacaco := false
	for _, v := range ctx.Globals() {
		if v == "macaco" {
			hasMacaco = true
			break
		}
	}
	if !hasMacaco {
		t.Errorf("macaco not found in globals %v", ctx.Globals())
	}
	ctx.Run("not_a_macaco = {}")
	hasNoMacaco := false
	for _, v := range ctx.Globals() {
		if v == "not_a_macaco" {
			hasNoMacaco = true
			break
		}
	}
	if !hasNoMacaco {
		t.Errorf("not_a_macaco not found in globals %v", ctx.Globals())
	}
}

func TestLoadRemote(t *testing.T) {
	ctx := newTestingContext(t)
	val1, err := ctx.Run("typeof btoa")
	if err != nil {
		t.Fatal(err)
	}
	if val1.String() != "undefined" {
		t.Fatal("btoa already defined")
	}
	if err := ctx.Load(remoteScript); err != nil {
		t.Fatal(err)
	}
	val2, err := ctx.Run("typeof btoa")
	if err != nil {
		t.Fatal(err)
	}
	if val2.String() != "function" {
		t.Errorf("expecting typeof btoa = function, got %v", val2.Interface())
	}
	ctx2 := newTestingContext(t)
	_, err = ctx2.Run(fmt.Sprintf("macaco.load(%q)", remoteScript))
	if err != nil {
		t.Fatal(err)
	}
	val3, err := ctx.Run("typeof btoa")
	if err != nil {
		t.Fatal(err)
	}
	if val3.String() != "function" {
		t.Errorf("expecting typeof btoa = function, got %v", val3.Interface())
	}
}

func TestLoadCache(t *testing.T) {
	ctx := newTestingContext(t)
	ctx.verbose = true
	var stdout bytes.Buffer
	ctx.Stdout = &stdout
	if err := ctx.Load(remoteScript); err != nil {
		t.Fatal(err)
	}
	if err := ctx.Load(remoteScript); err != nil {
		t.Fatal(err)
	}
	if strings.Count(stdout.String(), "GET") > 1 {
		t.Errorf("script downloaded multiple times: %v", stdout.String())
	}
	stdout.Reset()
	ctx2 := newTestingContext(t)
	ctx2.verbose = true
	ctx2.Stdout = &stdout
	if err := ctx2.Load(remoteScript); err != nil {
		t.Fatal(err)
	}
	if strings.Count(stdout.String(), "GET") > 0 {
		t.Errorf("script downloaded rather than loaded from disk: %v", stdout.String())
	}
}

func TestTests(t *testing.T) {
	ctx := newTestingContext(t)
	_, err := ctx.Run(`
	    function __testA() {
		console.log('a');
	    }
	    function __testB() {
		console.error('b');
	    }
	`)
	if !testing.Verbose() {
		ctx.Stdout = ioutil.Discard
		ctx.Stderr = ioutil.Discard
	}
	if err != nil {
		t.Fatal(err)
	}
	results, err := ctx.RunTests(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expecting 2 results, got %d", len(results))
	}
	ta, tb := results[0], results[1]
	if ta.Name != "A" {
		t.Errorf("expecting first test name A, got %q", ta.Name)
	}
	if !ta.Passed() {
		t.Error("test A not passed")
	}
	if tb.Name != "B" {
		t.Errorf("expecting second test name B, got %q", tb.Name)
	}
	if tb.Passed() {
		t.Error("test B not passed")
	}
	if len(tb.Errors) != 1 {
		t.Errorf("expecting one error in test B, got %d", len(tb.Errors))
	}
}

func TestSprintf(t *testing.T) {
	const expect = "a = a, b = 1"
	ctx := newTestingContext(t)
	res, err := ctx.Run(`
	    macaco.fmt.sprintf('a = %s, b = %d', 'a', 1);
	`)
	if err != nil {
		t.Fatal(err)
	}
	if res.String() != expect {
		t.Errorf("expecting %q, got %q instead", expect, res.String())
	}
}
