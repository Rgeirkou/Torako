package model

import "time"

type ApiKey struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Rank         string     `json:"rank"`
	KeyHash      string     `json:"-"`
	Scopes       []string   `json:"scopes"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
	RequestCount int64      `json:"request_count"`
	CreatedAt    time.Time  `json:"created_at"`
}

type CreateApiKeyInput struct {
	Name      string     `json:"name"`
	Rank      string     `json:"rank"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type ApiKeyView struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Rank         string     `json:"rank"`
	Key          string     `json:"key,omitempty"`
	Scopes       []string   `json:"scopes"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
	RequestCount int64      `json:"request_count"`
	CreatedAt    time.Time  `json:"created_at"`
}
