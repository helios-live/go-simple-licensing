package main

import (
	"fmt"
	"log"

	Licensing "github.com/ideatocode/go-simple-licensing"
)

func init() {
	res, err := Licensing.CheckLicense("https://licensing.ideatocode.tech/", false)
	if err != nil {
		sc := -1
		if res != nil {
			sc = res.StatusCode
		}
		log.Fatalln("Failed to check license: status:", sc, err)
	}
	log.Println("res:", res, "err:", err)
}

func main() {
	fmt.Println("License was varified!")
	for {
	}
}
