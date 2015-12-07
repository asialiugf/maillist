package maillist

// An account is equivalent to a user. All lists, messages, and subscribers must
// have an associated account
type Account struct {
	ID        int64  `db:"id"`
	FirstName string `db:"first_name" validate:"required"`
	LastName  string `db:"last_name" validate:"required"`
	Email     string `db:"email" validate:"required"`
	Status    string `db:"status" validate:"eq=active|eq=deleted"`
}

// InsertAccount adds the database to the account. The ID field will be
// updated
func (s *Session) InsertAccount(a *Account) error {
	if a.Status == "" {
		a.Status = "active"
	}
	return s.insert(a)
}

// GetAccount retrieves an account with a given ID
func (s *Session) GetAccount(userID int64) (*Account, error) {
	var a Account
	err := s.selectOne(&a, "id", userID)
	return &a, err
}

// UpdateAccount updates an account (identified by it's ID)
func (s *Session) UpdateAccount(a *Account) error {
	return s.update(a)
}

// DeleteAccount removes an account
func (s *Session) DeleteAccount(accountID int64) error {
	return s.delete(Account{}, accountID)
}