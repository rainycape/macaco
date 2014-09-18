package macaco

import (
	"bufio"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	programNameRe = regexp.MustCompile("^[0-9a-zA-Z][0-9a-zA-Z\\-]+$")
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

func macacoDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, ".macaco"), nil
}

func expandToken(tok string) string {
	if tok != "" {
		if dir, _ := macacoDir(); dir != "" {
			tokensFile := filepath.Join(dir, "tokens")
			if f, err := os.Open(tokensFile); err == nil {
				defer f.Close()
				r := bufio.NewReader(f)
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						break
					}
					fields := strings.Split(line, "=")
					if len(fields) < 2 {
						break
					}
					if strings.TrimSpace(fields[0]) == tok {
						tok = strings.TrimSpace(fields[1])
						break
					}
				}
			}
		}
	}
	return tok
}
