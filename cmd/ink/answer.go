package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mkubasz/agentink/internal/inbox/domain"
	"github.com/mkubasz/agentink/internal/platform/api"
	"github.com/mkubasz/agentink/internal/platform/client"
)

// exitPending says the question is known but the owner has not answered yet:
// the agent ends its turn and comes back when its orchestrator says so.
const exitPending = 4

// errPending carries the pending status up to exit().
var errPending = errors.New("no answer yet")

// answer prints where a question stands and, once confirmed, the owner's
// words. It never waits (brief §6.1).
func (c *cli) answer(args []string) error {
	if len(args) != 1 {
		return usageError{"answer takes one argument: the question id"}
	}

	ctx, cancel := c.ctx()
	defer cancel()

	q, err := c.client.Question(ctx, args[0])
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.Status == 404 {
			return fmt.Errorf("unknown question %s", args[0])
		}

		return err //nolint:wrapcheck // the client's errors are already named
	}

	if perr := c.print(q, func() string { return formatQuestion(q) }); perr != nil {
		return perr
	}

	if !answered(q.Status) {
		return errPending
	}

	return nil
}

func answered(status string) bool {
	return status == domain.StatusConfirmed || status == domain.StatusDelivered
}

func formatQuestion(q api.Question) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s · %s · %s · %s\n", q.QuestionID, q.Project, q.Kind, q.Status)
	fmt.Fprintf(&b, "%s\n", q.Title)

	if answered(q.Status) {
		fmt.Fprintf(&b, "answer: %s\n", q.Words)
	} else {
		b.WriteString("no answer yet: end your turn; the orchestrator resumes you when the owner has written\n")
	}

	return b.String()
}
