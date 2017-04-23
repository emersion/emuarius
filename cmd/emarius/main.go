package main

import (
	"log"
	"net/http"

	"github.com/ChimeraCoder/anaconda"
	"github.com/emersion/emuarius"
	"github.com/emersion/go-ostatus"
)

func main() {
	addr := ":4004"
	rootURL := "http://emuarius.emersion.fr"

	anaconda.SetConsumerKey("your-consumer-key")
	anaconda.SetConsumerSecret("your-consumer-secret")
	api := anaconda.NewTwitterApi("your-access-token", "your-access-token-secret")
	be := emuarius.NewBackend(api, rootURL)
	h := ostatus.NewHandler(be)

	s := &http.Server{
		Addr: addr,
		Handler: h,
	}

	log.Println("Starting server at", addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
