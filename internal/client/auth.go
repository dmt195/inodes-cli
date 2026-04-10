package client

import (
	"encoding/json"
	"fmt"
)

// MeUser is the user identity returned by GET /api/v1/me.
type MeUser struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// MeTeam is the (optional) active team returned by GET /api/v1/me. It is
// nil for personal API keys and set for team-scoped keys.
type MeTeam struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// MeResponse is the decoded payload of GET /api/v1/me.
type MeResponse struct {
	User MeUser  `json:"user"`
	Team *MeTeam `json:"team"`
}

// DisplayName returns a human-readable identity for logging/UI output.
// Falls back gracefully when first/last name are empty.
func (m *MeResponse) DisplayName() string {
	switch {
	case m.User.FirstName != "" && m.User.LastName != "":
		return m.User.FirstName + " " + m.User.LastName
	case m.User.FirstName != "":
		return m.User.FirstName
	case m.User.LastName != "":
		return m.User.LastName
	default:
		return m.User.Email
	}
}

// TestAuth validates the API key against the server and returns the
// authenticated user (and active team, if any).
//
// Historically this hit /api/v1/auth/test, but that endpoint is shadowed
// by the SPA /api/v1/auth subrouter in Chi's radix tree and returns a
// plain-text 404 in production. The replacement is /api/v1/me, which
// lives outside the /auth/* namespace and additionally returns the
// authenticated user's identity so the CLI can show a friendly "Logged
// in as Alice @ Acme Team" confirmation on `inodes configure`.
func (c *Client) TestAuth() (*MeResponse, error) {
	resp, err := c.get("/api/v1/me")
	if err != nil {
		return nil, err
	}
	data, err := decodeJSON(resp)
	if err != nil {
		return nil, err
	}
	var me MeResponse
	if len(data) > 0 && string(data) != "null" {
		if err := json.Unmarshal(data, &me); err != nil {
			return nil, fmt.Errorf("decoding /api/v1/me response: %w", err)
		}
	}
	return &me, nil
}
