/*
 * Free Fire player information command for Zyroxmusic.
 * Uses the documented public endpoints of the FreeFire-Api project.
 */

package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	td "github.com/Kanha/Meow"
)

const defaultFreeFireAPIBase = "https://freefire-api-six.vercel.app"

var freeFireUIDPattern = regexp.MustCompile(`^[0-9]{5,20}$`)

func ffhistoryHandler(c *td.Client, m *td.Message) error {
	if isChannel(c, m) {
		return nil
	}

	args := strings.Fields(m.Args())
	if len(args) < 1 || len(args) > 2 {
		_, err := m.ReplyText(c, "Usage: /ffhistory <UID> [region]\nExample: /ffhistory 11959685790 IND", nil)
		return err
	}

	uid := args[0]
	if !freeFireUIDPattern.MatchString(uid) {
		_, err := m.ReplyText(c, "UID must contain 5–20 digits. Example: /ffhistory 11959685790 IND", nil)
		return err
	}

	region := "IND"
	if len(args) == 2 {
		region = strings.ToUpper(args[1])
	}
	if !regexp.MustCompile(`^[A-Z]{2,3}$`).MatchString(region) {
		_, err := m.ReplyText(c, "Invalid region. Example: /ffhistory 11959685790 IND", nil)
		return err
	}

	base := strings.TrimRight(strings.TrimSpace(os.Getenv("FREEFIRE_API_BASE")), "/")
	if base == "" {
		base = defaultFreeFireAPIBase
	}

	_, err := m.ReplyText(c, "🔎 Fetching Free Fire profile and available BR/CS stats…", nil)
	if err != nil {
		return err
	}

	type result struct {
		title string
		body  string
		err   error
	}
	requests := []struct {
		title string
		path  string
		query url.Values
	}{
		{title: "BR ranked stats", path: "/get_player_stats", query: url.Values{"server": {strings.ToLower(region)}, "uid": {uid}, "gamemode": {"br"}, "matchmode": {"RANKED"}}},
		{title: "CS ranked stats", path: "/get_player_stats", query: url.Values{"server": {strings.ToLower(region)}, "uid": {uid}, "gamemode": {"cs"}, "matchmode": {"RANKED"}}},
		{title: "Personal show / profile", path: "/get_player_personal_show", query: url.Values{"server": {strings.ToLower(region)}, "uid": {uid}}},
	}

	results := make(chan result, len(requests))
	for _, req := range requests {
		req := req
		go func() {
			body, fetchErr := fetchFreeFireJSON(base+req.path+"?"+req.query.Encode())
			results <- result{title: req.title, body: body, err: fetchErr}
		}()
	}

	var sections []string
	successes := 0
	for range requests {
		r := <-results
		if r.err != nil {
			sections = append(sections, "❌ "+r.title+": "+r.err.Error())
			continue
		}
		successes++
		sections = append(sections, "📌 "+r.title+"\n"+r.body)
	}

	if successes == 0 {
		_, err = m.ReplyText(c, "Free Fire API se data nahi mil saka. API URL/configuration check karo aur baad mein retry karo.\n\n"+strings.Join(sections, "\n\n"), nil)
		return err
	}

	message := fmt.Sprintf("🔥 FREE FIRE PLAYER DATA\nUID: %s\nRegion: %s\n\n%s\n\nℹ️ This API documents profile and BR/CS statistics, not a complete match-by-match history.", uid, region, strings.Join(sections, "\n\n"))
	if len(message) > 3900 {
		message = message[:3900] + "\n…output shortened"
	}
	_, err = m.ReplyText(c, message, nil)
	return err
}

func fetchFreeFireJSON(endpoint string) (string, error) {
	client := &http.Client{Timeout: 12 * time.Second}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("invalid API URL")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Zyroxmusic-FreeFireLookup/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return "", fmt.Errorf("could not read API response")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("API returned HTTP %d", resp.StatusCode)
	}

	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("API returned invalid JSON")
	}
	formatted, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("could not format API response")
	}
	text := string(formatted)
	if len(text) > 1000 {
		text = text[:1000] + "\n…section shortened"
	}
	return text, nil
}
