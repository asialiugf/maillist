package maillist

import "fmt"

// A subscriber stores a single email address and some associated parameters.
// Each subscriber must have an associated account, and a given email address
// will have one subscriber for each account
type Subscriber struct {
	ID        int64  `db:"id"`
	AccountID int64  `db:"account_id" validate:"required"`
	FirstName string `db:"first_name" validate:"required"`
	LastName  string `db:"last_name" validate:"required"`
	Email     string `db:"email" validate:"required,email"`
	Status    string `db:"status" validate:"eq=active|eq=deleted"`
}

// GetSubscribers retrieves all the subscribers in a mailing list
func (s *Session) GetSubscribers(listID int64) ([]*Subscriber, error) {
	var subs []*Subscriber

	sql := fmt.Sprintf("select %s from subscriber inner join list_subscriber on subscriber.id = subscriber_id where list_id=?", s.selectString(&Subscriber{}))
	if _, err := s.dbmap.Select(&subs, sql, listID); err != nil {
		return nil, err
	}
	return subs, nil
}

// InsertSubscriber adds a subscriber to a mailing list
func (s *Session) InsertSubscriber(sub *Subscriber) error {
	if sub.Status == "" {
		sub.Status = "active"
	}
	return s.insert(sub)
}

// GetSubscriber retrieves a subscriber with a given ID
func (s *Session) GetSubscriber(subscriberID int64) (*Subscriber, error) {
	var sub Subscriber
	err := s.selectOne(&sub, "id", subscriberID)
	return &sub, err
}