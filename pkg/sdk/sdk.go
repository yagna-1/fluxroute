package sdk

import (
	"context"

	"github.com/your-org/agent-router/internal/router"
)

// Client is a thin wrapper for SDK ergonomics.
type Client struct {
	engine *router.Engine
}

func NewClient(engine *router.Engine) *Client {
	return &Client{engine: engine}
}

func (c *Client) Run(ctx context.Context, invocations []router.AgentInvocation) []router.AgentResult {
	return c.engine.Run(ctx, invocations)
}
