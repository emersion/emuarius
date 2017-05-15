package emuarius

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/emersion/go-ostatus/pubsubhubbub"
)

var subscriptionsBucket = []byte("subscriptions")

type subscriptionData struct {
	Secret string `json:"secret"`
	LeaseEnd time.Time `json:"lease_end"`
}

func subscriptionToKey(topicURL, callbackURL string) []byte {
	return []byte(topicURL + " " + callbackURL)
}

func keyToSubscription(k []byte) (topicURL, callbackURL string) {
	parts := strings.SplitN(string(k), " ", 2)
	if len(parts) == 2 {
		topicURL, callbackURL = parts[0], parts[1]
	}
	return
}

func NewSubscriptionDB(p *pubsubhubbub.Publisher, db *bolt.DB) error {
	// Restore old subscriptions
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(subscriptionsBucket)
		if b == nil {
			return nil
		}

		return b.ForEach(func(k, v []byte) error {
			topicURL, callbackURL := keyToSubscription(k)

			s := new(subscriptionData)
			if err := json.Unmarshal(v, s); err != nil {
				return err
			}
			if s.LeaseEnd.Before(time.Now()) {
				return b.Delete(k)
			}

			return p.Register(topicURL, callbackURL, s.Secret, s.LeaseEnd)
		})
	})
	if err != nil {
		return err
	}

	// Save new subscriptions
	p.SubscriptionState = func(topicURL, callbackURL, secret string, leaseEnd time.Time) {
		err := db.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists(subscriptionsBucket)
			if err != nil {
				return err
			}

			k := subscriptionToKey(topicURL, callbackURL)
			if !leaseEnd.IsZero() {
				s := &subscriptionData{Secret: secret, LeaseEnd: leaseEnd}
				v, err := json.Marshal(s)
				if err != nil {
					return err
				}

				if err := b.Put(k, v); err != nil {
					return err
				}
			} else {
				if err := b.Delete(k); err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			log.Println("emuarius: cannot save subscription:", err)
		}
	}

	return nil
}
