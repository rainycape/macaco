package macaco

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

const (
	api = "http://localhost:8888/api/v1"
)

type Macaco struct {
	Token string
}

func (m *Macaco) Upload(name string, src interface{}) error {
	if !ProgramNameIsValid(name) {
		return fmt.Errorf("program name %q is not valid", name)
	}
	if m.Token == "" {
		return errors.New("can't upload without access_token")
	}
	var data []byte
	switch x := src.(type) {
	case string:
		var buf bytes.Buffer
		w := zip.NewWriter(&buf)
		err := filepath.Walk(x, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			fw, err := w.Create(filepath.ToSlash(path))
			if err != nil {
				return err
			}
			if _, err := io.Copy(fw, f); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		data = buf.Bytes()
	case *os.File:
		return m.Upload(name, x.Name())
	case io.Reader:
		var err error
		data, err = ioutil.ReadAll(x)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid source type %T", src)
	}
	if err := ValidateProgramZipData(data); err != nil {
		return err
	}
	values := make(url.Values)
	values.Set("name", name)
	values.Set("access_token", m.Token)
	p := api + "/upload?" + values.Encode()
	resp, err := http.Post(p, "application/zip", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := validateHTTPResponse(resp); err != nil {
		return fmt.Errorf("error uploading program %s: %s", name, err)
	}
	return nil
}
