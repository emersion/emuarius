package main

import (
	"log"
	"net/http"
	"io/ioutil"

	"github.com/boltdb/bolt"
	"github.com/ChimeraCoder/anaconda"
	"github.com/emersion/emuarius"
	"github.com/emersion/go-ostatus"
	"github.com/pelletier/go-toml"
)

type config struct {
	Address string `toml:"address"`
	RootURL string `toml:"rootURL"`

	Twitter struct {
		ConsumerKey string `toml:"consumerKey"`
		ConsumerSecret string `toml:"consumerSecret"`
		AccessToken string `toml:"accessToken"`
		AccessTokenSecret string `toml:"accessTokenSecret"`
	} `toml:"twitter"`
}

const databasePath = "emuarius.db"

func main() {
	b, err := ioutil.ReadFile("emuarius.toml")
	if err != nil {
		log.Fatal(err)
	}

	var cfg config
	if err := toml.Unmarshal(b, &cfg); err != nil {
		log.Fatal(err)
	}

	db, err := bolt.Open(databasePath, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	anaconda.SetConsumerKey(cfg.Twitter.ConsumerKey)
	anaconda.SetConsumerSecret(cfg.Twitter.ConsumerSecret)
	api := anaconda.NewTwitterApi(cfg.Twitter.AccessToken, cfg.Twitter.AccessTokenSecret)
	be := emuarius.NewBackend(api, db, cfg.RootURL)
	h := ostatus.NewHandler(be, emuarius.HostMeta(cfg.RootURL))

	if err := emuarius.NewSubscriptionDB(h.Publisher, db); err != nil {
		log.Fatal(err)
	}

	s := &http.Server{
		Addr:    cfg.Address,
		Handler: h,
	}

	log.Println("Starting server at", cfg.Address)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
