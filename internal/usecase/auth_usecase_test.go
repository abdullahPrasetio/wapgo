package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/abdullahPrasetio/wapgo/internal/domain/entity"
	rediscache "github.com/abdullahPrasetio/wapgo/internal/repository/redis"
	"github.com/abdullahPrasetio/wapgo/pkg/auth"
	"github.com/abdullahPrasetio/wapgo/pkg/pagination"
)

// --- mocks ---

type mockAuthUserRepo struct {
	users map[string]*entity.User
}

func (m *mockAuthUserRepo) FindByEmail(_ context.Context, email string) (*entity.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return u, nil
}

func (m *mockAuthUserRepo) FindByID(_ context.Context, id uuid.UUID) (*entity.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockAuthUserRepo) Update(_ context.Context, user *entity.User) error {
	m.users[user.Email] = user
	return nil
}

func (m *mockAuthUserRepo) FindAll(_ context.Context) ([]*entity.User, error) { return nil, nil }
func (m *mockAuthUserRepo) FindAllPaged(_ context.Context, _ *pagination.Request) ([]*entity.User, int, error) {
	return nil, 0, nil
}
func (m *mockAuthUserRepo) Create(_ context.Context, _ *entity.User) error { return nil }
func (m *mockAuthUserRepo) Delete(_ context.Context, _ uuid.UUID) error    { return nil }
func (m *mockAuthUserRepo) ExistsByEmail(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// mockCacher is an in-memory Cacher that mirrors the real Redis JSON marshal/unmarshal
// behaviour so tests accurately reflect production semantics.
type mockCacher struct {
	data map[string][]byte
}

func newMockCacher() *mockCacher { return &mockCacher{data: map[string][]byte{}} }

func (c *mockCacher) Set(_ context.Context, key string, value interface{}, _ time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.data[key] = b
	return nil
}

func (c *mockCacher) Get(_ context.Context, key string, dest interface{}) error {
	b, ok := c.data[key]
	if !ok {
		return rediscache.ErrCacheMiss
	}
	return json.Unmarshal(b, dest)
}

func (c *mockCacher) Del(_ context.Context, keys ...string) error {
	for _, k := range keys {
		delete(c.data, k)
	}
	return nil
}

func (c *mockCacher) Exists(_ context.Context, key string) (bool, error) {
	_, ok := c.data[key]
	return ok, nil
}

// mockBlacklist is an in-memory Blacklist.
type mockBlacklist struct {
	revoked map[string]bool
}

func newMockBlacklist() *mockBlacklist { return &mockBlacklist{revoked: map[string]bool{}} }

func (b *mockBlacklist) Revoke(_ context.Context, jti string, _ time.Duration) error {
	b.revoked[jti] = true
	return nil
}

func (b *mockBlacklist) IsRevoked(_ context.Context, jti string) (bool, error) {
	return b.revoked[jti], nil
}

// --- helpers ---

func testJWTCfg() *auth.Config {
	return &auth.Config{
		Secret:   "test-secret-at-least-32-bytes-long!",
		Issuer:   "wapgo-test",
		Audience: "wapgo-test-api",
		Expiry:   15 * time.Minute,
	}
}

func newTestAuthUC(repo *mockAuthUserRepo, cacher *mockCacher, bl *mockBlacklist) AuthUseCase {
	return NewAuthUseCase(repo, cacher, testJWTCfg(), 7*24*time.Hour, bl, bcrypt.MinCost)
}

func hashPassword(t *testing.T, pw string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(h)
}

// --- tests ---

func TestAuthUseCase_Login_Success(t *testing.T) {
	repo := &mockAuthUserRepo{users: map[string]*entity.User{
		"alice@example.com": {
			ID:       uuid.New(),
			Email:    "alice@example.com",
			Password: hashPassword(t, "secret1234"),
		},
	}}
	uc := newTestAuthUC(repo, newMockCacher(), newMockBlacklist())

	resp, err := uc.Login(context.Background(), &LoginRequest{Email: "alice@example.com", Password: "secret1234"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("expected non-empty tokens")
	}
	if resp.ExpiresIn <= 0 {
		t.Fatal("expected positive expires_in")
	}
}

func TestAuthUseCase_Login_WrongPassword(t *testing.T) {
	repo := &mockAuthUserRepo{users: map[string]*entity.User{
		"alice@example.com": {ID: uuid.New(), Email: "alice@example.com", Password: hashPassword(t, "correct")},
	}}
	uc := newTestAuthUC(repo, newMockCacher(), newMockBlacklist())

	_, err := uc.Login(context.Background(), &LoginRequest{Email: "alice@example.com", Password: "wrong"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthUseCase_Login_UnknownEmail(t *testing.T) {
	repo := &mockAuthUserRepo{users: map[string]*entity.User{}}
	uc := newTestAuthUC(repo, newMockCacher(), newMockBlacklist())

	_, err := uc.Login(context.Background(), &LoginRequest{Email: "nobody@example.com", Password: "pw"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthUseCase_Refresh_Success(t *testing.T) {
	repo := &mockAuthUserRepo{users: map[string]*entity.User{
		"bob@example.com": {ID: uuid.New(), Email: "bob@example.com", Password: hashPassword(t, "pw")},
	}}
	cacher := newMockCacher()
	uc := newTestAuthUC(repo, cacher, newMockBlacklist())

	loginResp, _ := uc.Login(context.Background(), &LoginRequest{Email: "bob@example.com", Password: "pw"})

	refreshResp, err := uc.Refresh(context.Background(), &RefreshRequest{RefreshToken: loginResp.RefreshToken})
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if refreshResp.AccessToken == "" || refreshResp.RefreshToken == "" {
		t.Fatal("expected new tokens")
	}
	// Old refresh token must now be invalid (rotated).
	_, err = uc.Refresh(context.Background(), &RefreshRequest{RefreshToken: loginResp.RefreshToken})
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken after rotation, got %v", err)
	}
}

func TestAuthUseCase_Refresh_InvalidToken(t *testing.T) {
	uc := newTestAuthUC(&mockAuthUserRepo{users: map[string]*entity.User{}}, newMockCacher(), newMockBlacklist())

	_, err := uc.Refresh(context.Background(), &RefreshRequest{RefreshToken: "not.a.valid.token"})
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestAuthUseCase_Logout_RevokesToken(t *testing.T) {
	repo := &mockAuthUserRepo{users: map[string]*entity.User{
		"carol@example.com": {ID: uuid.New(), Email: "carol@example.com", Password: hashPassword(t, "pw")},
	}}
	bl := newMockBlacklist()
	uc := newTestAuthUC(repo, newMockCacher(), bl)

	loginResp, _ := uc.Login(context.Background(), &LoginRequest{Email: "carol@example.com", Password: "pw"})

	if err := uc.Logout(context.Background(), loginResp.AccessToken, loginResp.RefreshToken); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	// Access token JTI should now be in the blacklist.
	cfg := testJWTCfg()
	claims, _ := auth.Verify(loginResp.AccessToken, cfg)
	if claims != nil {
		revoked, _ := bl.IsRevoked(context.Background(), claims.ID)
		if !revoked {
			t.Fatal("expected access token to be blacklisted after logout")
		}
	}
}

func TestAuthUseCase_ForgotPassword_UnknownEmail(t *testing.T) {
	uc := newTestAuthUC(&mockAuthUserRepo{users: map[string]*entity.User{}}, newMockCacher(), newMockBlacklist())

	token, err := uc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "ghost@example.com"})
	// Should succeed silently (anti-enumeration).
	if err != nil {
		t.Fatalf("ForgotPassword unexpected error: %v", err)
	}
	if token != "" {
		t.Fatal("expected empty token for unknown email")
	}
}

func TestAuthUseCase_ResetPassword_Success(t *testing.T) {
	userID := uuid.New()
	repo := &mockAuthUserRepo{users: map[string]*entity.User{
		"dave@example.com": {ID: userID, Email: "dave@example.com", Password: hashPassword(t, "oldpw")},
	}}
	cacher := newMockCacher()
	uc := newTestAuthUC(repo, cacher, newMockBlacklist())

	token, err := uc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "dave@example.com"})
	if err != nil || token == "" {
		t.Fatalf("ForgotPassword: %v, token=%q", err, token)
	}

	if err := uc.ResetPassword(context.Background(), &ResetPasswordRequest{
		Token: token, Password: "newpassword99",
	}); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}

	// Old token should be consumed.
	err = uc.ResetPassword(context.Background(), &ResetPasswordRequest{Token: token, Password: "again"})
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken on reuse, got %v", err)
	}
}

func TestAuthUseCase_ResetPassword_InvalidToken(t *testing.T) {
	uc := newTestAuthUC(&mockAuthUserRepo{users: map[string]*entity.User{}}, newMockCacher(), newMockBlacklist())

	err := uc.ResetPassword(context.Background(), &ResetPasswordRequest{Token: "badtoken", Password: "newpw1234"})
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestAuthUseCase_Refresh_InvalidAfterPasswordReset(t *testing.T) {
	userID := uuid.New()
	repo := &mockAuthUserRepo{users: map[string]*entity.User{
		"eve@example.com": {ID: userID, Email: "eve@example.com", Password: hashPassword(t, "pw")},
	}}
	cacher := newMockCacher()
	uc := newTestAuthUC(repo, cacher, newMockBlacklist())

	// Login to get a refresh token.
	loginResp, err := uc.Login(context.Background(), &LoginRequest{Email: "eve@example.com", Password: "pw"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	// Trigger a password reset — this should bump the password version.
	resetToken, err := uc.ForgotPassword(context.Background(), &ForgotPasswordRequest{Email: "eve@example.com"})
	if err != nil || resetToken == "" {
		t.Fatalf("ForgotPassword: %v, token=%q", err, resetToken)
	}
	if err := uc.ResetPassword(context.Background(), &ResetPasswordRequest{
		Token: resetToken, Password: "newpassword99",
	}); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}

	// The pre-reset refresh token must now be rejected.
	_, err = uc.Refresh(context.Background(), &RefreshRequest{RefreshToken: loginResp.RefreshToken})
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken after password reset, got %v", err)
	}
}

func TestAuthUseCase_Logout_BlacklistsRefreshJTI(t *testing.T) {
	repo := &mockAuthUserRepo{users: map[string]*entity.User{
		"frank@example.com": {ID: uuid.New(), Email: "frank@example.com", Password: hashPassword(t, "pw")},
	}}
	bl := newMockBlacklist()
	uc := newTestAuthUC(repo, newMockCacher(), bl)

	loginResp, _ := uc.Login(context.Background(), &LoginRequest{Email: "frank@example.com", Password: "pw"})

	if err := uc.Logout(context.Background(), loginResp.AccessToken, loginResp.RefreshToken); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	// Both access and refresh JTIs must be blacklisted.
	refreshCfg := &auth.Config{
		Secret:   "test-secret-at-least-32-bytes-long!",
		Issuer:   "wapgo-test",
		Audience: "wapgo-test-api",
		Expiry:   7 * 24 * time.Hour,
	}
	refreshClaims, err := auth.Verify(loginResp.RefreshToken, refreshCfg)
	if err != nil {
		t.Fatalf("Verify refresh token: %v", err)
	}
	revoked, _ := bl.IsRevoked(context.Background(), refreshClaims.ID)
	if !revoked {
		t.Fatal("expected refresh token JTI to be blacklisted after logout")
	}
}
