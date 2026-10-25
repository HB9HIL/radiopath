package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	netmail "net/mail"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/crypto/bcrypt"

	"github.com/radiopath/radiopath/internal/store"
)

const (
	verifyTTL   = 24 * time.Hour
	resetTTL    = time.Hour
	maxEmailLen = 254

	mailWindow    = time.Hour
	forgotIPMax   = 5
	forgotMailMax = 3
	verifyUserMax = 3
)

var notices = map[string]string{
	"verified": "Address confirmed.",
	"sent":     "Confirmation link sent to the new address. It takes effect once you open it.",
	"password": "Password changed. Other devices have been logged out.",
}

// notice returns the message for the ?msg= key of the request, or an empty string.
func notice(r *http.Request) string { return notices[r.URL.Query().Get("msg")] }

const (
	verifyMail = `Hi %s,

confirm this address for your Radiopath account by opening

%s

The link is valid for 24 hours. If you did not ask for this, ignore this mail.
`
	resetMail = `Hi %s,

somebody asked to reset the password of your Radiopath account. Open

%s

to choose a new one; the link is valid for one hour. If that was not you,
ignore this mail, your password stays as it is.
`
)

// validEmail returns s trimmed if it is a plain address (no display name, max 254 bytes), otherwise an empty string.
func validEmail(s string) string {
	s = strings.TrimSpace(s)
	a, err := netmail.ParseAddress(s)
	if err != nil || a.Name != "" || a.Address != s || len(s) > maxEmailLen {
		return ""
	}
	return s
}

// sendLink stores a one-time token for purpose and mails the matching confirm or reset link to email.
func (s *Server) sendLink(ctx context.Context, u store.User, email, purpose string) error {
	token, id := newSessionToken()
	ttl, path, subject, text := resetTTL, "/reset/", "Reset your Radiopath password", resetMail
	if purpose == store.TokenVerify {
		ttl, path, subject, text = verifyTTL, "/verify/", "Confirm your Radiopath address", verifyMail
	}
	if err := s.Store.CreateEmailToken(ctx, id, u.ID, email, purpose, time.Now().Add(ttl)); err != nil {
		return err
	}
	return s.Mail.Send(ctx, email, subject, fmt.Sprintf(text, u.Name, s.BaseURL+path+token))
}

// sendVerification mails a confirmation link, limited per user; sent is false when the limit was hit.
func (s *Server) sendVerification(ctx context.Context, u store.User, email string) (bool, error) {
	n, err := s.Store.RateHit(ctx, fmt.Sprintf("verify:user:%d", u.ID), mailWindow)
	if err != nil {
		return false, err
	}
	if n > verifyUserMax {
		s.Log.Warn("confirmation mail rate limited", "user", u.Name)
		return false, nil
	}
	return true, s.sendLink(ctx, u, email, store.TokenVerify)
}

type accountData struct {
	base
	Email    string
	Verified bool
	Notice   string
	Errors   []string
}

// accountPage renders the account page with the given status and form errors.
func (s *Server) accountPage(w http.ResponseWriter, r *http.Request, status int, errs []string) {
	u := currentUser(r)
	s.render(w, status, "account", accountData{base: s.base(r), Email: u.Email, Verified: u.Verified, Notice: notice(r), Errors: errs})
}

// account renders GET /account.
func (s *Server) account(w http.ResponseWriter, r *http.Request) {
	s.accountPage(w, r, http.StatusOK, nil)
}

// accountEmail handles POST /account/email: mails a confirmation link to the new address, which only takes effect once opened.
func (s *Server) accountEmail(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	email := validEmail(r.PostFormValue("email"))
	if email == "" {
		s.accountPage(w, r, http.StatusUnprocessableEntity, []string{"Email: not a valid address"})
		return
	}
	if strings.EqualFold(email, u.Email) {
		http.Redirect(w, r, "/account", http.StatusSeeOther)
		return
	}
	sent, err := s.sendVerification(r.Context(), u, email)
	if err != nil {
		s.fail(w, err)
		return
	}
	if !sent {
		s.accountPage(w, r, http.StatusTooManyRequests, []string{tooMany(mailWindow)})
		return
	}
	s.Log.Info("email change requested", "user", u.Name)
	http.Redirect(w, r, "/account?msg=sent", http.StatusSeeOther)
}

