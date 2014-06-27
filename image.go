package macaco

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strings"

	"github.com/robertkrimen/otto"
)

type _image struct {
	image.Image
	format string
}

func (i *_image) Width() int {
	return i.Bounds().Dx()
}

func (i *_image) Height() int {
	return i.Bounds().Dy()
}

func (i *_image) Format() string {
	return i.format
}

type info struct {
	cfg    image.Config
	format string
}

func (i *info) Width() int {
	return i.cfg.Width
}

func (i *info) Height() int {
	return i.cfg.Height
}

func (i *info) Format() string {
	return i.format
}

func imageReader(call otto.FunctionCall) (io.Reader, error) {
	return strings.NewReader(call.Argument(0).String()), nil
}

func decodeImage(call otto.FunctionCall) otto.Value {
	r, err := imageReader(call)
	if err != nil {
		panic(err)
	}
	im, format, err := image.Decode(r)
	if err != nil {
		panic(err)
	}
	v, err := call.Otto.ToValue(&_image{im, strings.ToUpper(format)})
	if err != nil {
		panic(err)
	}
	return v
}

func decodeImageInfo(call otto.FunctionCall) otto.Value {
	r, err := imageReader(call)
	if err != nil {
		panic(err)
	}
	cfg, format, err := image.DecodeConfig(r)
	if err != nil {
		panic(err)
	}
	v, err := call.Otto.ToValue(&info{cfg, strings.ToUpper(format)})
	if err != nil {
		panic(err)
	}
	return v
}

func (c *Context) loadImage(obj *otto.Object) {
	imageObj := c.newMacacoObject("image")
	imageObj.Set("decode", decodeImage)
	imageObj.Set("decodeInfo", decodeImageInfo)
}
