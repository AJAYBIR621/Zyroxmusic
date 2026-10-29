/*
 * DM welcome auto-reply: reuse the existing /start welcome content, image,
 * and Add Me keyboard when a user mentions the bot or replies in a private chat.
 */
package modules

import (
	"strings"

	td "github.com/Kanha/Meow"

	"KanhaMusic/kanha/core"
	"KanhaMusic/kanha/database"
)

func dmAutoReplyFilter(m *td.Message) bool {
	if m == nil || !m.IsPrivate() || m.IsOutgoingMessage() || m.IsCommand() {
		return false
	}
	if m.ReplyToMessageID() != 0 {
		return true
	}

	text := strings.ToLower(strings.TrimSpace(m.GetText()))
	if text == "" || core.Bot == nil || core.Bot.Me == nil || core.Bot.Me.Usernames == nil {
		return false
	}

	for _, part := range strings.Fields(text) {
		part = strings.Trim(part, "()[]{}<>\"'!,.:;?")
		if !strings.HasPrefix(part, "@") {
			continue
		}
		username := strings.TrimPrefix(part, "@")
		for _, active := range core.Bot.Me.Usernames.ActiveUsernames {
			if strings.EqualFold(username, active) && active != "" {
				return true
			}
		}
	}
	return false
}

func dmAutoReplyHandler(c *td.Client, m *td.Message) error {
	if c == nil || c.Me == nil || m == nil {
		return nil
	}
	database.AddServedUser(m.ChatID())
	sender, _ := m.GetUser(c)
	return handlePmStart(c, m, sender)
}
