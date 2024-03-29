package utils

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	mrand "math/rand"
	"mime/multipart"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils/logger"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type ContextKey string

// DELETE FILE
func DeleteFile(path string) error {
	err := os.Remove("." + path)
	if err != nil {
		return err
	} else {
		return nil
	}
}

// Parse body multipart
func ParseMultipartForm(r *http.Request, size ...int64) (formData url.Values, formFiles map[string][]*multipart.FileHeader) {
	s := int64(32 << 20)
	if len(size) > 0 {
		s = size[0]
	}
	parseErr := r.ParseMultipartForm(s)
	if parseErr != nil {
		logger.Error("Parse error = ", parseErr)
	}
	defer func() {
		err := r.MultipartForm.RemoveAll()
		logger.CheckError(err)
	}()
	formData = r.Form
	formFiles = r.MultipartForm.File
	return formData, formFiles
}

// UPLOAD Multipart FILE
func UploadMultipartFile(file multipart.File, filename string, outPath string, acceptedFormats ...string) (string, error) {
	//create destination file making sure the path is writeable.
	if outPath == "" {
		outPath = settings.MEDIA_DIR + "/uploads/"
	} else {
		if !strings.HasSuffix(outPath, "/") {
			outPath += "/"
		}
	}
	err := os.MkdirAll(outPath, 0770)
	if err != nil {
		return "", err
	}

	l := []string{"jpg", "jpeg", "png", "json"}
	if len(acceptedFormats) > 0 {
		l = acceptedFormats
	}

	if StringContains(filename, l...) {
		dst, err := os.Create(outPath + filename)
		if err != nil {
			return "", err
		}
		defer dst.Close()

		//copy the uploaded file to the destination file
		if _, err := io.Copy(dst, file); err != nil {
			return "", err
		} else {
			url := "/" + outPath + filename
			return url, nil
		}
	} else {
		return "", fmt.Errorf("not in allowed extensions 'jpg','jpeg','png','json' : %v", l)
	}
}

// UPLOAD FILE
func UploadFileBytes(fileData []byte, filename string, outPath string, acceptedFormats ...string) (string, error) {
	//create destination file making sure the path is writeable.
	if outPath == "" {
		outPath = settings.MEDIA_DIR + "/uploads/"
	} else {
		if !strings.HasSuffix(outPath, "/") {
			outPath += "/"
		}
	}
	err := os.MkdirAll(outPath, 0770)
	if err != nil {
		return "", err
	}

	l := []string{"jpg", "jpeg", "png", "json"}
	if len(acceptedFormats) > 0 {
		l = acceptedFormats
	}

	if StringContains(filename, l...) {
		dst, err := os.Create(outPath + filename)
		if err != nil {
			return "", err
		}
		defer dst.Close()

		//copy the uploaded file to the destination file
		if _, err := io.Copy(dst, bytes.NewReader(fileData)); err != nil {
			return "", err
		} else {
			url := "/" + outPath + filename
			return url, nil
		}
	} else {
		return "", fmt.Errorf("not in allowed extensions 'jpg','jpeg','png','json' : %v", l)
	}
}

func UploadFile(received_filename, folder_out string, r *http.Request, acceptedFormats ...string) (string, []byte, error) {
	r.ParseMultipartForm(10 << 20) //10Mb
	defer func() {
		err := r.MultipartForm.RemoveAll()
		logger.CheckError(err)
	}()
	var buff bytes.Buffer
	file, header, err := r.FormFile(received_filename)
	if logger.CheckError(err) {
		return "", nil, err
	}
	defer file.Close()
	// copy the uploaded file to the buffer
	if _, err := io.Copy(&buff, file); err != nil {
		return "", nil, err
	}

	data_string := buff.String()

	// make DIRS if not exist
	err = os.MkdirAll(settings.MEDIA_DIR+"/"+folder_out+"/", 0664)
	if err != nil {
		return "", nil, err
	}
	// make file
	if len(acceptedFormats) == 0 {
		acceptedFormats = []string{"jpg", "jpeg", "png", "json"}
	}
	if StringContains(header.Filename, acceptedFormats...) {
		dst, err := os.Create(settings.MEDIA_DIR + "/" + folder_out + "/" + header.Filename)
		if err != nil {
			return "", nil, err
		}
		defer dst.Close()
		dst.Write([]byte(data_string))

		url := settings.MEDIA_DIR + "/" + folder_out + "/" + header.Filename
		return url, []byte(data_string), nil
	} else {
		return "", nil, fmt.Errorf("expecting filename to finish to be %v", acceptedFormats)
	}
}

