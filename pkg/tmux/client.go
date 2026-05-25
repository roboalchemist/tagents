package tmux

type Client struct {
	executor Executor
}

type Executor interface {
	Run(args ...string) (string, error)
}

func NewClient(exec Executor) *Client {
	return &Client{executor: exec}
}

func (c *Client) ListSessions() ([]Session, error) {
	return nil, nil
}

func (c *Client) CapturePane(session string, lines int) (string, error) {
	return "", nil
}

func (c *Client) SendKeys(session, message string) error {
	return nil
}

func (c *Client) GetPaneCWD(session string) (string, error) {
	return "", nil
}

func (c *Client) SessionExists(name string) bool {
	return false
}
