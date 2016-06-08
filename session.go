package maillist

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"time"

	"github.com/sendgrid/sendgrid-go"
)

type getAttendeeFunc func(eventID int64) []*Subscriber

// Session is an opaque type holding database connections and other
// implementation details
type Session struct {
	database
	config    Config
	wake      chan bool
	templates map[int64]*template.Template
	sgClient  *sendgrid.SGClient
}

// Config stores application defined options
type Config struct {
	DatabaseAddress      string
	JustPrint            bool
	Logger               io.Writer
	GetAttendeesCallback getAttendeeFunc
	UnsubscribeURL       string

	SendGridAPIKey   string
	SendGridUsername string
	SendGridPassword string
}

// OpenSession initialises a connection with the mailing list system. A call to
// Session.Close() should follow to ensure a clean exit.
func OpenSession(config *Config) (*Session, error) {
	var s Session
	var err error

	s.database, err = openDatabase(config.DatabaseAddress)
	if err != nil {
		return nil, err
	}

	s.config = *config

	if !config.JustPrint {
		if config.SendGridAPIKey == "" {
			return nil, errors.New("maillist: SendGridAPIKey must be set")
		}
		if config.SendGridUsername == "" {
			return nil, errors.New("maillist: SendGridUsername must be set")
		}
		if config.SendGridPassword == "" {
			return nil, errors.New("maillist: SendGridPassword must be set")
		}
	}

	if s.config.Logger == nil {
		s.config.Logger = os.Stderr
	}

	if s.config.GetAttendeesCallback == nil {
		fmt.Fprintln(s.config.Logger, "maillist: GetAttendeesCallback not set -- sending to events disabled")
		s.config.GetAttendeesCallback = func(eventID int64) []*Subscriber {
			fmt.Fprintln(s.config.Logger, "maillist: GetAttendeesCallback not set -- sending to events disabled")
			return nil
		}
	}

	s.addTable(Account{}, "account")
	s.addTable(List{}, "list")
	s.addTable(Campaign{}, "campaign")
	s.addTable(Subscriber{}, "subscriber")
	s.addTable(Message{}, "message")
	s.addTable(ListSubscriber{}, "list_subscriber")

	// err = s.dbmap.CreateTablesIfNotExists()

	s.templates = make(map[int64]*template.Template)
	s.sgClient = sendgrid.NewSendGridClientWithApiKey(s.config.SendGridAPIKey)

	s.wake = make(chan bool)
	go service(&s)
	s.wake <- true

	return &s, err
}

// Close closes the session. It blocks until the session is cleanly exited
func (s *Session) Close() error {
	close(s.wake)
	return s.db.Close()
}

// listens for commands from the API. This is intended to be run asynchronously
// and mainly exists to prevent the API from blocking.
func service(s *Session) {
	ticker := time.NewTicker(time.Minute)

next:
	select {
	case _, ok := <-s.wake:
		if !ok {
			ticker.Stop()
			return
		}
	case <-ticker.C:
	}

	for {
		c, err := getDueCampaign(s)
		if err == ErrNotFound {
			break

		} else if err != nil {
			s.logf("couldn't retrieve due campaign: %v\n", err)
			break
		}

		if err = s.sendCampaign(c.ID); err != nil {
			s.logf("couldn't send campaign: %v\n", err)
			break
		}
	}

	for {
		m, err := pendingMessage(s)
		if err == ErrNotFound {
			break

		} else if err != nil {
			s.logf("couldn't retrieve pending message: %v\n", err)
			break
		}

		if err = s.sendMessage(m); err != nil {
			s.logf("couldn't send message: %v\n", err)
			break
		}
		time.Sleep(time.Second)
	}
	goto next
}

func (s *Session) logf(format string, args ...interface{}) {
	fmt.Fprintf(s.config.Logger, format, args)
}
