# emuarius

Bridge between Twitter and Mastodon (or any OStatus-compliant instance). Powered by [go-ostatus](https://github.com/emersion/go-ostatus).

For the moment, this is a read-only bridge: you'll be able to see Twitter
activity from Mastodon (try to follow _twitter_username@rootURL_), but you won't be able to interact with it.

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
docker run -p 4004:4004 \
-e "EMUARIUS_TWITTER_CONSUMER_SECRET=xxx" \
-e "EMUARIUS_TWITTER_ACCESS_TOKEN=xxx" \
-e "EMUARIUS_TWITTER_ACCESS_TOKEN_SECRET=xxx" \
-e "EMUARIUS_TWITTER_CONSUMER_KEY=xxx"  emuarius
```

## Configuration

You can configure emuarius with a toml configuration file or environment variables. Environment variables will take precedence over toml values.

| toml                      | env                                  | default                |
| ------------------------- | ------------------------------------ | ---------------------- |
|                           | PORT                                 |                        |
| address                   | EMUARIUS_ADDRESS                     | :4004 or 0.0.0.0:$PORT |
| rootURL                   | EMUARIUS_ROOT_URL                    | http://localhost:4004  |
| databasePath              | EMUARIUS_DATABASE_PATH               | ./emuarius.db          |
| twitter.consumerKey       | EMUARIUS_TWITTER_CONSUMER_KEY        |                        |
| twitter.consumerSecret    | EMUARIUS_TWITTER_CONSUMER_SECRET     |                        |
| twitter.accessToken       | EMUARIUS_TWITTER_ACCESS_TOKEN        |                        |
| twitter.accessTokenSecret | EMUARIUS_TWITTER_ACCESS_TOKEN_SECRET |                        |

## License

MIT
