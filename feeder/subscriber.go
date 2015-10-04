package feeder

import (
	"github.com/th3osmith/rss"
)

type Subscriber interface {
	AddItem(item *rss.Item) error
}

type TestSubscriber struct {
	Items []*rss.Item
}

func (s *TestSubscriber) AddItem(item *rss.Item) (err error) {

	s.Items = append(s.Items, item)
	return

}
