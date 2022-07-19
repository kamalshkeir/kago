package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	mrand "math/rand"
	"mime/multipart"
	"net/smtp"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils/logger"
)

type ContextKey string


// DELETE FILE
func DeleteFile(path string) error {
	err := os.Remove("."+path)
	if err != nil {
		return err
	} else {
		return nil
	}
}

// UPLOAD FILE
func UploadFile(file multipart.File,filename string, acceptedFormats ...string) (string,error) {
	//create destination file making sure the path is writeable.
	err := os.MkdirAll("media/uploads/",0770)
	if err != nil {
		return "",err
	}

	l := []string{"jpg","jpeg","png","json"}
	if len(acceptedFormats) > 0 {
		l=acceptedFormats
	}

	if StringContains(filename,l...) {
		dst, err := os.Create("media/uploads/" + filename)
		if err != nil {
			return "",err
		}
		defer dst.Close()

		//copy the uploaded file to the destination file
		if _, err := io.Copy(dst, file); err != nil {
			return "",err
		}else {
			url := "/media/uploads/"+filename
			return url,nil
		}
	} else {
		return "",errors.New("allowed extensions 'jpg','jpeg','png','json'")
	}
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
            var data, err1 = ioutil.ReadFile(filepath.Join(source, relPath))
            if err1 != nil {
                return err1
            }
            return ioutil.WriteFile(filepath.Join(destination, relPath), data, 0777)
        }
    })
    return err
}


func SliceContains[T comparable](elems []T, vs ...T) bool {
    for _, s := range elems {
		for _,v := range vs {
			if v == s {
				return true
			}
		}
    }
    return false
}

func StringContains(s string,subs ...string) bool {
	for _,sub := range subs {
		if strings.Contains(s,sub) {
			return true
		}
	}
	return false
}

// Send Email
func SendEmail(to_email string,subject string,textToSend string) {
	from := settings.GlobalConfig.SmtpEmail
	pass := settings.GlobalConfig.SmtpPass
	if pass == "" {
		logger.Error("CANNOT READ FROM ENV FILE")
	}

	to := []string{
		to_email,
	}

	smtpHost := settings.GlobalConfig.SmtpHost
  	smtpPort := settings.GlobalConfig.SmtpPort

	auth := smtp.PlainAuth("", from, pass, smtpHost)
	t, _ := template.ParseFiles("templates/email.html")
	var body bytes.Buffer
	mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body.Write([]byte(fmt.Sprintf("Subject: %s \n%s\n\n",subject, mimeHeaders)))
	t.Execute(&body, map[string]interface{}{
		"body":textToSend,
	})

	// Sending email.
	errMail := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, body.Bytes())
	if errMail != nil {
	  fmt.Println("errMail in SendEmail: ", errMail)
	  return
	}
}

// Cronjob like
func RunEvery(t time.Duration,function any) {
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

func RetryEvery(t time.Duration,function func() error,maxRetry ...int) {
	i := 0
	err := function()
	for err != nil {
		time.Sleep(t)
		i++
		if len(maxRetry) > 0 {
			if i < maxRetry[0] {
				err = function()
			} else {
				fmt.Println("stoping retry after",maxRetry,"times")
				break
			}
		} else {
			err = function()
		}
	}
}

// ReverseSlice
func ReverseSlice[S ~[]E, E any](s S)  {
    for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
        s[i], s[j] = s[j], s[i]
    }
}

func IsSameSlice[A []T ,B []T,T comparable](x []T,y []T) bool {
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


func randomizeStringSlice(slice []string) []string  {
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
		err = exec.Command("xdg-open",url).Start()
	case "windows":
		err = exec.Command("rundll32","url.dll,FileProtocolHandler",url).Start()
	case "darwin":
		err = exec.Command("open",url).Start()
	default:
	}
	_=err
}





