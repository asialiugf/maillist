package maillist

import "fmt"

// Account is equivalent to a user. All lists, messages, and subscribers must
// have an associated account
type Account struct {
	ID        int64  `db:"id"`
	FirstName string `db:"first_name" validate:"required"`
	LastName  string `db:"last_name" validate:"required"`
	Email     string `db:"email" validate:"required"`
	Status    string `db:"status" validate:"eq=active|eq=deleted"`
}

// InsertAccount adds the database to the account. The ID field will be
// updated. It is an error to have duplicate email addresses for the account
// table
func (s *Session) InsertAccount(a *Account) error {
	if a.Status == "" {
		a.Status = "active"
	}
	return s.insert(a)
}

// UpsertAccount updates an account if the associated email address already
// exists. Otherwise it inserts a new account
func (s *Session) UpsertAccount(a *Account) error {
	id, err := s.dbmap.SelectInt("select id from account where email=?", a.Email)
	if err != nil {
		return err
	}
	if id == 0 {
		return s.InsertAccount(a)
	}
	a.ID = id
	return s.UpdateAccount(a)
}

// GetAccount retrieves an account with a given ID
func (s *Session) GetAccount(accountID int64) (*Account, error) {
	var a Account
	sql := fmt.Sprintf("select %s from account where status!='deleted' and id=?", s.selectString(&a))
	err := s.dbmap.SelectOne(&a, sql, accountID)
	return &a, err
}

// UpdateAccount updates an account (identified by it's ID)
func (s *Session) UpdateAccount(a *Account) error {
	if a.Status == "" {
		a.Status = "active"
	}
	return s.update(a)
}

// DeleteAccount removes an account
func (s *Session) DeleteAccount(accountID int64) error {
	return s.delete(Account{}, accountID)
}