func UploadFiles(received_filenames []string, folder_out string, r *http.Request, acceptedFormats ...string) ([]string, [][]byte, error) {
	_, formFiles := ParseMultipartForm(r)
	urls := []string{}
	datas := [][]byte{}
	for inputName, files := range formFiles {
		var buff bytes.Buffer
		if len(files) > 0 && SliceContains(received_filenames, inputName) {
			for _, f := range files {
				file, err := f.Open()
				if logger.CheckError(err) {
					return nil, nil, err
				}
				defer file.Close()
				// copy the uploaded file to the buffer
				if _, err := io.Copy(&buff, file); err != nil {
					return nil, nil, err
				}

				data_string := buff.String()

				// make DIRS if not exist
				err = os.MkdirAll(settings.MEDIA_DIR+"/"+folder_out+"/", 0664)
				if err != nil {
					return nil, nil, err
				}
				// make file
				if len(acceptedFormats) == 0 {
					acceptedFormats = []string{"jpg", "jpeg", "png", "json"}
				}
				if StringContains(f.Filename, acceptedFormats...) {
					dst, err := os.Create(settings.MEDIA_DIR + "/" + folder_out + "/" + f.Filename)
					if err != nil {
						return nil, nil, err
					}
					defer dst.Close()
					dst.Write([]byte(data_string))

					url := settings.MEDIA_DIR + "/" + folder_out + "/" + f.Filename
					urls = append(urls, url)
					datas = append(datas, []byte(data_string))
				} else {
					logger.Info(f.Filename, "not handled")
					return nil, nil, fmt.Errorf("expecting filename to finish to be %v", acceptedFormats)
				}
			}
		}

	}
	return urls, datas, nil
}

func CopyDir(source, destination string) error {
	var err error = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		var relPath string = strings.Replace(path, source, "", 1)
		if relPath == "" {
			return nil
		}
		if info.IsDir() {
			return os.Mkdir(filepath.Join(destination, relPath), 0755)
		} else {
			var data, err1 = os.ReadFile(filepath.Join(source, relPath))
			if err1 != nil {
				return err1
			}
			return os.WriteFile(filepath.Join(destination, relPath), data, 0777)
		}
	})
	return err
}

// Graceful Shutdown
func GracefulShutdown(f func() error) error {
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	<-s
	return f()
}

func SliceContains[T comparable](elems []T, vs ...T) bool {
	for _, s := range elems {
		for _, v := range vs {
			if v == s {
				return true
			}
		}
	}
	return false
}

func StringContains(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// Send Email
func SendEmail(to_email string, subject string, textToSend string) {
	from := settings.Config.Smtp.Email
	pass := settings.Config.Smtp.Pass
	if pass == "" {
		logger.Error("CANNOT READ FROM ENV FILE")
	}

	to := []string{
		to_email,
	}

	smtpHost := settings.Config.Smtp.Host
	smtpPort := settings.Config.Smtp.Port

	auth := smtp.PlainAuth("", from, pass, smtpHost)
	t, _ := template.ParseFiles("templates/email.html")
	var body bytes.Buffer
	mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body.Write([]byte(fmt.Sprintf("Subject: %s \n%s\n\n", subject, mimeHeaders)))
	t.Execute(&body, map[string]interface{}{
		"body": textToSend,
	})

	// Sending email.
	errMail := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, body.Bytes())
	if errMail != nil {
		fmt.Println("errMail in SendEmail: ", errMail)
		return
	}
}

