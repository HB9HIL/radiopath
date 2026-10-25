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
	"regexp"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
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

// currentUser returns the user that requireUser put into the request context, or the zero value.
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

// requireUser puts the session user into the context, or answers 401; store errors give a 500.
func (s *Server) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := s.sessionUser(r)
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not logged in", http.StatusUnauthorized)
			return
		}
		if err != nil {
			s.fail(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
	})
}

const maintenanceText = "Radiopath is closed for maintenance. Logins are disabled, please try again later."

// maintenanceErr returns a 503 while maintenance mode is on, nil otherwise.
func (s *Server) maintenanceErr(ctx context.Context) error {
	on, err := s.Store.Maintenance(ctx)
	if err != nil {
		return s.storeErr(err)
	}
	if on {
		return huma.Error503ServiceUnavailable(maintenanceText)
	}
	return nil
}

type LoginInput struct {
	Body struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
}

type SessionOutput struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
}

type Me struct {
	Name     string `json:"name"`
	Email    string `json:"email" doc:"Empty until an address is confirmed"`
	Verified bool   `json:"verified"`
}

type MeOutput struct{ Body Me }

// registerAuth registers login, logout, registration and the current user.
func (s *Server) registerAuth(api huma.API) {
	o := op("create-session", http.MethodPost, "/api/session", "", "Log in and set the session cookie")
	o.DefaultStatus = http.StatusNoContent
	o.Errors = []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests, http.StatusServiceUnavailable}
	o.Description = "403 means the address of the account is not confirmed yet."
	huma.Register(api, o, s.login)

	o = op("delete-session", http.MethodDelete, "/api/session", "", "Log out and clear the session cookie")
	o.DefaultStatus = http.StatusNoContent
	huma.Register(api, o, func(ctx context.Context, _ *struct{}) (*SessionOutput, error) {
		if c := reqInfo(ctx).cookie; c != "" {
			if err := s.Store.DeleteSession(ctx, sessionID(c)); err != nil {
				s.Log.Warn("logout", "err", err)
			}
		}
		return &SessionOutput{SetCookie: http.Cookie{Name: sessionCookie, Value: "", Path: "/", HttpOnly: true, MaxAge: -1}}, nil
	})

	o = op("register", http.MethodPost, "/api/register", "", "Create an account and mail the confirmation link")
	o.DefaultStatus = http.StatusNoContent
	o.Errors = []int{http.StatusNotFound, http.StatusConflict, http.StatusTooManyRequests, http.StatusServiceUnavailable}
	o.Description = "404 when registration is closed."
	huma.Register(api, o, s.register)

	huma.Register(api, op("get-me", http.MethodGet, "/api/me", scopeAccountRead, "The logged-in user"),
		func(ctx context.Context, _ *struct{}) (*MeOutput, error) {
			u := userOf(ctx)
			return &MeOutput{Body: Me{Name: u.Name, Email: u.Email, Verified: u.Verified}}, nil
		})
}

// login checks the credentials, counts failures against the IP and user rate limits and starts a session on success; unverified accounts get a 403.
func (s *Server) login(ctx context.Context, in *LoginInput) (*SessionOutput, error) {
	name := strings.ToLower(strings.TrimSpace(in.Body.Name))
	ip := reqInfo(ctx).ip
	if err := s.maintenanceErr(ctx); err != nil {
		return nil, err
	}

	ipKey, nameKey := "login:ip:"+ip, "login:user:"+name
	for _, l := range []struct {
		key string
		max int
	}{{ipKey, loginIPMax}, {nameKey, loginUserMax}} {
		n, err := s.Store.RateCount(ctx, l.key, loginWindow)
		if err != nil {
			return nil, s.storeErr(err)
		}
		if n >= l.max {
			metrics.Logins.WithLabelValues("rate_limited").Inc()
			s.Log.Warn("login rate limited", "user", name, "remote", ip)
			return nil, huma.Error429TooManyRequests(tooMany(loginWindow))
		}
	}

	u, hash, err := s.Store.Credentials(ctx, name)
	if errors.Is(err, store.ErrNotFound) {
		hash, err = "$2a$12$invalidinvalidinvalidinvalidinvalidinvalidinvalidinval", nil
	}
	if err != nil {
		return nil, s.storeErr(err)
	}
	// unknown users burn a real bcrypt compare so names cannot be probed
	if u.ID == 0 || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Body.Password)) != nil {
		for _, k := range []string{ipKey, nameKey} {
			if _, err := s.Store.RateHit(ctx, k, loginWindow); err != nil {
				s.Log.Warn("rate limit", "err", err)
			}
		}
		metrics.Logins.WithLabelValues("failure").Inc()
		s.Log.Warn("login failed", "user", name, "remote", ip)
		return nil, huma.Error401Unauthorized("Wrong user name or password.")
	}
	if !u.Verified {
		return nil, huma.Error403Forbidden("Your address is not confirmed yet. Open the link from the mail we sent you, or request a new one.")
	}
	c, err := s.newSession(ctx, u.ID)
	if err != nil {
		return nil, s.storeErr(err)
	}
	metrics.Logins.WithLabelValues("success").Inc()
	s.Log.Info("login", "user", name)
	return &SessionOutput{SetCookie: c}, nil
}

