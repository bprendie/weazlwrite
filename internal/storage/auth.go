package storage

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/sha3"
)

var passwordAttemptDelays = []time.Duration{
	5 * time.Second,
	10 * time.Second,
	30 * time.Second,
	time.Minute,
	5 * time.Minute,
	15 * time.Minute,
}

func (s *Store) HasVault() (bool, error) {
	var count int
	if err := s.db.QueryRow(`select count(*) from vault where id = 1`).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) CreateVault(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(`insert into vault (id, password_hash) values (1, ?)`, string(hash)); err != nil {
		return err
	}
	s.resetPasswordAttempts()
	s.unlockWith(password)
	return nil
}

func (s *Store) Unlock(password string) error {
	if time.Now().Before(s.lockoutUntil) {
		remaining := time.Until(s.lockoutUntil).Round(time.Second)
		return fmt.Errorf("too many failed attempts, try again in %v", remaining)
	}
	var hash string
	if err := s.db.QueryRow(`select password_hash from vault where id = 1`).Scan(&hash); err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		s.recordFailedPasswordAttempt()
		return errors.New("bad vault password")
	}
	s.resetPasswordAttempts()
	s.unlockWith(password)
	return nil
}

func (s *Store) unlockWith(password string) {
	sum := sha3.Sum256([]byte(password))
	s.key = sum[:]
	s.unlocked = true
	s.UpdateActivity()
}

func (s *Store) recordFailedPasswordAttempt() {
	s.failedAttempts++
	delay := passwordAttemptDelays[len(passwordAttemptDelays)-1]
	if s.failedAttempts <= len(passwordAttemptDelays) {
		delay = passwordAttemptDelays[s.failedAttempts-1]
	}
	s.lockoutUntil = time.Now().Add(delay)
}

func (s *Store) resetPasswordAttempts() {
	s.failedAttempts = 0
	s.lockoutUntil = time.Time{}
}
