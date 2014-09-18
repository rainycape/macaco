package macaco

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"code.google.com/p/go.net/html"
	"github.com/rainycape/otto"
)

const (
	nodeTypeText = iota + 1
	nodeTypeElement
	nodeTypeDocument
	nodeTypeComment
	nodeTypeDoctype
)

type node struct {
	node *html.Node
	vm   *otto.Otto
}

func asNode(n *html.Node, vm *otto.Otto) *node {
	if n == nil {
		return nil
	}
	return &node{n, vm}
}

func (n *node) String() string {
	var buf bytes.Buffer
	if err := html.Render(&buf, n.node); err != nil {
		panic(err)
	}
	return buf.String()
}

func (n *node) Parent() *node {
	return asNode(n.node.Parent, n.vm)
}

func (n *node) Next() *node {
	return asNode(n.node.NextSibling, n.vm)
}

func (n *node) Prev() *node {
	return asNode(n.node.PrevSibling, n.vm)
}

func (n *node) FirstChild() *node {
	return asNode(n.node.FirstChild, n.vm)
}

func (n *node) LastChild() *node {
	return asNode(n.node.LastChild, n.vm)
}

func (n *node) Children() otto.Value {
	var children []*node
	for nn := n.node.FirstChild; nn != nil; nn = nn.NextSibling {
		children = append(children, asNode(nn, n.vm))
	}
	v, err := n.vm.ToValue(children)
	if err != nil {
		panic(err)
	}
	return v
}

func (n *node) Type() int {
	switch n.node.Type {
	case html.TextNode:
		return nodeTypeText
	case html.DocumentNode:
		return nodeTypeDocument
	case html.ElementNode:
		return nodeTypeElement
	case html.CommentNode:
		return nodeTypeComment
	case html.DoctypeNode:
		return nodeTypeDoctype
	}
	return 0
}

func (n *node) Data() string {
	return n.node.Data
}

func (n *node) Attr(name string) string {
	for _, v := range n.node.Attr {
		if v.Key == name {
			return v.Val
		}
	}
	return ""
}

func (n *node) matchArguments(call otto.FunctionCall) (string, map[string]string, error) {
	var name string
	var attrs map[string]string
	for _, arg := range call.ArgumentList {
		val, _ := arg.Export()
		switch x := val.(type) {
		case string:
			if name != "" {
				return "", nil, fmt.Errorf("unhandled argument %v", x)
			}
			name = x
		case map[string]interface{}:
			if attrs == nil {
				attrs = make(map[string]string)
			}
			for k, iv := range x {
				var v string
				rv := reflect.Indirect(reflect.ValueOf(iv))
				switch rv.Kind() {
				case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
					v = strconv.FormatInt(rv.Int(), 10)
				case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
					v = strconv.FormatUint(rv.Uint(), 10)
				case reflect.Float32, reflect.Float64:
					fv := rv.Float()
					if _, frac := math.Modf(fv); frac == 0 {
						v = strconv.FormatInt(int64(fv), 10)
					} else {
						v = strconv.FormatFloat(fv, 'g', -1, 64)
					}
				case reflect.String:
					v = rv.String()
				default:
					return "", nil, fmt.Errorf("can't match attribute on %v", iv)
				}
				attrs[k] = v
			}
		default:
			return "", nil, fmt.Errorf("invalid argument %v", x)
		}
	}
	return name, attrs, nil
}

func (n *node) Matches(call otto.FunctionCall) bool {
	name, attrs, err := n.matchArguments(call)
	if err != nil {
		panic(err)
	}
	return n.matches(name, attrs)
}

