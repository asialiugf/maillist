package maillist_test

import (
	"bytes"
	"log"
	"os"
	"testing"
	"time"

	"github.com/Attendly/maillist"
	_ "github.com/go-sql-driver/mysql"
)

// Example session of sending a single test email. DatabaseAddress,
// SendGridAPIKey would have to be set appropriately. JustPrint should be
// false, and Subscriber.Email changed to send a real message.
func Example() {
	var err error
	var s *maillist.Session
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	config := maillist.Config{
		DatabaseAddress: os.Getenv("ATTENDLY_EMAIL_DATABASE"),
		JustPrint:       true,
		Logger:          os.Stdout,
		UnsubscribeURL:  "https://myeventarc.localhost/unsubscribe",

		SendGridUsername: os.Getenv("ATTENDLY_EMAIL_USERNAME"),
		SendGridPassword: os.Getenv("ATTENDLY_EMAIL_PASSWORD"),
		SendGridAPIKey:   os.Getenv("ATTENDLY_EMAIL_APIKEY"),
	}

	if s, err = maillist.OpenSession(&config); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	a := maillist.Account{
		FirstName: "Joe",
		LastName:  "Bloggs",
		Email:     "sendgrid@eventarc.com",
	}
	if err := s.UpsertAccount(&a); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	l := maillist.List{
		AccountID: a.ID,
		Name:      "My Awesome Mailing List",
	}
	if err = s.InsertList(&l); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	sub := maillist.Subscriber{
		AccountID: a.ID,
		FirstName: "Tommy",
		LastName:  "Barker",
		Email:     "tom@attendly.com",
	}
	if err = s.GetOrInsertSubscriber(&sub); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	if err = s.AddSubscriberToList(l.ID, sub.ID); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	c := maillist.Campaign{
		AccountID: a.ID,
		Subject:   "Awesome Event 2016",
		Body:      "Hi {{.FirstName}} {{.LastName}},\nThis is a test of attendly email list service",
		Scheduled: time.Now(),
	}
	if err = s.InsertCampaign(&c, []int64{l.ID}, nil); err != nil {
		log.Fatalf("error: %v\n", err)
	}
	time.Sleep(5 * time.Second)
	if err := s.Close(); err != nil {
		log.Fatalf("could not close session: %v", err)
	}

	// Output:
	// Email to send
	// To: tom@attendly.com (Tommy Barker)
	// From: sendgrid@eventarc.com (Joe Bloggs)
	// Subject: Awesome Event 2016
	// Body: Hi Tommy Barker,
	// This is a test of attendly email list service
}

func TestGetAttendeesCallback(t *testing.T) {
	var err error
	var s *maillist.Session
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var accountID int64

	getAttendees := func(eventID int64) []*maillist.Subscriber {
		return []*maillist.Subscriber{{
			AccountID: accountID,
			FirstName: "Freddy",
			LastName:  "Example",
			Email:     "fred@example.com",
		}}
	}

	var buf bytes.Buffer

	config := maillist.Config{
		DatabaseAddress:      os.Getenv("ATTENDLY_EMAIL_DATABASE"),
		GetAttendeesCallback: getAttendees,
		JustPrint:            true,
		Logger:               &buf,
		UnsubscribeURL:       "https://myeventarc.localhost/unsubscribe",

		SendGridUsername: os.Getenv("ATTENDLY_EMAIL_USERNAME"),
		SendGridPassword: os.Getenv("ATTENDLY_EMAIL_PASSWORD"),
		SendGridAPIKey:   os.Getenv("ATTENDLY_EMAIL_APIKEY"),
	}

	if s, err = maillist.OpenSession(&config); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	a := maillist.Account{
		FirstName: "Spamface",
		LastName:  "The Bold",
		Email:     "example@example.com",
	}
	if err := s.UpsertAccount(&a); err != nil {
		log.Fatalf("error: %v\n", err)
	}
	accountID = a.ID

	l := maillist.List{
		AccountID: a.ID,
		Name:      "My Awesome Mailing List",
	}
	if err = s.InsertList(&l); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	c := maillist.Campaign{
		AccountID: a.ID,
		Subject:   "Awesome Event 2016",
		Body:      "Hi {{.FirstName}} {{.LastName}},\nThis is a test of attendly email list service",
		Scheduled: time.Now(),
	}
	if err = s.InsertCampaign(&c, nil, []int64{5}); err != nil {
		log.Fatalf("error: %v\n", err)
	}
	time.Sleep(5 * time.Second)
	if err := s.Close(); err != nil {
		log.Fatalf("could not close session: %v", err)
	}

	out := buf.String()
	want := `Email to send
To: fred@example.com (Freddy Example)
From: example@example.com (Spamface The Bold)
Subject: Awesome Event 2016
Body: Hi Freddy Example,
This is a test of attendly email list service
`
	if out != want {
		log.Fatalf("got: '%s'\n\nwant: '%s'\n\n", out, want)
	}
}

