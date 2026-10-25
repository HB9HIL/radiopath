package web

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/radiopath/radiopath/internal/metrics"
	"github.com/radiopath/radiopath/internal/store"
)

const (
	sessionCookie    = "radiopath_session"
	sessionIdle      = 7 * 24 * time.Hour
	sessionTTL       = 30 * 24 * time.Hour
	bcryptCost       = 12
	maxPasswordBytes = 72 // bcrypt rejects longer input instead of truncating

	loginWindow    = 15 * time.Minute
	loginIPMax     = 20
	loginUserMax   = 5
	registerWindow = time.Hour
	registerIPMax  = 5
)

// clientIP returns the caller's IP; behind a trusted proxy it takes the last X-Forwarded-For entry, which is the one our proxy appended.
func (s *Server) clientIP(r *http.Request) string {
	if s.TrustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// limited reports whether key already has max or more hits within window, without counting a new one.
func (s *Server) limited(r *http.Request, key string, window time.Duration, max int) (bool, error) {
	n, err := s.Store.RateCount(r.Context(), key, window)
	return n >= max, err
}

// tooMany returns the rate-limit error text with the wait time in minutes.
func tooMany(window time.Duration) string {
	return fmt.Sprintf("Too many attempts. Try again in %d minutes.", int(window.Minutes()))
}

type ctxKey int

const userKey ctxKey = 0

// HashPassword returns the bcrypt hash of password at bcryptCost.
func HashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(h), err
}

// randomToken returns 32 random bytes as unpadded base64url.
func randomToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// newSessionToken returns a fresh token for the cookie and its id, which is the only form stored in the database.
func newSessionToken() (token, id string) {
	token = randomToken()
	return token, sessionID(token)
}

// sessionID returns the hex SHA-256 of token, so a leaked database cannot be replayed as cookies.
func sessionID(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// currentUser returns the user that requireAuth put into the request context, or the zero value.
func currentUser(r *http.Request) store.User {
	u, _ := r.Context().Value(userKey).(store.User)
	return u
}

// secureCookie reports whether the request came over HTTPS, directly or via a proxy header.
func secureCookie(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

// sessionUser returns the user of the session cookie, or store.ErrNotFound when it is missing, idle or expired.
func (s *Server) sessionUser(r *http.Request) (store.User, error) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return store.User{}, store.ErrNotFound
	}
	return s.Store.SessionUser(r.Context(), sessionID(c.Value), sessionIdle, sessionTTL)
}

// requireAuth puts the session user into the context, or redirects to /login with a safe return path; store errors give a 500.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, err := s.sessionUser(r); err == nil {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
			return
		} else if !errors.Is(err, store.ErrNotFound) {
			s.fail(w, err)
			return
		}
		next_ := "/"
		if r.Method == http.MethodGet && r.URL.Path != "/" {
			next_ = r.URL.RequestURI()
		}
		http.Redirect(w, r, "/login?next="+url.QueryEscape(next_), http.StatusSeeOther)
	})
}

type loginData struct {
	base
	Next         string
	Error        string
	Notice       string
	Unverified   string
	Registration bool
	Forgot       bool
	Maintenance  bool
}

// loginPage returns login template data with the registration and forgot-password links switched on as configured.
func (s *Server) loginPage(next, errMsg string) loginData {
	return loginData{Next: next, Error: errMsg, Registration: s.Registration, Forgot: s.Mail != nil}
}

// maintenance returns the maintenance flag; ok is false when the lookup failed and the error response was already written.
func (s *Server) maintenance(w http.ResponseWriter, r *http.Request) (bool, bool) {
	on, err := s.Store.Maintenance(r.Context())
	if err != nil {
		s.fail(w, err)
		return false, false
	}
	return on, true
}

// loginForm renders GET /login (503 in maintenance mode) or redirects to / when a session exists.
func (s *Server) loginForm(w http.ResponseWriter, r *http.Request) {
	if _, err := s.sessionUser(r); err == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	on, ok := s.maintenance(w, r)
	if !ok {
		return
	}
	status := http.StatusOK
	if on {
		status = http.StatusServiceUnavailable
	}
	d := s.loginPage(safeNext(r.URL.Query().Get("next")), "")
	d.Maintenance, d.Notice = on, notice(r)
	s.render(w, status, "login", d)
}