// newSession stores a new session for userID and returns its cookie.
func (s *Server) newSession(ctx context.Context, userID int64) (http.Cookie, error) {
	token, sid := newSessionToken()
	if err := s.Store.CreateSession(ctx, sid, userID, time.Now().Add(sessionIdle)); err != nil {
		return http.Cookie{}, err
	}
	return http.Cookie{
		Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: reqInfo(ctx).secure,
		SameSite: http.SameSiteLaxMode, MaxAge: int(sessionTTL.Seconds()),
	}, nil
}

var userNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{2,31}$`)

const userNameRule = "User name: 3 to 32 characters, letters, digits, . _ -, starting with a letter or digit"

// ValidateCredentials returns the user name and password rule violations; empty means valid.
func ValidateCredentials(name, password string) []string {
	var errs []string
	if !userNameRe.MatchString(name) {
		errs = append(errs, userNameRule)
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

type RegisterInput struct {
	Body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
}

// register handles POST /api/register: per-IP rate limit, validation, account creation and the confirmation mail.
func (s *Server) register(ctx context.Context, in *RegisterInput) (*struct{}, error) {
	if !s.Registration || s.Mail == nil {
		return nil, huma.Error404NotFound("Registration is closed")
	}
	if err := s.maintenanceErr(ctx); err != nil {
		return nil, err
	}
	ip := reqInfo(ctx).ip
	n, err := s.Store.RateHit(ctx, "register:ip:"+ip, registerWindow)
	if err != nil {
		return nil, s.storeErr(err)
	}
	if n > registerIPMax {
		s.Log.Warn("registration rate limited", "remote", ip)
		return nil, huma.Error429TooManyRequests(tooMany(registerWindow))
	}
	name := strings.ToLower(strings.TrimSpace(in.Body.Name))
	email := validEmail(in.Body.Email)
	var errs []error
	if !userNameRe.MatchString(name) {
		errs = append(errs, &huma.ErrorDetail{Location: "body.name", Message: userNameRule, Value: in.Body.Name})
	}
	if email == "" {
		errs = append(errs, &huma.ErrorDetail{Location: "body.email", Message: "Email: not a valid address", Value: in.Body.Email})
	}
	for _, m := range validatePassword(in.Body.Password) {
		errs = append(errs, &huma.ErrorDetail{Location: "body.password", Message: m})
	}
	if len(errs) > 0 {
		return nil, huma.Error422UnprocessableEntity("Invalid input", errs...)
	}
	hash, err := HashPassword(in.Body.Password)
	if err != nil {
		return nil, s.storeErr(err)
	}
	id, err := s.Store.Register(ctx, name, email, hash)
	if errors.Is(err, store.ErrExists) {
		return nil, huma.Error409Conflict("User name or address is already taken")
	}
	if err != nil {
		return nil, s.storeErr(err)
	}
	if err := s.sendLink(ctx, store.User{ID: id, Name: name}, email, store.TokenVerify); err != nil {
		return nil, s.storeErr(err)
	}
	s.Log.Info("registered", "user", name, "remote", ip)
	return nil, nil
}
