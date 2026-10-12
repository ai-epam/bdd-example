// Package auth is a small in-memory login/session service for the demo.
// Passwords are stored as salted PBKDF2 hashes; nothing is persisted.
package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// Login failure reason codes.
const (
	InvalidCredentials = "INVALID_CREDENTIALS"
	MissingCredentials = "MISSING_CREDENTIALS"
	AccountLocked      = "ACCOUNT_LOCKED"
)

const (
	MaxFailures = 3
	LockFor     = 5 * time.Minute
	SessionTTL  = 30 * time.Minute
)

// LoginError carries a reason code.
type LoginError struct{ Reason string }

func (e *LoginError) Error() string { return e.Reason }

type Session struct {
	Token     string    `json:"token"`
	Username  string    `json:"username"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type user struct {
	salt []byte
	hash []byte
}

type attempts struct {
	failures    int
	lockedUntil time.Time
}

type Service struct {
	mu       sync.Mutex
	users    map[string]user
	sessions map[string]Session
	attempts map[string]*attempts
	now      func() time.Time
}

// New returns a service seeded with the demo users.
func New() *Service {
	s := &Service{now: time.Now}
	s.Reset()
	return s
}

// SetClock replaces the time source (tests); nil restores time.Now.
func (s *Service) SetClock(now func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if now == nil {
		now = time.Now
	}
	s.now = now
}

// Reset restores the seed users and clears sessions and lockouts.
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users = map[string]user{}
	s.sessions = map[string]Session{}
	s.attempts = map[string]*attempts{}
	for name, pw := range seedUsers {
		s.users[name] = newUser(pw)
	}
}

func derive(pw string, salt []byte) []byte {
	h, err := pbkdf2.Key(sha256.New, pw, salt, 10_000, 32)
	if err != nil {
		panic(err) // only possible for invalid parameters
	}
	return h
}

func newUser(pw string) user {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		panic(err)
	}
	return user{salt: salt, hash: derive(pw, salt)}
}

func normalize(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

// Login checks credentials and opens a session. Unknown users and wrong
// passwords produce the same error so accounts cannot be enumerated.
func (s *Service) Login(username, password string) (Session, error) {
	name := normalize(username)
	if name == "" || password == "" {
		return Session{}, &LoginError{MissingCredentials}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()

	a := s.attempts[name]
	if a == nil {
		a = &attempts{}
		s.attempts[name] = a
	}
	if now.Before(a.lockedUntil) {
		return Session{}, &LoginError{AccountLocked}
	}

	u, ok := s.users[name]
	if !ok { // still burn a hash so timing is similar
		u = user{salt: make([]byte, 16), hash: make([]byte, 32)}
	}
	match := subtle.ConstantTimeCompare(derive(password, u.salt), u.hash) == 1
	if !ok || !match {
		a.failures++
		if a.failures >= MaxFailures {
			a.failures = 0
			a.lockedUntil = now.Add(LockFor)
		}
		return Session{}, &LoginError{InvalidCredentials}
	}

	a.failures = 0
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	sess := Session{Token: hex.EncodeToString(buf), Username: name, ExpiresAt: now.Add(SessionTTL)}
	s.sessions[sess.Token] = sess
	return sess, nil
}

// Validate returns the session for a live token.
func (s *Service) Validate(token string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[token]
	if !ok {
		return Session{}, false
	}
	if !s.now().Before(sess.ExpiresAt) {
		delete(s.sessions, token)
		return Session{}, false
	}
	return sess, true
}

// Logout invalidates a token (idempotent).
func (s *Service) Logout(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}
