package client

// TestAuth validates the API key against the server
func (c *Client) TestAuth() error {
	resp, err := c.get("/api/v1/auth/test")
	if err != nil {
		return err
	}
	_, err = decodeJSON(resp)
	return err
}
