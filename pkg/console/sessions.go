package console

import (
	"strconv"

	"github.com/Chocapikk/pik/pkg/c2"
	"github.com/Chocapikk/pik/pkg/log"
	"github.com/Chocapikk/pik/pkg/output"
)

func (c *Console) cmdSessions(args []string) {
	handler := c.sessionHandler()
	if handler == nil {
		output.Warning("No active listener with session support")
		return
	}

	if len(args) > 0 {
		id, ok := c.parseSessionID(args[0])
		if !ok {
			return
		}
		if c.program != nil {
			go c.program.Send(SessionInteractMsg{ID: id})
		}
		return
	}

	sessions := handler.Sessions()
	if len(sessions) == 0 {
		output.Warning("No active sessions")
		return
	}

	output.Println()
	output.Print("  %s  %s  %s\n",
		log.Pad(log.UnderlineText("ID"), 6),
		log.Pad(log.UnderlineText("Remote Address"), 25),
		log.UnderlineText("Opened"),
	)
	for _, sess := range sessions {
		output.Print("  %s  %s  %s\n",
			log.Pad(log.Cyan(strconv.Itoa(sess.ID)), 6),
			log.Pad(log.White(sess.RemoteAddr), 25),
			log.Gray(sess.CreatedAt.Format("15:04:05")),
		)
	}
	output.Println()
}

func (c *Console) cmdKill(args []string) {
	if len(args) == 0 {
		output.Error("Usage: kill <session_id>")
		return
	}

	handler := c.sessionHandler()
	if handler == nil {
		output.Warning("No active listener with session support")
		return
	}

	id, ok := c.parseSessionID(args[0])
	if !ok {
		return
	}
	if err := handler.Kill(id); err != nil {
		output.Error("%v", err)
	}
}

func (c *Console) parseSessionID(raw string) (int, bool) {
	id, err := strconv.Atoi(raw)
	if err != nil {
		output.Error("Invalid session ID: %s", raw)
		return 0, false
	}
	return id, true
}

func (c *Console) shutdownBackend() {
	if c.activeBackend != nil {
		_ = c.activeBackend.Shutdown()
		c.activeBackend = nil
	}
}

func (c *Console) sessionHandler() c2.SessionHandler {
	if c.activeBackend == nil {
		return nil
	}
	handler, ok := c.activeBackend.(c2.SessionHandler)
	if !ok {
		return nil
	}
	return handler
}
