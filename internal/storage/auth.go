package storage

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/sha3"
)

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
	s.unlockWith(password)
	return nil
}

func (s *Store) Unlock(password string) error {
	var hash string
	if err := s.db.QueryRow(`select password_hash from vault where id = 1`).Scan(&hash); err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return errors.New("bad vault password")
	}
	s.unlockWith(password)
	return nil
}

func (s *Store) unlockWith(password string) {
	sum := sha3.Sum256([]byte(password))
	s.key = sum[:]
	s.unlocked = true
}