func TestGetSpamReports(t *testing.T) {
	config := maillist.Config{
		DatabaseAddress: os.Getenv("ATTENDLY_EMAIL_DATABASE"),
		UnsubscribeURL:  "https://myeventarc.localhost/unsubscribe",
		JustPrint:       true,

		SendGridUsername: os.Getenv("ATTENDLY_EMAIL_USERNAME"),
		SendGridPassword: os.Getenv("ATTENDLY_EMAIL_PASSWORD"),
		SendGridAPIKey:   os.Getenv("ATTENDLY_EMAIL_APIKEY"),
	}

	s, err := maillist.OpenSession(&config)
	if err != nil {
		t.Fatalf("%v", err)
	}

	if b, err := s.HasReportedSpam("example@example.com"); err != nil {
		t.Fatalf("error: %v\n", err)
	} else if b {
		t.Fatalf("Example incorrectly has reported spam\n")
	}

	if b, err := s.HasReportedSpam("jorgen@hotmail.com"); err != nil {
		t.Fatalf("error: %v\n", err)
	} else if !b {
		t.Fatalf("Example incorrectly has not reported spam\n")
	}
}

func TestUnsubscribeToken(t *testing.T) {
	var (
		err       error
		token     string
		sub, sub2 *maillist.Subscriber
		s         *maillist.Session
	)

	config := maillist.Config{
		DatabaseAddress: os.Getenv("ATTENDLY_EMAIL_DATABASE"),
		UnsubscribeURL:  "https://myeventarc.localhost/unsubscribe",
		JustPrint:       true,

		SendGridUsername: os.Getenv("ATTENDLY_EMAIL_USERNAME"),
		SendGridPassword: os.Getenv("ATTENDLY_EMAIL_PASSWORD"),
		SendGridAPIKey:   os.Getenv("ATTENDLY_EMAIL_APIKEY"),
	}

	if s, err = maillist.OpenSession(&config); err != nil {
		t.Fatalf("%v", err)
	}

	sub = &maillist.Subscriber{
		FirstName: "Johnny",
		LastName:  "Knoxville",
		Email:     "johnny.k@example.com",
	}

	if err = s.GetOrInsertSubscriber(sub); err != nil {
		t.Fatalf("error: %v", err)
	}

	if token, err = s.UnsubscribeToken(sub); err != nil {
		t.Fatalf("error: %v", err)
	}

	if sub2, err = s.GetSubscriberByToken(token); err != nil {
		t.Fatalf("error:%v", err)
	}
	if sub.ID != sub2.ID {
		t.Fatalf("GetSubscriberByToken result incorrect\n")
	}
}

func TestGetLists(t *testing.T) {
	config := maillist.Config{
		DatabaseAddress: os.Getenv("ATTENDLY_EMAIL_DATABASE"),
		UnsubscribeURL:  "https://myeventarc.localhost/unsubscribe",
		JustPrint:       true,

		SendGridUsername: os.Getenv("ATTENDLY_EMAIL_USERNAME"),
		SendGridPassword: os.Getenv("ATTENDLY_EMAIL_PASSWORD"),
		SendGridAPIKey:   os.Getenv("ATTENDLY_EMAIL_APIKEY"),
	}

	s, err := maillist.OpenSession(&config)
	if err != nil {
		t.Fatalf("%v", err)
	}

	a := maillist.Account{
		FirstName: "Brian",
		LastName:  "Cohen",
		Email:     "briancohen@example.com",
	}
	if err := s.UpsertAccount(&a); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	l1 := maillist.List{
		AccountID: a.ID,
		Name:      "TestGetLists 1",
	}
	if err = s.InsertList(&l1); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	l2 := maillist.List{
		AccountID: a.ID,
		Name:      "TestGetLists 2",
	}
	if err = s.InsertList(&l2); err != nil {
		log.Fatalf("error: %v\n", err)
	}

	lists, err := s.GetLists(a.ID)
	if err != nil {
		log.Fatalf("Could not GetLists: %v", err)
	}

	if len(lists) != 2 {
		log.Fatalf("Error in GetLists: length is %d, want %d\n", len(lists), 2)
	}

	if (lists[0].ID != l1.ID || lists[1].ID != l2.ID) &&
		(lists[0].ID != l2.ID || lists[1].ID != l1.ID) {
		log.Fatalf("error in GetLists: didn't get list\n")
	}

	if err := s.DeleteList(l1.ID); err != nil {
		log.Fatalf("Could not delete mailing lists: %v", err)
	}

	if err := s.DeleteList(l2.ID); err != nil {
		log.Fatalf("Could not delete mailing lists: %v", err)
	}
}