// accountPassword handles POST /account/password: checks the current password, sets the new one, logs out all other sessions and starts a fresh one.
func (s *Server) accountPassword(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	_, hash, err := s.Store.Credentials(r.Context(), u.Name)
	if err != nil {
		s.fail(w, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(r.PostFormValue("current"))) != nil {
		s.accountPage(w, r, http.StatusUnprocessableEntity, []string{"Current password is wrong"})
		return
	}
	password := r.PostFormValue("password")
	errs := validatePassword(password)
	if password != r.PostFormValue("password2") {
		errs = append(errs, "Passwords do not match")
	}
	if len(errs) > 0 {
		s.accountPage(w, r, http.StatusUnprocessableEntity, errs)
		return
	}
	if err := s.setPassword(r.Context(), u, password); err != nil {
		s.fail(w, err)
		return
	}
	if err := s.startSession(w, r, u.ID); err != nil {
		s.fail(w, err)
		return
	}
	s.Log.Info("password changed", "user", u.Name)
	http.Redirect(w, r, "/account?msg=password", http.StatusSeeOther)
}

// setPassword stores the new hash for u and revokes all of its sessions.
func (s *Server) setPassword(ctx context.Context, u store.User, password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	if err := s.Store.SetPassword(ctx, u.Name, hash); err != nil {
		return err
	}
	_, err = s.Store.DeleteUserSessions(ctx, u.Name)
	return err
}

type NameInput struct {
	Body struct {
		Name string `json:"name"`
	}
}

type EmailInput struct {
	Body struct {
		Email string `json:"email"`
	}
}

type TokenInput struct {
	Body struct {
		Token string `json:"token" doc:"The token from the mailed link"`
	}
}

type ResetInput struct {
	Body struct {
		Token    string `json:"token" doc:"The token from the mailed link"`
		Password string `json:"password"`
	}
}

const staleLink = "This link has expired or was already used. Request a new one."

// registerMailFlows registers address confirmation and password reset; without a mailer every one answers 404.
func (s *Server) registerMailFlows(api huma.API) {
	noMail := huma.Error404NotFound("Mail is not configured")

	o := op("verify-email", http.MethodPost, "/api/verify", "", "Confirm an address with the token from the mail")
	o.DefaultStatus = http.StatusNoContent
	o.Errors = []int{http.StatusNotFound, http.StatusConflict, http.StatusGone}
	o.Description = "410 for an expired or used token, 409 if another account owns the address."
	huma.Register(api, o, func(ctx context.Context, in *TokenInput) (*struct{}, error) {
		if s.Mail == nil {
			return nil, noMail
		}
		u, email, err := s.Store.ConsumeEmailToken(ctx, sessionID(in.Body.Token), store.TokenVerify)
		if errors.Is(err, store.ErrNotFound) {
			return nil, huma.Error410Gone("This confirmation link has expired or was already used. Log in to get a new one.")
		}
		if err != nil {
			return nil, s.storeErr(err)
		}
		err = s.Store.SetEmail(ctx, u.ID, email)
		if errors.Is(err, store.ErrExists) {
			return nil, huma.Error409Conflict("That address is already used by another account.")
		}
		if err != nil {
			return nil, s.storeErr(err)
		}
		s.Log.Info("email confirmed", "user", u.Name)
		return nil, nil
	})

	o = op("resend-verification", http.MethodPost, "/api/verify/resend", "", "Mail a new confirmation link to an unconfirmed account")
	o.DefaultStatus = http.StatusNoContent
	o.Description = "Answers the same for unknown and confirmed names, so accounts cannot be probed."
	huma.Register(api, o, func(ctx context.Context, in *NameInput) (*struct{}, error) {
		if s.Mail == nil {
			return nil, noMail
		}
		u, _, err := s.Store.Credentials(ctx, strings.ToLower(strings.TrimSpace(in.Body.Name)))
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			return nil, s.storeErr(err)
		}
		if err == nil && !u.Verified {
			if _, err := s.sendVerification(ctx, u, u.Email); err != nil {
				return nil, s.storeErr(err)
			}
			s.Log.Info("confirmation resent", "user", u.Name)
		}
		return nil, nil
	})

	o = op("forgot-password", http.MethodPost, "/api/password/forgot", "", "Mail a password reset link")
	o.DefaultStatus = http.StatusNoContent
	o.Errors = []int{http.StatusNotFound, http.StatusTooManyRequests}
	o.Description = "Answers the same whether the address belongs to an account or not."
	huma.Register(api, o, func(ctx context.Context, in *EmailInput) (*struct{}, error) {
		if s.Mail == nil {
			return nil, noMail
		}
		email := validEmail(in.Body.Email)
		if email == "" {
			return nil, huma.Error422UnprocessableEntity("Invalid input",
				&huma.ErrorDetail{Location: "body.email", Message: "Email: not a valid address", Value: in.Body.Email})
		}
		ip := reqInfo(ctx).ip
		for _, l := range []struct {
			key string
			max int
		}{{"forgot:ip:" + ip, forgotIPMax}, {"forgot:mail:" + strings.ToLower(email), forgotMailMax}} {
			n, err := s.Store.RateHit(ctx, l.key, mailWindow)
			if err != nil {
				return nil, s.storeErr(err)
			}
			if n > l.max {
				s.Log.Warn("reset rate limited", "remote", ip)
				return nil, huma.Error429TooManyRequests(tooMany(mailWindow))
			}
		}
		u, err := s.Store.UserByEmail(ctx, email)
		if errors.Is(err, store.ErrNotFound) {
			s.Log.Info("reset requested for unknown address", "remote", ip)
			return nil, nil
		}
		if err != nil {
			return nil, s.storeErr(err)
		}
		if err := s.sendLink(ctx, u, email, store.TokenReset); err != nil {
			return nil, s.storeErr(err)
		}
		s.Log.Info("reset link sent", "user", u.Name)
		return nil, nil
	})

	o = op("check-reset-token", http.MethodGet, "/api/password/reset/{token}", "", "Check that a reset token is still valid")
	o.DefaultStatus = http.StatusNoContent
	o.Errors = []int{http.StatusNotFound, http.StatusGone}
	huma.Register(api, o, func(ctx context.Context, in *struct {
		Token string `path:"token"`
	}) (*struct{}, error) {
		if s.Mail == nil {
			return nil, noMail
		}
		ok, err := s.Store.EmailTokenValid(ctx, sessionID(in.Token), store.TokenReset)
		if err != nil {
			return nil, s.storeErr(err)
		}
		if !ok {
			return nil, huma.Error410Gone(staleLink)
		}
		return nil, nil
	})

	o = op("reset-password", http.MethodPost, "/api/password/reset", "", "Set a new password with the token from the mail")
	o.DefaultStatus = http.StatusNoContent
	o.Errors = []int{http.StatusNotFound, http.StatusGone}
	o.Description = "Logs out every session of the account."
	huma.Register(api, o, func(ctx context.Context, in *ResetInput) (*struct{}, error) {
		if s.Mail == nil {
			return nil, noMail
		}
		var errs []error
		for _, m := range validatePassword(in.Body.Password) {
			errs = append(errs, &huma.ErrorDetail{Location: "body.password", Message: m})
		}
		if len(errs) > 0 {
			return nil, huma.Error422UnprocessableEntity("Invalid input", errs...)
		}
		u, _, err := s.Store.ConsumeEmailToken(ctx, sessionID(in.Body.Token), store.TokenReset)
		if errors.Is(err, store.ErrNotFound) {
			return nil, huma.Error410Gone(staleLink)
		}
		if err != nil {
			return nil, s.storeErr(err)
		}
		if err := s.setPassword(ctx, u, in.Body.Password); err != nil {
			return nil, s.storeErr(err)
		}
		s.Log.Info("password reset", "user", u.Name)
		return nil, nil
	})
}
