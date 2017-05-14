package emuarius

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/ChimeraCoder/anaconda"
	"github.com/boltdb/bolt"
	"github.com/emersion/go-ostatus"
	"github.com/emersion/go-ostatus/activitystream"
	"github.com/emersion/go-ostatus/salmon"
	"github.com/emersion/go-ostatus/xrd"

	"log"
)

const keyBits = 2048

var keysBucket = []byte("RSAKeys")

func uriToUsername(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return ""
	}

	// TODO: check u.Host
	switch u.Scheme {
	case "acct":
		return strings.SplitN(u.Opaque, "@", 2)[0]
	case "http", "https", "":
		return strings.TrimSuffix(strings.Trim(u.Path, "/@"), ".atom")
	}
	return ""
}

func feedPath(username string) string {
	return "/@" + username + ".atom"
}

func profileURL(username string) string {
	return "https://twitter.com/" + username
}

func tweetURL(username, id string) string {
	return profileURL(username) + "/status/" + id
}

func hashtagURL(hastag string) string {
	return "https://twitter.com/hashtag/" + hastag
}

func itob(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

type entityURL struct {
	Indices      []int
	Url          string
	Display_url  string
	Expanded_url string
}

func formatTweet(tweet *anaconda.Tweet) string {
	urls := tweet.Entities.Urls

	for _, mention := range tweet.Entities.User_mentions {
		urls = append(urls, entityURL{
			Indices: mention.Indices,
			Url:     profileURL(mention.Screen_name),
		})
	}

	for _, hashtag := range tweet.Entities.Hashtags {
		urls = append(urls, entityURL{
			Indices: hashtag.Indices,
			Url:     hashtagURL(hashtag.Text),
		})
	}

	for _, media := range tweet.Entities.Media {
		urls = append(urls, entityURL{
			Indices: media.Indices,
			Url:     media.Media_url,
		})
	}

	sort.Slice(urls, func(i, j int) bool {
		return urls[i].Indices[0] < urls[j].Indices[0]
	})

	formatted := []rune(tweet.Text)
	delta := 0
	for _, u := range urls {
		before := formatted[:u.Indices[0]+delta]
		between := formatted[u.Indices[0]+delta : u.Indices[1]+delta]
		after := formatted[u.Indices[1]+delta:]

		insertBefore := `<a href="` + u.Url + `">`
		insertAfter := `</a>`
		delta += len(insertBefore) + len(insertAfter)

		rest := insertBefore + string(between) + insertAfter + string(after)

		formatted = append(before, []rune(rest)...)
	}

	return strings.Replace(string(formatted), "\n", "<br>", -1)
}

type subscription struct {
	ticker   *time.Ticker
	notifies chan<- *activitystream.Feed
}

type Backend struct {
	api     *anaconda.TwitterApi
	db      *bolt.DB
	rootURL string
	domain  string
	topics  map[string]*subscription
}

func NewBackend(api *anaconda.TwitterApi, db *bolt.DB, rootURL string) *Backend {
	u, _ := url.Parse(rootURL)

	return &Backend{
		api:     api,
		db:      db,
		rootURL: rootURL,
		domain:  u.Host,
		topics:  make(map[string]*subscription),
	}
}

func (be *Backend) accountURI(username string) string {
	return "acct:" + username + "@" + be.domain
}

func (be *Backend) tweetURI(id string) string {
	return "tag:" + be.domain + ",2017-04-23:tweet:" + id
}

func (be *Backend) newFeed(u *anaconda.User) *activitystream.Feed {
	feedURL := be.rootURL + feedPath(u.ScreenName)

	updated := time.Now()
	if u.Status != nil {
		updated, _ = u.Status.CreatedAtTime()
	}

	return &activitystream.Feed{
		ID:       feedURL,
		Title:    u.Name,
		Subtitle: u.Description,
		Logo:     u.ProfileImageURL,
		Updated:  activitystream.NewTime(updated),
		Link: []activitystream.Link{
			{Rel: "alternate", Type: "text/html", Href: profileURL(u.ScreenName)},
			{Rel: "self", Type: "application/atom+xml", Href: feedURL},
			{Rel: "hub", Href: be.rootURL + ostatus.HubPath},
			// TODO: rel=next
			{Rel: ostatus.LinkSalmon, Href: be.rootURL + ostatus.SalmonPath},
		},
		Author: be.newPerson(u),
	}
}

func (be *Backend) newPerson(u *anaconda.User) *activitystream.Person {
	return &activitystream.Person{
		ID:         be.accountURI(u.ScreenName),
		URI:        be.accountURI(u.ScreenName),
		Name:       u.Name,
		Email:      u.ScreenName + "@" + be.domain,
		Summary:    u.Description,
		ObjectType: activitystream.ObjectPerson,
		Link: []activitystream.Link{
			{Rel: "alternate", Type: "text/html", Href: profileURL(u.ScreenName)},
			{Rel: "avatar", Href: u.ProfileImageURL},
			{Rel: "header", Href: u.ProfileBannerURL},
		},
		PreferredUsername: u.ScreenName,
		DisplayName:       u.Name,
		Note:              u.Description,
	}
}

func (be *Backend) newEntryFromTweet(u *anaconda.User, tweet *anaconda.Tweet) *activitystream.Entry {
	createdAt, _ := tweet.CreatedAtTime()

	entry := &activitystream.Entry{
		ID:        be.tweetURI(tweet.IdStr),
		Title:     "Tweet",
		Published: activitystream.NewTime(createdAt),
		Updated:   activitystream.NewTime(createdAt),
		Link: []activitystream.Link{
			{Rel: "alternate", Type: "text/html", Href: tweetURL(u.ScreenName, tweet.IdStr)},
			{Rel: "mentioned", ObjectType: activitystream.ObjectCollection, Href: activitystream.CollectionPublic},
		},
		Content: &activitystream.Text{
			Type: "html",
			Lang: tweet.Lang,
			Body: formatTweet(tweet),
		},
	}

	if u.Id != tweet.User.Id {
		entry.Author = be.newPerson(&tweet.User)
	}

	if tweet.RetweetedStatus != nil {
		entry.Title = "Retweet"
		entry.ObjectType = activitystream.ObjectActivity
		entry.Verb = activitystream.VerbShare
		entry.Object = be.newEntryFromTweet(u, tweet.RetweetedStatus)
	} else if tweet.InReplyToStatusID != 0 {
		entry.Title = "Reply"
		entry.ObjectType = activitystream.ObjectComment
		entry.Verb = activitystream.VerbPost

		entry.InReplyTo = &activitystream.InReplyTo{
			Ref:  be.tweetURI(tweet.InReplyToStatusIdStr),
			Href: tweetURL(u.ScreenName, tweet.IdStr),
			Type: "text/html",
		}

		for _, mention := range tweet.Entities.User_mentions {
			entry.Link = append(entry.Link, activitystream.Link{
				Rel:        "mentioned",
				ObjectType: activitystream.ObjectPerson,
				Href:       be.accountURI(mention.Screen_name),
			})
		}
	} else {
		entry.Title = "Tweet"
		entry.ObjectType = activitystream.ObjectNote
		entry.Verb = activitystream.VerbPost
	}

	return entry
}

func (be *Backend) Subscribe(topicURL string, notifies chan<- *activitystream.Feed) error {
	username := uriToUsername(topicURL)
	if username == "" {
		return errors.New("Invalid topic")
	}

	u, err := be.api.GetUsersShow(username, make(url.Values))
	if err != nil {
		return err
	}

	var lastId int64
	var lastIdStr string
	if u.Status != nil {
		lastId = u.Status.Id
		lastIdStr = u.Status.IdStr
	}

	ticker := time.NewTicker(5 * time.Minute)
	be.topics[topicURL] = &subscription{ticker, notifies}

	go func() {
		defer close(notifies)

		for range ticker.C {
			v := make(url.Values)
			v.Set("user_id", u.IdStr)
			v.Set("include_rts", "1")
			v.Set("since_id", lastIdStr)
			v.Set("count", "200")
			tweets, err := be.api.GetUserTimeline(v)
			if err != nil {
				log.Println("emuarius: cannot poll user:", err)
				continue
			}

			if len(tweets) == 0 {
				continue
			}

			entries := make([]*activitystream.Entry, 0, len(tweets))
			for _, tweet := range tweets {
				entries = append(entries, be.newEntryFromTweet(&u, &tweet))

				if tweet.Id > lastId {
					lastId = tweet.Id
					lastIdStr = tweet.IdStr
					u.Status = &tweet
				}
			}

			feed := be.newFeed(&u)
			feed.Entry = entries
			notifies <- feed
		}
	}()

	return nil
}

func (be *Backend) Unsubscribe(notifies chan<- *activitystream.Feed) error {
	for topic, sub := range be.topics {
		if notifies == sub.notifies {
			delete(be.topics, topic)
			sub.ticker.Stop()
			return nil
		}
	}

	return nil
}

func (be *Backend) Notify(entry *activitystream.Entry) error {
	if entry.ObjectType != activitystream.ObjectActivity {
		return errors.New("Unsupported object type")
	}

	switch entry.Verb {
	case activitystream.VerbFollow, activitystream.VerbUnfollow:
		return nil // Nothing to do
	default:
		return errors.New("Unsupported verb")
	}
}

func (be *Backend) Feed(topicURL string) (*activitystream.Feed, error) {
	username := uriToUsername(topicURL)
	if username == "" {
		return nil, errors.New("Invalid topic")
	}

	u, err := be.api.GetUsersShow(username, make(url.Values))
	if err != nil {
		return nil, err
	}

	v := make(url.Values)
	v.Set("user_id", u.IdStr)
	v.Set("count", "20")
	v.Set("include_rts", "1")
	tweets, err := be.api.GetUserTimeline(v)
	if err != nil {
		return nil, err
	}

	feed := be.newFeed(&u)

	for _, tweet := range tweets {
		feed.Entry = append(feed.Entry, be.newEntryFromTweet(&u, &tweet))
	}

	return feed, nil
}

func (be *Backend) Resource(uri string, rel []string) (*xrd.Resource, error) {
	username := uriToUsername(uri)
	u, err := be.api.GetUsersShow(username, make(url.Values))
	if err != nil {
		return nil, err
	}

	var pub crypto.PublicKey
	err = be.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(keysBucket)
		if err != nil {
			return err
		}

		k := itob(u.Id)
		v := b.Get(k)
		var priv *rsa.PrivateKey
		if v == nil {
			priv, err = rsa.GenerateKey(rand.Reader, keyBits)
			if err != nil {
				return err
			}

			v = x509.MarshalPKCS1PrivateKey(priv)
			if err := b.Put(k, v); err != nil {
				return err
			}
		} else {
			priv, err = x509.ParsePKCS1PrivateKey(v)
			if err != nil {
				return err
			}
		}

		pub = priv.Public()
		return nil
	})
	if err != nil {
		return nil, err
	}

	publicKeyURL, err := salmon.PublicKeyDataURL(pub)
	if err != nil {
		return nil, err
	}

	accountURI := be.accountURI(u.ScreenName)
	profileURL := profileURL(u.ScreenName)
	resource := &xrd.Resource{
		Subject: accountURI,
		Aliases: []string{profileURL},
		Links: []*xrd.Link{
			{Rel: ostatus.LinkProfilePage, Type: "text/html", Href: profileURL},
			{Rel: ostatus.LinkUpdatesFrom, Type: "application/atom+xml", Href: be.rootURL + feedPath(u.ScreenName)},
			{Rel: ostatus.LinkSalmon, Href: be.rootURL + ostatus.SalmonPath},
			{Rel: ostatus.LinkMagicPublicKey, Href: publicKeyURL},
		},
	}
	return resource, nil
}
