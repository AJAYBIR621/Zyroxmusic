/*
 * DM welcome auto-reply: reuse the existing /start welcome content, image,
 * and Add Me keyboard when a user mentions the bot or replies in a private chat.
 */
package modules

import (
	"strings"

	td "github.com/Kanha/Meow"

	"KanhaMusic/kanha/database"
)

func dmAutoReplyFilter(m *td.Message) bool {
	if m == nil || !m.IsPrivate() || m.IsOutgoingMessage() || m.IsCommand() {
		return false
	}
	// Only inspect messages that may be a mention or a reply.
	return m.ReplyToMessageID() != 0 || strings.Contains(m.GetText(), "@")
}

func isBotMention(c *td.Client, text string) bool {
	if c == nil || c.Me == nil || c.Me.Usernames == nil {
		return false
	}
	text = strings.ToLower(strings.TrimSpace(text))
	for _, part := range strings.Fields(text) {
		part = strings.Trim(part, "()[]{}<>\"'!,.:;?")
		if !strings.HasPrefix(part, "@") {
			continue
		}
		username := strings.TrimPrefix(part, "@")
		for _, active := range c.Me.Usernames.ActiveUsernames {
			if active != "" && strings.EqualFold(username, active) {
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
	// Replies repeat the welcome message. Mentions only trigger when they
	// match one of this bot's active Telegram usernames.
	if m.ReplyToMessageID() == 0 && !isBotMention(c, m.GetText()) {
		return nil
	}

	database.AddServedUser(m.ChatID())
	sender, _ := m.GetUser(c)
	return handlePmStart(c, m, sender)
}
