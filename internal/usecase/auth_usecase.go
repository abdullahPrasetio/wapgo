package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	domainrepo "github.com/abdullahPrasetio/wapgo/internal/domain/repository"
	rediscache "github.com/abdullahPrasetio/wapgo/internal/repository/redis"
	"github.com/abdullahPrasetio/wapgo/pkg/auth"
)

// --- DTOs ---

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // access token TTL in seconds
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type LogoutRequest struct {
	// AccessToken is expected from the Authorization header (Bearer), not the body.
	// RefreshToken is optional; if provided the refresh session is also revoked.
	RefreshToken string `json:"refresh_token"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token"    validate:"required"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

// --- Interface ---

type AuthUseCase interface {
	Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error)
	Refresh(ctx context.Context, req *RefreshRequest) (*LoginResponse, error)
	// Logout revokes the access token (from Authorization header value) and,
	// if a refresh token is supplied, also removes the refresh session.
	Logout(ctx context.Context, accessToken string, refreshToken string) error
	// ForgotPassword generates a reset token (stored in Redis for 15 min) and
	// returns it. In production this token would be emailed; here it is returned
	// for testing and development.
	ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) (string, error)
	ResetPassword(ctx context.Context, req *ResetPasswordRequest) error
}

// --- Implementation ---

const (
	refreshKeyPrefix = "auth:refresh:"
	resetKeyPrefix   = "auth:reset:"
	versionKeyPrefix = "auth:ver:" // per-user password version, incremented on password reset
	resetTTL         = 15 * time.Minute
	versionTTL       = 90 * 24 * time.Hour // longer than any refresh token lifetime
)

// refreshSession is the value stored in Redis for each active refresh session.
type refreshSession struct {
	Subject string `json:"sub"`
	Version int    `json:"ver"` // must match the user's current password version
}

type authUseCase struct {
	userRepo      domainrepo.UserRepository
	cacher        domainrepo.Cacher
	jwtCfg        *auth.Config
	refreshCfg    *auth.Config // same secret/iss/aud but longer expiry
	bl            auth.Blacklist
	bcryptCost    int
}

// NewAuthUseCase creates an AuthUseCase.
// refreshExpiry is the lifetime of refresh tokens (e.g. 168*time.Hour for 7 days).
// bcryptCost should be at least 10; pass 0 to use bcrypt.DefaultCost (10).
func NewAuthUseCase(
	userRepo domainrepo.UserRepository,
	cacher domainrepo.Cacher,
	jwtCfg *auth.Config,
	refreshExpiry time.Duration,
	bl auth.Blacklist,
	bcryptCost int,
) AuthUseCase {
	if bcryptCost < 10 {
		bcryptCost = bcrypt.DefaultCost
	}
	refreshCfg := &auth.Config{
		Secret:   jwtCfg.Secret,
		Issuer:   jwtCfg.Issuer,
		Audience: jwtCfg.Audience,
		Expiry:   refreshExpiry,
	}
	return &authUseCase{
		userRepo:   userRepo,
		cacher:     cacher,
		jwtCfg:     jwtCfg,
		refreshCfg: refreshCfg,
		bl:         bl,
		bcryptCost: bcryptCost,
	}
}

// Login validates credentials and returns an access+refresh token pair.
func (a *authUseCase) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	user, err := a.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("login find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return a.issueTokenPair(ctx, user.ID.String(), user.Email)
}

// Refresh verifies a refresh token, rotates it, and issues a new pair.
func (a *authUseCase) Refresh(ctx context.Context, req *RefreshRequest) (*LoginResponse, error) {
	claims, err := auth.Verify(req.RefreshToken, a.refreshCfg)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Verify refresh session still exists (not already rotated or logged out).
	sessionKey := refreshKeyPrefix + claims.ID
	var session refreshSession
	if err := a.cacher.Get(ctx, sessionKey, &session); err != nil {
		if errors.Is(err, rediscache.ErrCacheMiss) {
			return nil, ErrInvalidToken // session expired or revoked
		}
		return nil, fmt.Errorf("refresh session lookup: %w", err)
	}

	if session.Subject != claims.Subject {
		return nil, ErrInvalidToken
	}

	// Reject sessions that predate the last password reset.
	var currentVersion int
	if err := a.cacher.Get(ctx, versionKeyPrefix+claims.Subject, &currentVersion); err != nil {
		currentVersion = 0 // no password reset yet
	}
	if session.Version < currentVersion {
		_ = a.cacher.Del(ctx, sessionKey)
		return nil, ErrInvalidToken
	}

	// Revoke old refresh session before issuing new pair (rotation).
	_ = a.cacher.Del(ctx, sessionKey)

	// Verify the user still exists and is active (handles deleted/banned accounts).
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if _, err := a.userRepo.FindByID(ctx, userID); err != nil {
		return nil, ErrInvalidToken
	}

	return a.issueTokenPair(ctx, claims.Subject, "")
}

// Logout revokes the access token and optionally the refresh session.
func (a *authUseCase) Logout(ctx context.Context, accessToken string, refreshToken string) error {
	// Revoke access token via blacklist.
	accessClaims, err := auth.Verify(accessToken, a.jwtCfg)
	if err != nil {
		return ErrInvalidToken
	}
	if accessClaims.ID != "" && a.bl != nil {
		remaining := time.Until(accessClaims.ExpiresAt.Time)
		if remaining > 0 {
			if err := a.bl.Revoke(ctx, accessClaims.ID, remaining); err != nil {
				return fmt.Errorf("revoke access token: %w", err)
			}
		}
	}

	// Optionally revoke refresh session AND blacklist the refresh token JTI so
	// it cannot be used as a Bearer access token after logout.
	if refreshToken != "" {
		if refreshClaims, err := auth.Verify(refreshToken, a.refreshCfg); err == nil {
			_ = a.cacher.Del(ctx, refreshKeyPrefix+refreshClaims.ID)
			if refreshClaims.ID != "" && a.bl != nil {
				if remaining := time.Until(refreshClaims.ExpiresAt.Time); remaining > 0 {
					_ = a.bl.Revoke(ctx, refreshClaims.ID, remaining)
				}
			}
		}
	}

	return nil
}

// ForgotPassword generates a time-limited reset token stored in Redis.
// In production integrate with an email delivery service.
func (a *authUseCase) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) (string, error) {
	// Verify email exists without leaking existence via timing attack.
	user, err := a.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return success regardless — prevents email enumeration.
			return "", nil
		}
		return "", fmt.Errorf("forgot password: %w", err)
	}

	token := uuid.New().String()
	key := resetKeyPrefix + token
	if err := a.cacher.Set(ctx, key, user.Email, resetTTL); err != nil {
		return "", fmt.Errorf("store reset token: %w", err)
	}
	return token, nil
}

// ResetPassword verifies the reset token and updates the user's password.
func (a *authUseCase) ResetPassword(ctx context.Context, req *ResetPasswordRequest) error {
	key := resetKeyPrefix + req.Token
	var email string
	if err := a.cacher.Get(ctx, key, &email); err != nil {
		if errors.Is(err, rediscache.ErrCacheMiss) {
			return ErrInvalidToken // expired or already used
		}
		return fmt.Errorf("reset password lookup: %w", err)
	}

	user, err := a.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("reset password find user: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), a.bcryptCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	user.Password = string(hash)
	if err := a.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	// Consume the reset token — single use.
	_ = a.cacher.Del(ctx, key)

	// Invalidate all active refresh sessions by bumping the user's password version.
	var currentVersion int
	if err := a.cacher.Get(ctx, versionKeyPrefix+user.ID.String(), &currentVersion); err != nil {
		currentVersion = 0
	}
	_ = a.cacher.Set(ctx, versionKeyPrefix+user.ID.String(), currentVersion+1, versionTTL)

	return nil
}

// issueTokenPair signs an access + refresh token and persists the refresh session.
// subject is the authoritative user identity (UUID string).
func (a *authUseCase) issueTokenPair(ctx context.Context, subject string, _ string) (*LoginResponse, error) {
	accessToken, _, err := auth.Sign(subject, nil, "access", a.jwtCfg)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	// Sign returns the JTI directly — no round-trip Verify needed.
	refreshToken, refreshJTI, err := auth.Sign(subject, nil, "refresh", a.refreshCfg)
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	// Embed the current password version so the session can be invalidated on reset.
	var pwVersion int
	if err := a.cacher.Get(ctx, versionKeyPrefix+subject, &pwVersion); err != nil {
		pwVersion = 0
	}

	sessionKey := refreshKeyPrefix + refreshJTI
	if err := a.cacher.Set(ctx, sessionKey, refreshSession{Subject: subject, Version: pwVersion}, a.refreshCfg.Expiry); err != nil {
		return nil, fmt.Errorf("store refresh session: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(a.jwtCfg.Expiry.Seconds()),
	}, nil
}

// --- Auth-specific sentinel errors ---

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

