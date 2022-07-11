package envloader

import (
	"bufio"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/kamalshkeir/kago/core/utils/logger"
)



func Load(files ...string) error {
	if len(files) == 0 {
		files = []string{".env"}
	}

	for _, f := range files {
		err := loadFile(f)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("no data")
}

func LoadToMap(files ...string) (map[string]string,error) {
	if len(files) == 0 {
		files = []string{".env"}
	}

	for _, f := range files {
		m,err := loadToMap(f)
		if err != nil {
			return nil,err
		}
		return m,nil
	}
	return nil,errors.New("no data")
}

func loadToMap(filename string) (map[string]string,error) {
	// ouvrir le fichier
	file, err := os.Open(filename)
	if err != nil {
		return nil,err
	}
	defer file.Close()

	m, err := parse(file)
	if logger.CheckError(err) {
		return nil,err
	}

	envActuel := map[string]bool{}
	rawEnv := os.Environ()
	for _, rawEnvLine := range rawEnv {
		key := strings.Split(rawEnvLine, "=")[0]
		envActuel[key] = true
	}
	
	for key, value := range m {
		if !envActuel[key] {
			os.Setenv(key, value)
		}
	}

	return m,nil
}

func loadFile(filename string) error {
	// ouvrir le fichier
	file, err := os.Open(filename)
	if logger.CheckError(err) {
		return err
	}
	defer file.Close()

	m, err := parse(file)
	if logger.CheckError(err) {
		return err
	}

	envActuel := map[string]bool{}
	rawEnv := os.Environ()
	for _, rawEnvLine := range rawEnv {
		key := strings.Split(rawEnvLine, "=")[0]
		envActuel[key] = true
	}

	for key, value := range m {
		if !envActuel[key] {
			os.Setenv(key, value)
		}
	}

	return nil
}

func parse(r io.Reader) (m map[string]string, err error) {
	m = make(map[string]string)

	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err = scanner.Err(); err != nil {
		return
	}

	for _, fullLine := range lines {
		if !isIgnoredLine(fullLine) {
			var key, value string
			key, value, err = parseLine(fullLine, m)

			if err != nil {
				return
			}
			m[key] = value
		}
	}
	return
}

func parseLine(line string, envMap map[string]string) (key string, value string, err error) {
	if len(line) == 0 {
		err = errors.New("zero length string")
		return
	}

	// ditch the comments (but keep quoted hashes)
	if strings.Contains(line, "#") {
		segmentsBetweenHashes := strings.Split(line, "#")
		quotesAreOpen := false
		var segmentsToKeep []string
		for _, segment := range segmentsBetweenHashes {
			if strings.Count(segment, "\"") == 1 || strings.Count(segment, "'") == 1 {
				if quotesAreOpen {
					quotesAreOpen = false
					segmentsToKeep = append(segmentsToKeep, segment)
				} else {
					quotesAreOpen = true
				}
			}

			if len(segmentsToKeep) == 0 || quotesAreOpen {
				segmentsToKeep = append(segmentsToKeep, segment)
			}
		}

		line = strings.Join(segmentsToKeep, "#")
	}

	firstEquals := strings.Index(line, "=")
	firstColon := strings.Index(line, ":")
	splitString := strings.SplitN(line, "=", 2)
	if firstColon != -1 && (firstColon < firstEquals || firstEquals == -1) {
		//this is a yaml-style line
		splitString = strings.SplitN(line, ":", 2)
	}

	if len(splitString) != 2 {
		err = errors.New("can't separate key from value")
		return
	}

	// Parse the key
	key = splitString[0]
	key = strings.TrimPrefix(key, "export")
	key = strings.Trim(key, " ")

	// Parse the value
	value = parseValue(splitString[1], envMap)
	return
}

func parseValue(value string, envMap map[string]string) string {

	// trim
	value = strings.Trim(value, " ")

	// check if we've got quoted values or possible escapes
	if len(value) > 1 {
		rs := regexp.MustCompile(`\A'(.*)'\z`)
		singleQuotes := rs.FindStringSubmatch(value)

		rd := regexp.MustCompile(`\A"(.*)"\z`)
		doubleQuotes := rd.FindStringSubmatch(value)

		if singleQuotes != nil || doubleQuotes != nil {
			// pull the quotes off the edges
			value = value[1 : len(value)-1]
		}

		if doubleQuotes != nil {
			// expand newlines
			escapeRegex := regexp.MustCompile(`\\.`)
			value = escapeRegex.ReplaceAllStringFunc(value, func(match string) string {
				c := strings.TrimPrefix(match, `\`)
				switch c {
				case "n":
					return "\n"
				case "r":
					return "\r"
				default:
					return match
				}
			})
			// unescape characters
			e := regexp.MustCompile(`\\([^$])`)
			value = e.ReplaceAllString(value, "$1")
		}

		if singleQuotes == nil {
			value = expandVariables(value, envMap)
		}
	}

	return value
}

func expandVariables(v string, m map[string]string) string {
	r := regexp.MustCompile(`(\\)?(\$)(\()?\{?([A-Z0-9_]+)?\}?`)

	return r.ReplaceAllStringFunc(v, func(s string) string {
		submatch := r.FindStringSubmatch(s)

		if submatch == nil {
			return s
		}
		if submatch[1] == "\\" || submatch[2] == "(" {
			return submatch[0][1:]
		} else if submatch[4] != "" {
			return m[submatch[4]]
		}
		return s
	})
}

func isIgnoredLine(line string) bool {
	trimmedLine := strings.Trim(line, " \n\t")
	return len(trimmedLine) == 0 || strings.HasPrefix(trimmedLine, "#")
}