// Cronjob like
func RunEvery(t time.Duration, function any) {
	//Usage : go RunEvery(2 * time.Second,func(){})
	fn, ok := function.(func())
	if !ok {
		fmt.Println("ERROR : fn is not a function")
		return
	}

	fn()
	c := time.NewTicker(t)

	for range c.C {
		fn()
	}
}

func RetryEvery(t time.Duration, function func() error, maxRetry ...int) {
	i := 0
	err := function()
	for err != nil {
		time.Sleep(t)
		i++
		if len(maxRetry) > 0 {
			if i < maxRetry[0] {
				err = function()
			} else {
				fmt.Println("stoping retry after", maxRetry, "times")
				break
			}
		} else {
			err = function()
		}
	}
}

// ReverseSlice
func ReverseSlice[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// IsSameSlice check if both slice are equal
func IsSameSlice[A []T, B []T, T comparable](x []T, y []T) bool {
	if len(x) != len(y) {
		return false
	}
	// create a map of string -> int
	diff := make(map[T]int, len(x))
	for _, _x := range x {
		// 0 value for int is 0, so just increment a counter for the string
		diff[_x]++
	}
	for _, _y := range y {
		// If the string _y is not in diff bail out early
		if _, ok := diff[_y]; !ok {
			return false
		}
		diff[_y] -= 1
		if diff[_y] == 0 {
			delete(diff, _y)
		}
	}
	return len(diff) == 0
}

func Difference[T comparable](slice1 []T, slice2 []T) []T {
	var diff []T

	// Loop two times, first to find slice1 strings not in slice2,
	// second loop to find slice2 strings not in slice1
	for i := 0; i < 2; i++ {
		for _, s1 := range slice1 {
			found := false
			for _, s2 := range slice2 {
				if s1 == s2 {
					found = true
					break
				}
			}
			// String not found. We add it to return slice
			if !found {
				diff = append(diff, s1)
			}
		}
		// Swap the slices, only if it was the first loop
		if i == 0 {
			slice1, slice2 = slice2, slice1
		}
	}

	return diff
}

func SliceRemove[T comparable](slice *[]T, elemsToRemove ...T) {
	for i, elem := range *slice {
		for _, e := range elemsToRemove {
			if e == elem {
				*slice = append((*slice)[:i], (*slice)[i+1:]...)
			}
		}
	}
}

func randomizeStringSlice(slice []string) []string {
	mrand.Seed(time.Now().UnixNano())
	mrand.Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})

	return slice
}

func ShuffleCharacters(text string) string {
	characters := strings.Split(text, "")
	randomCharacters := randomizeStringSlice(characters)

	return strings.Join(randomCharacters, "")
}

// Check if file exists
func PathExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// UUID
func GenerateUUID() (string, error) {
	var uuid [16]byte
	_, err := io.ReadFull(rand.Reader, uuid[:])
	if err != nil {
		return "", err
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10
	var buf [36]byte
	encodeHex(buf[:], uuid)
	return string(buf[:]), nil
}

func encodeHex(dst []byte, uuid [16]byte) {
	hex.Encode(dst, uuid[:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], uuid[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], uuid[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], uuid[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:], uuid[10:])
}

// Generate Random String
func GenerateRandomString(s int) string {
	b, _ := GenerateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b)
}

// Generate Random Bytes
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// err == nil only if len(b) == n
	if err != nil {
		return nil, err
	}

	return b, nil
}

// MemUsage
func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func OpenBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
	}
	_ = err
}

func GetIpCountry(ip string) string {
	cmd := exec.Command("nmap", "-n", "-sn", "-Pn", "--script", "ip-geolocation-geoplugin", ip)
	d, err := cmd.Output()
	if err != nil {
		return ""
	}
	bReader := bytes.NewReader(d)
	scanner := bufio.NewScanner(bReader)
	var line string
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "_location") {
			line = scanner.Text()
			sp := strings.Split(line, "location:")
			line = strings.TrimSpace(sp[len(sp)-1])
			spl := strings.Split(line, ",")
			spl[1] = strings.TrimSpace(spl[1])
			return spl[1]
		}
	}
	return ""
}

