package jwt

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var signingMethod = jwt.SigningMethodHS256

type Identity struct {
	ID       int
	Email    string
	FullName string
	Role     string
}

type Claims struct {
	Email    string `json:"email"`
	FullName string `json:"name"`
	Role     string `json:"role"`

	jwt.RegisteredClaims
}

type Issuer struct {
	secret []byte
	ttl    time.Duration
}

type IssuerOptions struct {
	Secret string
	TTL    time.Duration
}

func NewIssuer(opts *IssuerOptions) (*Issuer, error) {
	if opts == nil {
		return nil, fmt.Errorf("IssuerOptions cannot be nil")
	}
	if opts.Secret == "" {
		return nil, fmt.Errorf("IssuerOptions.Secret cannot be empty")
	}
	if opts.TTL <= 0 {
		return nil, fmt.Errorf("IssuerOptions.TTL must be positive")
	}

	return &Issuer{secret: []byte(opts.Secret), ttl: opts.TTL}, nil
}

func (ti *Issuer) Issue(identity Identity) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(ti.ttl)

	claims := Claims{
		Email:    identity.Email,
		FullName: identity.FullName,
		Role:     identity.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.Itoa(identity.ID),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	signed, err := jwt.NewWithClaims(signingMethod, claims).SignedString(ti.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return signed, expiresAt, nil
}

func (ti *Issuer) Verify(raw string) (*Claims, error) {
	claims := new(Claims)

	_, err := jwt.ParseWithClaims(raw, claims,
		func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != signingMethod.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}

			return ti.secret, nil
		},
		jwt.WithValidMethods([]string{signingMethod.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	return claims, nil
}

func (c *Claims) UserID() (int, error) {
	id, err := strconv.Atoi(c.Subject)
	if err != nil {
		return 0, fmt.Errorf("subject %q is not a valid user id: %w", c.Subject, err)
	}

	return id, nil
}
