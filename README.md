# emuarius

Bridge between Twitter and Mastodon (or any OStatus-compliant instance). Powered by [go-ostatus](https://github.com/emersion/go-ostatus).

For the moment, this is a read-only bridge: you'll be able to see Twitter
activity from Mastodon, but you won't be able to interact with it.

## Usage

Your server must be configured with a domain name and HTTPS.

```shell
go get -u github.com/emersion/emuarius/cmd/...
cp emuarius.example.toml emuarius.toml
# Fill emuarius.toml with Twitter app credentials
emuarius
```

You should now be able to follow _twitter_username@domain_name_.

## License

MIT
