package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/ChimeraCoder/anaconda"
	"github.com/boltdb/bolt"
	"github.com/emersion/emuarius"
	"github.com/emersion/go-ostatus"
	"github.com/pelletier/go-toml"
)

type TwitterConfig struct {
	ConsumerKey       string `toml:"consumerKey"`
	ConsumerSecret    string `toml:"consumerSecret"`
	AccessToken       string `toml:"accessToken"`
	AccessTokenSecret string `toml:"accessTokenSecret"`
}
type Config struct {
	DatabasePath  string `toml:"databasePath"`
	Address       string `toml:"address"`
	RootURL       string `toml:"rootURL"`
	TwitterConfig `toml:"twitter"`
}

func getFirstNonEmpty(params ...string) string {
	for _, element := range params {
		if element != "" {
			return element
		}
	}
	return ""
}

func readToml(path string) Config {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return Config{}
	}

	var cfg Config
	if err := toml.Unmarshal(b, &cfg); err != nil {
		return Config{}
	}

	return cfg
}

func getHerokuAddress() string {
	var port = os.Getenv("PORT")

	if port != "" {
		port = fmt.Sprintf("0.0.0.0:%s", port)
	}

	return port
}

func getConf(path string) Config {
	var toml = readToml(path)

	return Config{
		DatabasePath: getFirstNonEmpty(os.Getenv("EMUARIUS_DATABASE_PATH"), toml.DatabasePath, "./emuarius.db"),
		Address:      getFirstNonEmpty(os.Getenv("EMUARIUS_ADDRESS"), toml.Address, getHerokuAddress(), ":4004"),
		RootURL:      getFirstNonEmpty(os.Getenv("EMUARIUS_ROOT_URL"), toml.RootURL, "http://localhost:4004"),
		TwitterConfig: TwitterConfig{
			ConsumerKey:       getFirstNonEmpty(os.Getenv("EMUARIUS_TWITTER_CONSUMER_KEY"), toml.TwitterConfig.ConsumerKey),
			ConsumerSecret:    getFirstNonEmpty(os.Getenv("EMUARIUS_TWITTER_CONSUMER_SECRET"), toml.TwitterConfig.ConsumerSecret),
			AccessToken:       getFirstNonEmpty(os.Getenv("EMUARIUS_TWITTER_ACCESS_TOKEN"), toml.TwitterConfig.AccessToken),
			AccessTokenSecret: getFirstNonEmpty(os.Getenv("EMUARIUS_TWITTER_ACCESS_TOKEN_SECRET"), toml.TwitterConfig.AccessTokenSecret),
		},
	}
}

func main() {
	var cfg = getConf("emuarius.toml")

	b, _ := toml.Marshal(cfg)
	log.Println("Configuration: \n" + fmt.Sprintf("%s", b))

	db, err := bolt.Open(cfg.DatabasePath, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	anaconda.SetConsumerKey(cfg.TwitterConfig.ConsumerKey)
	anaconda.SetConsumerSecret(cfg.TwitterConfig.ConsumerSecret)
	api := anaconda.NewTwitterApi(cfg.TwitterConfig.AccessToken, cfg.TwitterConfig.AccessTokenSecret)

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
