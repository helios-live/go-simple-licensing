package Licensing

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
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

func decodeBase64(s string) []byte {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		errorHandle(err)
	}
	return data
}

func Encrypt(key, text string) string {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		errorHandle(err)
	}
	plaintext := []byte(text)
	cfb := cipher.NewCFBEncrypter(block, iv)
	ciphertext := make([]byte, len(plaintext))
	cfb.XORKeyStream(ciphertext, plaintext)
	return encodeBase64(ciphertext)
}

func Decrypt(key, text string) string {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		errorHandle(err)
	}
	ciphertext := decodeBase64(text)
	cfb := cipher.NewCFBEncrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	cfb.XORKeyStream(plaintext, ciphertext)
	return string(plaintext)
}

func ForceCheckLicense(api string, ssl, silent bool) {
	if !CheckFileExist("license.dat") {
		if !silent {
			fmt.Println("license.dat not found.")
		}
		os.Exit(0)
	}

	li, err := ioutil.ReadFile("license.dat") //string(li)
	if err != nil {
		if !silent {
			errorHandle(err)
		}
	}

	client := &http.Client{}
	data := url.Values{}
	data.Set("license", string(li))
	u, _ := url.ParseRequestURI(api + "check")
	urlStr := fmt.Sprintf("%v", u)
	r, _ := http.NewRequest("POST", urlStr, bytes.NewBufferString(data.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(r)
	if err != nil {
		if !silent {
			fmt.Println("Unable to connect to license server.")
		}
		os.Exit(0)
	}
	defer resp.Body.Close()
	resp_body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == 200 {
		if string(resp_body) != "Good" {
			if string(resp_body) == "Expired" {
				if !silent {
					fmt.Println("License is Expired.")
				}
				os.Exit(0)
			} else {
				if !silent {
					fmt.Println("Connot verify license, Please contact your seller.")
				}
				os.Exit(0)
			}
		}
	}
}

func CheckLicense(api string, ssl, silent bool) (bool, error) {
	if !CheckFileExist("license.dat") {

		return false, errors.New("license.dat not found")
	}

	li, err := ioutil.ReadFile("license.dat") //string(li)
	if err != nil {
		return false, err
	}

	client := &http.Client{}
	data := url.Values{}

	data.Set("license", string(li))
	u, _ := url.ParseRequestURI(api + "check")
	urlStr := fmt.Sprintf("%v", u)
	r, _ := http.NewRequest("POST", urlStr, bytes.NewBufferString(data.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(r)
	if err != nil {
		return false, errors.New("unable to connect to license server")
	}

	defer resp.Body.Close()
	resp_body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == 200 {
		if string(resp_body) != "Good" {
			if string(resp_body) == "Expired" {
				return false, errors.New("license is Expired")
			} else {
				return false, errors.New("cannot verify license, Please contact your seller")
			}
		}
	}

	return true, nil
}
