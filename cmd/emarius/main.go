package main

import (
	"log"
	"net/http"

	"github.com/boltdb/bolt"
	"github.com/ChimeraCoder/anaconda"
	"github.com/emersion/emuarius"
	"github.com/emersion/go-ostatus"
)

func main() {
	dbPath := "emuarius.db"
	addr := ":4004"
	rootURL := "http://localhost:4004"

	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	anaconda.SetConsumerKey("your-consumer-key")
	anaconda.SetConsumerSecret("your-consumer-secret")
	api := anaconda.NewTwitterApi("your-access-token", "your-access-token-secret")
	be := emuarius.NewBackend(api, db, rootURL)
	h := ostatus.NewHandler(be, rootURL)

	s := &http.Server{
		Addr:    addr,
		Handler: h,
	}

	log.Println("Starting server at", addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
