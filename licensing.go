package licensing

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/rotisserie/eris"
)

var (
	iv = []byte{35, 46, 57, 24, 85, 35, 24, 74, 87, 35, 88, 98, 66, 32, 14, 05}
)

func errorHandle(err error) {
	fmt.Println("[ERROR]An error has occurred. Please contact your seller.")
	os.Exit(0)
}

func CheckFileExist(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

func encodeBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func decodeBase64(s string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return data, err
}

func CheckLicense(api string, insecureSSL bool) (*http.Response, error) {
	if !CheckFileExist("license.dat") {

		return nil, errors.New("license.dat not found")
	}

	li, err := ioutil.ReadFile("license.dat") //string(li)
	if err != nil {
		return nil, err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSSL},
	}
	client := &http.Client{Transport: tr}
	data := url.Values{}

	data.Set("license", string(li))
	u, _ := url.ParseRequestURI(api + "check")
	urlStr := fmt.Sprintf("%v", u)
	r, _ := http.NewRequest("POST", urlStr, bytes.NewBufferString(data.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(r)
	if err != nil {
		return nil, eris.Wrap(err, "unable to call license server")
	}

	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)

	// reset the read state
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(respBody))
	if resp.StatusCode == 200 {
		if string(respBody) != "Good" {
			if string(respBody) == "Expired" {
				return nil, errors.New("license is Expired")
			}
			return nil, errors.New("cannot verify license, Please contact your seller")
		}
	} else {
		return resp, errors.New("Request failed")
	}

	return resp, nil
}
