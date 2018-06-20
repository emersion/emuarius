# emuarius

Bridge between Twitter and Mastodon (or any OStatus-compliant instance). Powered by [go-ostatus](https://github.com/emersion/go-ostatus).

For the moment, this is a read-only bridge: you'll be able to see Twitter
activity from Mastodon (try to follow to follow _twitter_username@rootURL_), but you won't be able to interact with it.

## Usage

Your server must be configured with a domain name and HTTPS.

```shell
go get -u github.com/emersion/emuarius/cmd/...
cp emuarius.example.toml emuarius.toml
# Fill emuarius.toml with Twitter app credentials
emuarius
```

### Docker
```shell
docker build -t emuarius .
docker run -e "EMUARIUS_TWITTER_CONSUMER_KEY=xxx" \
-e "EMUARIUS_TWITTER_CONSUMER_SECRET=xxx" \
-e "EMUARIUS_TWITTER_ACCESS_TOKEN=xxx" \
-e "EMUARIUS_TWITTER_ACCESS_TOKEN_SECRET=xxx" emuarius
```

### Heroku
You need the heroku [cli tool](https://devcenter.heroku.com/articles/heroku-cli).
```shell
heroku container:login
heroku create
heroku container:push web
heroku container:release web
heroku ps:scale web=1
heroku open # The opened url is the EMUARIUS_ROOT_URL
```
Configure environment variables with Twitter app credentials inside the [dashboard](https://dashboard.heroku.com/). You dont need to set _EMUARIUS_ADDRESS_ or _EMUARIUS_ROOT_URL_



## Configuration

You can configure emuarius with a toml configuration file or environment variables. Environment variables will take precedence over toml values.


| toml                      | env                                 | default               |
| ------------------------- | ----------------------------------- | --------------------- |
| address                   | EMUARIUS_ADDRESS                     | :4004                 |
| rootURL                   | EMUARIUS_ROOT_URL                    | http://localhost:4004 |
| databasePath              | EMUARIUS_DATABASE_PATH               | ./emuarius.db         |
| twitter.consumerKey       | EMUARIUS_TWITTER_CONSUMER_KEY        |                       |
| twitter.consumerSecret    | EMUARIUS_TWITTER_CONSUMER_SECRET     |                       |
| twitter.accessToken       | EMUARIUS_TWITTER_ACCESS_TOKEN        |                       |
| twitter.accessTokenSecret | EMUARIUS_TWITTER_ACCESS_TOKEN_SECRET |                       |


## License

MIT
