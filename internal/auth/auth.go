// Package auth handles admin login, sessions and request authentication.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	CookieName = "bkk_session"
	SessionTTL = 14 * 24 * time.Hour
	CSRFHeader = "X-BKK-Admin" // custom header: cross-site forms can't send it without a CORS preflight we never allow
	timeLayout = "2006-01-02 15:04:05"
)

var studentIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{2,31}$`)

var (
	ErrBadCredentials = errors.New("wrong student ID or password")
	ErrInvalidID      = errors.New("student ID must be 3–32 letters, digits or dashes")
)

// NormalizeStudentID trims and validates a student ID.
func NormalizeStudentID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if !studentIDPattern.MatchString(id) {
		return "", ErrInvalidID
	}
	return id, nil
}

type Admin struct {
	StudentID string `json:"studentId"`
	FullName  string `json:"fullName"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"createdAt"`
}

type Service struct {
	DB     *sql.DB
	Secure bool // set the Secure cookie flag (HTTPS deployments)
	Now    func() time.Time
}

// Login checks credentials for an active admin and creates a session.
// It returns the raw session token for the cookie.
func (s *Service) Login(ctx context.Context, studentID, password string) (Admin, string, error) {
	var a Admin
	var hash string
	err := s.DB.QueryRowContext(ctx,
		`SELECT student_id, full_name, active, created_at, password_hash FROM admins WHERE student_id = ?`,
		strings.TrimSpace(studentID)).Scan(&a.StudentID, &a.FullName, &a.Active, &a.CreatedAt, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		CheckPassword(dummyHash, password)
		return Admin{}, "", ErrBadCredentials
	}
	if err != nil {
		return Admin{}, "", err
	}
	ok, err := CheckPassword(hash, password)
	if err != nil {
		return Admin{}, "", err
	}
	if !ok || !a.Active {
		return Admin{}, "", ErrBadCredentials
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Admin{}, "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	expires := s.Now().UTC().Add(SessionTTL).Format(timeLayout)
	if _, err := s.DB.ExecContext(ctx, `INSERT INTO sessions (token_hash, student_id, expires_at) VALUES (?, ?, ?)`,
		HashToken(token), a.StudentID, expires); err != nil {
		return Admin{}, "", err
	}
	// Opportunistic cleanup of expired sessions.
	s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, s.Now().UTC().Format(timeLayout))
	return a, token, nil
}

// Logout deletes the session for token.
func (s *Service) Logout(ctx context.Context, token string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, HashToken(token))
	return err
}

// adminForToken returns the active admin owning an unexpired session.
func (s *Service) adminForToken(ctx context.Context, token string) (Admin, error) {
	var a Admin
	err := s.DB.QueryRowContext(ctx, `SELECT a.student_id, a.full_name, a.active, a.created_at
		FROM sessions s JOIN admins a ON a.student_id = s.student_id
		WHERE s.token_hash = ? AND s.expires_at > ? AND a.active = 1`,
		HashToken(token), s.Now().UTC().Format(timeLayout)).Scan(&a.StudentID, &a.FullName, &a.Active, &a.CreatedAt)
	return a, err
}

func (s *Service) SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: token, Path: "/",
		MaxAge: int(SessionTTL.Seconds()), HttpOnly: true, Secure: s.Secure, SameSite: http.SameSiteLaxMode,
	})
}

func (s *Service) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.Secure, SameSite: http.SameSiteLaxMode})
}

type ctxKey struct{}

// Current returns the admin authenticated for this request.
func Current(ctx context.Context) (Admin, bool) {
	a, ok := ctx.Value(ctxKey{}).(Admin)
	return a, ok
}

// Require rejects requests without a valid session, and mutating requests
// without the CSRF header.
func (s *Service) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Header.Get(CSRFHeader) != "1" {
			http.Error(w, `{"error":"missing `+CSRFHeader+` header"}`, http.StatusForbidden)
			return
		}
		c, err := r.Cookie(CookieName)
		if err != nil {
			http.Error(w, `{"error":"not logged in"}`, http.StatusUnauthorized)
			return
		}
		a, err := s.adminForToken(r.Context(), c.Value)
		if err != nil {
			s.ClearCookie(w)
			http.Error(w, `{"error":"not logged in"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, a)))
	})
}

// HashToken is how session tokens are stored: never the raw value.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
