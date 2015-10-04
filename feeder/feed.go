package feeder

import (
	"github.com/th3osmith/rss"
	"strings"
	"time"
)

const (
	StatusOK           = iota
	StatusNotFound     = iota
	StatusUnauthorized = iota
	StatusAccessDenied = iota
	StatusError        = iota
)

const SeenLength = 200

type Feed struct {
	Name        string
	Status      int
	Refresh     time.Time
	subscribers []Subscriber
	username    string
	password    string
	feed        *rss.Feed
	Url         string
	Seen        []string
}

type FetchFunc func() (*rss.Feed, error)

func NewFeed(url string) (*Feed, error) {

	fetchFunc := func() (*rss.Feed, error) {
		return rss.Fetch(url)
	}

	feed := new(Feed)
	feed.Url = url

	return CreateFeedWithFunc(feed, fetchFunc)
}

func NewAuthFeed(url string, username string, password string) (*Feed, error) {
	fetchFunc := func() (*rss.Feed, error) {
		return rss.FetchBasicAuth(url, username, password)
	}

	feed := new(Feed)
	feed.Url = url
	feed.username = username
	feed.password = password

	return CreateFeedWithFunc(feed, fetchFunc)
}

func NewFeedFromSeed(seed Seed) (feed *Feed, err error) {

	// We disable caching because the first parsing is going to be discarded
	caching := rss.CacheParsedItemIDs(false)

	if seed.Username != "" && seed.Password != "" {
		feed, err = NewAuthFeed(seed.Url, seed.Username, seed.Password)
	} else {
		feed, err = NewFeed(seed.Url)
	}

	if err != nil {
		return nil, err
	}

	feed.Clear()

	feed.Seen = seed.Seen
	feed.feed.ItemMap = make(map[string]struct{})
	for _, el := range seed.Seen {
		feed.feed.ItemMap[el] = struct{}{}
	}

	rss.CacheParsedItemIDs(caching)

	return feed, nil

}

func CreateFeedWithFunc(feedIn *Feed, fetchFunc FetchFunc) (feed *Feed, err error) {

	rawFeed, err := fetchFunc()
	if err != nil {
		return nil, err
	}

	feed = feedIn

	feed.Name = rawFeed.Title
	feed.Status = StatusOK
	feed.feed = rawFeed

	feed.Seen = make([]string, 0, SeenLength)

	return
}

func (feed *Feed) Register(subscriber Subscriber) {

	feed.subscribers = append(feed.subscribers, subscriber)

}

func (feed *Feed) Update(force bool) (err error) {

	if force {
		feed.feed.Refresh = time.Now()
	}

	unread := feed.feed.Unread

	err = feed.feed.Update()

	if err != nil {
		if strings.Contains(err.Error(), "Code 404") {
			feed.Status = StatusNotFound

		} else if strings.Contains(err.Error(), "Code 401") {
			feed.Status = StatusUnauthorized

		} else if strings.Contains(err.Error(), "Code 403") {
			feed.Status = StatusAccessDenied

		} else {
			feed.Status = StatusError
		}

		return err
	}

	if feed.feed.Unread > unread {
		feed.ReadNew()
	}

	return nil

}

func (feed *Feed) ReadNew() {

	ids := []string{}
	for _, item := range feed.feed.Items {
		for _, sub := range feed.subscribers {
			sub.AddItem(item)
		}
		// Register seen
		ids = append(ids, item.ID)
	}

	feed.Seen = AddNew(feed.Seen, ids...)
	feed.Clear()
}

func (feed *Feed) Clear() {

	feed.feed.Unread = 0

	// Remove elements and make them eligible to garbage collection
	for index := range feed.feed.Items {
		feed.feed.Items[index] = nil
	}
	feed.feed.Items = feed.feed.Items[:0]

}

type Seed struct {
	Url      string
	Seen     []string
	Username string
	Password string
}

// Add new element in the beginning and remove elements beyond the capacity
func AddNew(s []string, e ...string) []string {
	tmp := append([]string{}, e...)
	return append(tmp, s[0:cap(s)-len(tmp)]...)
}

func (feed *Feed) ExportSeed() Seed {
	return Seed{feed.Url, feed.Seen, feed.username, feed.password}
}
