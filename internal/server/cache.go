package server

import (
	"time"
)

type RateLimitEntry struct {
	Count     int
	StartTime time.Time
}

type VerificationCodeEntry struct {
	Code      string
	CreatedAt time.Time
	Attempts  int
}

// RegisterEmailAttempt checks if an email is allowed to receive a code and updates the rate limiter.
func (s *Server) RegisterEmailAttempt(email string) bool {
	// Cleanup expired entries
	s.cleanupExpired()

	s.RateLimiterMu.Lock()
	defer s.RateLimiterMu.Unlock()

	entry, exists := s.EmailRateLimiter[email]

	if !exists || time.Since(entry.StartTime) > 3*time.Minute {
		s.EmailRateLimiter[email] = &RateLimitEntry{
			Count:     1,
			StartTime: time.Now(),
		}
		return true
	}

	if entry.Count >= 3 {
		return false
	}

	entry.Count++
	return true
}

// StoreVerificationCode saves a new verification code for an email.
func (s *Server) StoreVerificationCode(email, code string) {
	// Cleanup expired entries
	s.cleanupExpired()

	s.CodesMu.Lock()
	defer s.CodesMu.Unlock()
	s.VerificationCodes[email] = &VerificationCodeEntry{
		Code:      code,
		CreatedAt: time.Now(),
	}
}

// VerifyCode checks if the provided code is correct for the given email.
// It tracks failed attempts and deletes the code if a limit is reached to prevent brute force.
func (s *Server) VerifyCode(email, code string) bool {
	s.CodesMu.Lock()
	defer s.CodesMu.Unlock()

	entry, exists := s.VerificationCodes[email]
	if !exists {
		return false
	}

	if entry.Code != code {
		entry.Attempts++
		if entry.Attempts >= 3 {
			delete(s.VerificationCodes, email)
		}
		return false
	}

	return true
}

// DeleteVerificationCode removes the verification code for the given email.
func (s *Server) DeleteVerificationCode(email string) {
	s.CodesMu.Lock()
	defer s.CodesMu.Unlock()
	delete(s.VerificationCodes, email)
}

// cleanupExpired removes entries older than 1 hour from the in-memory caches.
func (s *Server) cleanupExpired() {
	now := time.Now()
	expirationLimit := 1 * time.Hour

	// Cleanup EmailRateLimiter
	s.RateLimiterMu.Lock()
	for email, entry := range s.EmailRateLimiter {
		if now.Sub(entry.StartTime) > expirationLimit {
			delete(s.EmailRateLimiter, email)
		}
	}
	s.RateLimiterMu.Unlock()

	// Cleanup VerificationCodes
	s.CodesMu.Lock()
	for email, entry := range s.VerificationCodes {
		if now.Sub(entry.CreatedAt) > expirationLimit {
			delete(s.VerificationCodes, email)
		}
	}
	s.CodesMu.Unlock()
}
