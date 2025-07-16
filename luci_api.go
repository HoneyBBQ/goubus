package goubus

import "time"

type LuciManager struct {
	client *Client
}

func (c *Client) Luci() *LuciManager {
	return &LuciManager{
		client: c,
	}
}

func (c *LuciManager) GetLocalTime() (time.Time, error) {
	return c.client.luciGetTime()
}

func (c *LuciManager) SetLocalTime(time time.Time) error {
	return c.client.luciSetTime(time)
}
