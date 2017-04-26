package emuarius

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/ChimeraCoder/anaconda"
	"github.com/emersion/go-ostatus"
	"github.com/emersion/go-ostatus/activitystream"
	"github.com/emersion/go-ostatus/salmon"
	"github.com/emersion/go-ostatus/xrd"
)

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

type subscription struct {
	stream   *anaconda.Stream
	notifies chan<- *activitystream.Feed
}

type Backend struct {
	api     *anaconda.TwitterApi
	rootURL string
	domain  string
	topics  map[string]*subscription
}

func NewBackend(api *anaconda.TwitterApi, rootURL string) *Backend {
	u, _ := url.Parse(rootURL)

	return &Backend{
		api:     api,
		rootURL: rootURL,
		domain:  u.Host,
		topics:  make(map[string]*subscription),
	}
}

func (be *Backend) accountURI(username string) string {
	return "acct:" + username + "@" + be.domain
}

func (be *Backend) Subscribe(topic string, notifies chan<- *activitystream.Feed) error {
	// TODO: parse topic
	/*u, err := url.Parse(topic)
	if err != nil {
		return err
	}*/

	v := make(url.Values)
	v.Set("follow", "emersion_fr")
	s := be.api.PublicStreamFilter(v)

	be.topics[topic] = &subscription{s, notifies}

	go func() {
		defer close(notifies)

		for update := range s.C {
			switch update.(type) {
			case *anaconda.EventTweet:
			case *anaconda.EventList:
			case *anaconda.Event:
			}
		}
	}()

	return nil
}

func (be *Backend) Unsubscribe(notifies chan<- *activitystream.Feed) error {
	for topic, sub := range be.topics {
		if notifies == sub.notifies {
			delete(be.topics, topic)
			sub.stream.Stop()
			return nil
		}
	}

	return nil
}

func (be *Backend) Notify(entry *activitystream.Entry) error {
	return errors.New("Not yet implemented") // TODO
}

func (be *Backend) Feed(topicURL string) (*activitystream.Feed, error) {
	username := uriToUsername(topicURL)

	u, err := be.api.GetUsersShow(username, make(url.Values))
	if err != nil {
		return nil, err
	}

	v := make(url.Values)
	v.Set("screen_name", u.ScreenName)
	v.Set("count", "20")
	v.Set("include_rts", "1")
	tweets, err := be.api.GetUserTimeline(v)
	if err != nil {
		return nil, err
	}

	feedURL := be.rootURL + feedPath(u.ScreenName)

	feed := &activitystream.Feed{
		ID:       feedURL,
		Title:    u.Name,
		Subtitle: u.Description,
		Logo:     u.ProfileImageURL,
		Updated:  activitystream.NewTime(time.Now()), // TODO
		Link: []activitystream.Link{
			{Rel: "alternate", Type: "text/html", Href: profileURL(u.ScreenName)},
			{Rel: "self", Type: "application/atom+xml", Href: feedURL},
			{Rel: "hub", Href: be.rootURL + ostatus.HubPath},
			// TODO: rel=next
			{Rel: ostatus.LinkSalmon, Href: be.rootURL + ostatus.SalmonPath},
		},
		Author: &activitystream.Person{
			ID:         be.accountURI(u.ScreenName),
			URI:        be.accountURI(u.ScreenName),
			Name:       u.Name,
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
		},
	}

	for _, tweet := range tweets {
		createdAt, _ := tweet.CreatedAtTime()

		// TODO: transform tweet to HTML
		feed.Entry = append(feed.Entry, &activitystream.Entry{
			ID:        "tag:" + be.domain + ",2017-04-23:tweet:" + tweet.IdStr,
			Title:     "Tweet",
			Published: activitystream.NewTime(createdAt),
			Updated:   activitystream.NewTime(createdAt),
			Link: []activitystream.Link{
				{Rel: "alternate", Type: "text/html", Href: profileURL(u.ScreenName)+"/status/"+tweet.IdStr},
				{Rel: "mentioned", ObjectType: activitystream.ObjectCollection, Href: activitystream.CollectionPublic},
			},
			Content: &activitystream.Text{
				Type: "text/html",
				Body: tweet.Text,
			},
			ObjectType: activitystream.ObjectNote,
			Verb:       activitystream.VerbPost,
		})
	}

	return feed, nil
}

func (be *Backend) Resource(uri string, rel []string) (*xrd.Resource, error) {
	username := uriToUsername(uri)
	u, err := be.api.GetUsersShow(username, make(url.Values))
	if err != nil {
		return nil, err
	}

	// TODO: retrieve public key
	publicKey, _ := salmon.ParsePublicKey("RSA.mVgY8RN6URBTstndvmUUPb4UZTdwvwmddSKE5z_jvKUEK6yk1u3rrC9yN8k6FilGj9K0eeUPe2hf4Pj-5CmHww.AQAB")

	publicKeyURL, err := salmon.PublicKeyDataURL(publicKey)
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
