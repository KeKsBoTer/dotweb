package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/KeKsBoTer/dotweb"
)

func main() {
	config, err := dotweb.ConfigFromFlags(os.Args[1:])
	if err != nil {
		log.Fatalln(err)
	}

	config.Handler = func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "it works")
	}
	err = dotweb.StartWebServer(*config)
	if err != nil {
		log.Fatal(err)
	}
}