func (n *node) matches(name string, attrs map[string]string) bool {
	if n.node.Type == html.ErrorNode || n.node.Type == html.TextNode || n.node.Type == html.CommentNode {
		return false
	}
	if name != "" && strings.ToLower(n.node.Data) != strings.ToLower(name) {
		return false
	}
	for k, v := range attrs {
		val := n.Attr(k)
		switch {
		case v == "":
			if val == "" {
				return false
			}
		case v[0] == '|':
			if !strings.HasPrefix(val, v[1:]) {
				return false
			}
		case v[0] == '~':
			words := strings.Split(val, " ")
			contains := false
			for _, w := range words {
				if w != "" && w == v[1:] {
					contains = true
					break
				}
			}
			if !contains {
				return false
			}
		case v[0] == '*':
			if !strings.Contains(val, v[1:]) {
				return false
			}
		case v[0] == '$':
			if !strings.HasSuffix(val, v[1:]) {
				return false
			}
		case v[0] == '=':
			if val != v[1:] {
				return false
			}
		default:
			if val != v {
				return false
			}

		}
	}
	return true
}

func (n *node) Find(call otto.FunctionCall) otto.Value {
	name, attrs, err := n.matchArguments(call)
	if err != nil {
		panic(err)
	}
	var nodes []*node
	n.visit(n.node, func(node *html.Node) bool {
		nn := asNode(node, n.vm)
		if nn.matches(name, attrs) {
			nodes = append(nodes, nn)
		}
		return false
	})
	v, err := n.vm.ToValue(nodes)
	if err != nil {
		panic(err)
	}
	return v
}

func (n *node) appendText(buf *bytes.Buffer, node *html.Node) {
	if node.Type == html.TextNode {
		buf.WriteString(node.Data)
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		n.appendText(buf, c)
	}
}

func (n *node) Text() string {
	var buf bytes.Buffer
	n.appendText(&buf, n.node)
	return buf.String()
}

func (n *node) visit(node *html.Node, f func(*html.Node) bool) bool {
	if f(node) {
		return true
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if n.visit(c, f) {
			return true
		}
	}
	return false
}

func (n *node) Visit(call otto.FunctionCall) otto.Value {
	fn := call.Argument(0)
	if fn.IsFunction() {
		thisVal, err := n.vm.ToValue(nil)
		if err != nil {
			panic(err)
		}
		n.visit(n.node, func(node *html.Node) bool {
			nodeVal, err := n.vm.ToValue(asNode(node, n.vm))
			if err != nil {
				panic(err)
			}
			res, err := fn.Call(thisVal, nodeVal)
			if err != nil {
				panic(err)
			}
			if b, _ := res.ToBoolean(); b {
				return true
			}
			return false
		})
	}
	return otto.Value{}
}

func (c *Context) htmlParse(call otto.FunctionCall) otto.Value {
	doc, err := html.Parse(strings.NewReader(call.Argument(0).String()))
	if err != nil {
		c.Errorf("error parsing HTML: %s\n", err)
		return otto.Value{}
	}
	val, err := c.vm.ToValue(asNode(doc, c.vm))
	if err != nil {
		panic(err)
	}
	return val
}

func (c *Context) htmlParseFragment(call otto.FunctionCall) otto.Value {
	fragment := call.Argument(0).String()
	var ctx *html.Node
	arg1, _ := call.Argument(1).Export()
	if c, ok := arg1.(*node); ok {
		ctx = c.node
	}
	nodes, err := html.ParseFragment(strings.NewReader(fragment), ctx)
	if err != nil {
		c.Errorf("error parsing HTML fragment: %s\n", err)
		return otto.Value{}
	}
	values := make([]*node, len(nodes))
	for ii, v := range nodes {
		values[ii] = asNode(v, c.vm)
	}
	val, err := c.vm.ToValue(values)
	if err != nil {
		panic(err)
	}
	return val
}

func (c *Context) loadHTML(obj *otto.Object) {
	htmlObject := c.newMacacoObject("html")
	htmlObject.Set("parse", c.htmlParse)
	htmlObject.Set("parse_fragment", c.htmlParseFragment)
	htmlObject.Set("_parses_doctype_node", true)
	htmlObject.Set("escape", html.EscapeString)
	htmlObject.Set("unescape", html.UnescapeString)
}
