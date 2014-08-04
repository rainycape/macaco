package macaco

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	programNameRe = regexp.MustCompile("^[0-9a-zA-Z]+$")
)

func ProgramNameIsValid(name string) bool {
	return programNameRe.MatchString(name)
}

func ValidateProgramZipData(data []byte) error {
	return nil
}

func SplitProgramName(fullName string) (userName string, programName string, versionName string) {
	userName, programName = path.Split(fullName)
	userName = strings.Trim(userName, "/")
	if dot := strings.IndexByte(programName, '.'); dot >= 0 {
		versionName = programName[dot+1:]
		programName = programName[:dot]
	}
	return
}

func ListProgramFiles(path string) ([]string, error) {
	var names []string
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.ToLower(filepath.Ext(p)) == ".js" {
			names = append(names, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return names, nil
}

func looksLikeURL(p string) bool {
	lower := strings.ToLower(p)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}