// login checks the credentials, counts failures against the IP and user rate limits and starts a session on success; unverified accounts get a 403.
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	name := strings.ToLower(strings.TrimSpace(r.PostFormValue("name")))
	password := r.PostFormValue("password")
	next := safeNext(r.PostFormValue("next"))
	ip := s.clientIP(r)

	if on, ok := s.maintenance(w, r); !ok {
		return
	} else if on {
		s.render(w, http.StatusServiceUnavailable, "login", loginData{Next: next, Maintenance: true})
		return
	}

	ipKey, userKey := "login:ip:"+ip, "login:user:"+strings.ToLower(name)
	for _, l := range []struct {
		key string
		max int
	}{{ipKey, loginIPMax}, {userKey, loginUserMax}} {
		hit, err := s.limited(r, l.key, loginWindow, l.max)
		if err != nil {
			s.fail(w, err)
			return
		}
		if hit {
			metrics.Logins.WithLabelValues("rate_limited").Inc()
			s.Log.Warn("login rate limited", "user", name, "remote", ip)
			s.render(w, http.StatusTooManyRequests, "login", s.loginPage(next, tooMany(loginWindow)))
			return
		}
	}

	u, hash, err := s.Store.Credentials(r.Context(), name)
	if errors.Is(err, store.ErrNotFound) {
		hash, err = "$2a$12$invalidinvalidinvalidinvalidinvalidinvalidinvalidinval", nil
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	// unknown users burn a real bcrypt compare so names cannot be probed
	if u.ID == 0 || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		for _, k := range []string{ipKey, userKey} {
			if _, err := s.Store.RateHit(r.Context(), k, loginWindow); err != nil {
				s.Log.Warn("rate limit", "err", err)
			}
		}
		metrics.Logins.WithLabelValues("failure").Inc()
		s.Log.Warn("login failed", "user", name, "remote", ip)
		s.render(w, http.StatusUnauthorized, "login", s.loginPage(next, "Wrong user name or password."))
		return
	}
	if !u.Verified {
		d := s.loginPage(next, "")
		d.Notice, d.Unverified = notices["unverified"], u.Name
		s.render(w, http.StatusForbidden, "login", d)
		return
	}
	if err := s.startSession(w, r, u.ID); err != nil {
		s.fail(w, err)
		return
	}
	metrics.Logins.WithLabelValues("success").Inc()
	s.Log.Info("login", "user", name)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

// startSession stores a new session for userID and sets its cookie.
func (s *Server) startSession(w http.ResponseWriter, r *http.Request, userID int64) error {
	token, sid := newSessionToken()
	if err := s.Store.CreateSession(r.Context(), sid, userID, time.Now().Add(sessionIdle)); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: secureCookie(r),
		SameSite: http.SameSiteLaxMode, MaxAge: int(sessionTTL.Seconds()),
	})
	return nil
}

var userNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{2,31}$`)

// ValidateCredentials returns the user name and password rule violations; empty means valid.
func ValidateCredentials(name, password string) []string {
	var errs []string
	if !userNameRe.MatchString(name) {
		errs = append(errs, "User name: 3 to 32 characters, letters, digits, . _ -, starting with a letter or digit")
	}
	return append(errs, validatePassword(password)...)
}

// validatePassword returns the password length violations; the upper limit is bcrypt's 72 bytes.
func validatePassword(password string) []string {
	var errs []string
	if len(password) < 10 {
		errs = append(errs, "Password: at least 10 characters")
	}
	if len(password) > maxPasswordBytes {
		errs = append(errs, fmt.Sprintf("Password: at most %d characters", maxPasswordBytes))
	}
	return errs
}

type registerData struct {
	base
	Name   string
	Email  string
	Errors []string
}

// registerForm renders GET /register (503 in maintenance mode) or redirects to / when a session exists.
func (s *Server) registerForm(w http.ResponseWriter, r *http.Request) {
	if _, err := s.sessionUser(r); err == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if on, ok := s.maintenance(w, r); !ok {
		return
	} else if on {
		s.render(w, http.StatusServiceUnavailable, "login", loginData{Maintenance: true})
		return
	}
	s.render(w, http.StatusOK, "register", registerData{})
}

// register handles POST /register: per-IP rate limit, validation, account creation and the confirmation mail, then redirects to /login.
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if on, ok := s.maintenance(w, r); !ok {
		return
	} else if on {
		s.render(w, http.StatusServiceUnavailable, "login", loginData{Maintenance: true})
		return
	}
	ip := s.clientIP(r)
	n, err := s.Store.RateHit(r.Context(), "register:ip:"+ip, registerWindow)
	if err != nil {
		s.fail(w, err)
		return
	}
	if n > registerIPMax {
		s.Log.Warn("registration rate limited", "remote", ip)
		s.render(w, http.StatusTooManyRequests, "register", registerData{Errors: []string{tooMany(registerWindow)}})
		return
	}
	name := strings.ToLower(strings.TrimSpace(r.PostFormValue("name")))
	email := validEmail(r.PostFormValue("email"))
	password := r.PostFormValue("password")
	errs := ValidateCredentials(name, password)
	if email == "" {
		errs = append(errs, "Email: not a valid address")
	}
	if password != r.PostFormValue("password2") {
		errs = append(errs, "Passwords do not match")
	}
	d := registerData{Name: name, Email: r.PostFormValue("email")}
	if len(errs) > 0 {
		d.Errors = errs
		s.render(w, http.StatusUnprocessableEntity, "register", d)
		return
	}
	hash, err := HashPassword(password)
	if err != nil {
		s.fail(w, err)
		return
	}
	id, err := s.Store.Register(r.Context(), name, email, hash)
	if errors.Is(err, store.ErrExists) {
		d.Errors = []string{"User name or address is already taken"}
		s.render(w, http.StatusUnprocessableEntity, "register", d)
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.sendLink(r, store.User{ID: id, Name: name}, email, store.TokenVerify); err != nil {
		s.fail(w, err)
		return
	}
	s.Log.Info("registered", "user", name, "remote", ip)
	http.Redirect(w, r, "/login?msg=registered", http.StatusSeeOther)
}

// logout deletes the session, clears the cookie and redirects to /login.
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		if err := s.Store.DeleteSession(r.Context(), sessionID(c.Value)); err != nil {
			s.Log.Warn("logout", "err", err)
		}
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", HttpOnly: true, MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// safeNext returns next only if it is a local path, otherwise /, so login cannot redirect off site.
func safeNext(next string) string {
	// "//host" and "/\host" (browsers turn \ into /) would leave the site
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, "/\\") {
		return "/"
	}
	return next
}