func IsUpper(s string) bool {
	for _, r := range s {
		if !unicode.IsUpper(r) && unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func IsLower(s string) bool {
	for _, r := range s {
		if !unicode.IsLower(r) && unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func ToSlug(s string) (string, error) {
	str := []byte(strings.ToLower(s))

	// convert all spaces to dash
	regE := regexp.MustCompile("[[:space:]]")
	str = regE.ReplaceAll(str, []byte("-"))

	// remove all blanks such as tab
	regE = regexp.MustCompile("[[:blank:]]")
	str = regE.ReplaceAll(str, []byte(""))

	// remove all punctuations with the exception of dash

	regE = regexp.MustCompile("[!/:-@[-`{-~]")
	str = regE.ReplaceAll(str, []byte(""))

	regE = regexp.MustCompile("/[^\x20-\x7F]/")
	str = regE.ReplaceAll(str, []byte(""))

	regE = regexp.MustCompile("`&(amp;)?#?[a-z0-9]+;`i")
	str = regE.ReplaceAll(str, []byte("-"))

	regE = regexp.MustCompile("`&([a-z])(acute|uml|circ|grave|ring|cedil|slash|tilde|caron|lig|quot|rsquo);`i")
	str = regE.ReplaceAll(str, []byte("\\1"))

	regE = regexp.MustCompile("`[^a-z0-9]`i")
	str = regE.ReplaceAll(str, []byte("-"))

	regE = regexp.MustCompile("`[-]+`")
	str = regE.ReplaceAll(str, []byte("-"))

	strReplaced := strings.Replace(string(str), "&", "", -1)
	strReplaced = strings.Replace(strReplaced, `"`, "", -1)
	strReplaced = strings.Replace(strReplaced, "&", "-", -1)
	strReplaced = strings.Replace(strReplaced, "--", "-", -1)

	if strings.HasPrefix(strReplaced, "-") || strings.HasPrefix(strReplaced, "--") {
		strReplaced = strings.TrimPrefix(strReplaced, "-")
		strReplaced = strings.TrimPrefix(strReplaced, "--")
	}

	if strings.HasSuffix(strReplaced, "-") || strings.HasSuffix(strReplaced, "--") {
		strReplaced = strings.TrimSuffix(strReplaced, "-")
		strReplaced = strings.TrimSuffix(strReplaced, "--")
	}

	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	slug, _, err := transform.String(t, strReplaced)

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(slug), nil
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func SnakeCaseToTitle(inputUnderScoreStr string) (camelCase string) {
	//snake_case to camelCase
	isToUpper := false
	for k, v := range inputUnderScoreStr {
		if k == 0 {
			camelCase = strings.ToUpper(string(inputUnderScoreStr[0]))
		} else {
			if isToUpper {
				camelCase += strings.ToUpper(string(v))
				isToUpper = false
			} else {
				if v == '_' {
					isToUpper = true
				} else {
					camelCase += string(v)
				}
			}
		}
	}
	return
}

/* Private Ip */
func GetPrivateIp() string {
	pIp := getOutboundIP()
	if pIp == "" {
		pIp = resolveHostIp()
		if pIp == "" {
			pIp = getLocalPrivateIps()[0]
		}
	}
	return pIp
}

func resolveHostIp() string {
	netInterfaceAddresses, err := net.InterfaceAddrs()

	if err != nil {
		return ""
	}

	for _, netInterfaceAddress := range netInterfaceAddresses {
		networkIp, ok := netInterfaceAddress.(*net.IPNet)
		if ok && !networkIp.IP.IsLoopback() && networkIp.IP.To4() != nil {
			ip := networkIp.IP.String()
			return ip
		}
	}

	return ""
}

func getLocalPrivateIps() []string {
	ips := []string{}
	host, _ := os.Hostname()
	addrs, _ := net.LookupIP(host)
	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			ips = append(ips, ipv4.String())
		}
	}
	return ips
}

func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	if localAddr.IP.To4().IsPrivate() {
		return localAddr.IP.String()
	}
	return ""
}
