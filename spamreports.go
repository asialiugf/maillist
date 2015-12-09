package maillist

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"
)

var (
	spamReportsUpdated time.Time
	convTimeToISO      *regexp.Regexp
	spamReports        map[string]bool
)

func init() {
	convTimeToISO = regexp.MustCompile(`"(\d\d\d\d-\d\d-\d\d) (\d\d:\d\d:\d\d)"`)
}

func initSpamReports(s *Session) error {
	if s.config.SendGridUsername == "" {
		return errors.New("SendGrid username not set")
	}
	if s.config.SendGridPassword == "" {
		return errors.New("SendGrid password not set")
	}
	url := fmt.Sprintf(
		`https://api.sendgrid.com/api/spamreports.get.json?api_user=%s&api_key=%s&date=1`,
		s.config.SendGridUsername, s.config.SendGridPassword)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("http error %s: %s", resp.Status, resp.Body)
	}

	buf, err := ioutil.ReadAll(resp.Body)
	buf = convTimeToISO.ReplaceAll(buf, []byte(`"${1}T${2}Z"`))

	if err != nil {
		return err
	}

	var spamReportsList []struct {
		Email string `json:"email"`
	}

	if err := json.Unmarshal(buf, &spamReportsList); err != nil {
		return err
	}

	spamReports = make(map[string]bool)
	for _, report := range spamReportsList {
		spamReports[report.Email] = true
	}

	return nil
}

func (s *Session) HasReportedSpam(email string) (bool, error) {
	if spamReports == nil || time.Now().Sub(spamReportsUpdated) > 6*time.Hour {
		if err := initSpamReports(s); err != nil {
			return false, err
		}
	}
	return spamReports[email], nil
}
