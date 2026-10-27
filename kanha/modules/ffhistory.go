package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type FFPlayerResponse struct {
	BasicInfo struct {
		Nickname    string      `json:"nickname"`
		AccountID   string      `json:"accountId"`
		Region      string      `json:"region"`
		Level       int         `json:"level"`
		Exp         int         `json:"exp"`
		Liked       int         `json:"liked"`
		CreateAt    interface{} `json:"createAt"`
		LastLoginAt interface{} `json:"lastLoginAt"`
	} `json:"basicInfo"`
	ClanBasicInfo struct {
		ClanName  string `json:"clanName"`
		ClanLevel int    `json:"clanLevel"`
	} `json:"clanBasicInfo"`
	SocialInfo struct {
		Signature string `json:"signature"`
	} `json:"socialInfo"`
}

func formatFFTime(ts interface{}) string {
	var sec int64
	switch v := ts.(type) {
	case float64:
		sec = int64(v)
	case string:
		sec, _ = strconv.ParseInt(v, 10, 64)
	default:
		return "N/A"
	}
	if sec == 0 {
		return "N/A"
	}
	return time.Unix(sec, 0).Format("02 Jan 2006, 03:04 PM")
}

// FetchFFHistory function: UID se Free Fire Player ki history nikalne ke liye
func FetchFFHistory(input string) string {
	args := strings.Fields(input)
	if len(args) == 0 {
		return "❌ **UID dalein!**\nExample: `/ffhistory 123456789`"
	}

	uid := args[0]
	region := "ind"
	if len(args) > 1 {
		region = strings.ToLower(args[1])
	}

	apiURL := fmt.Sprintf("https://free-ff-api-src-5plp.onrender.com/api/v1/account?region=%s&uid=%s", region, uid)

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "⚠️ **Error:** Free Fire API server abhi busy hai, thodi der baad try karein."
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var data FFPlayerResponse
	if err := json.Unmarshal(body, &data); err != nil || data.BasicInfo.Nickname == "" {
		return "❌ **Error:** Player nahi mila! Kripya sahi UID check karein."
	}

	createdDate := formatFFTime(data.BasicInfo.CreateAt)
	lastLogin := formatFFTime(data.BasicInfo.LastLoginAt)
	guild := data.ClanBasicInfo.ClanName
	if guild == "" {
		guild = "No Guild"
	}

	return fmt.Sprintf(
		"🔥 **FREE FIRE PLAYER HISTORY** 🔥\n"+
			"━━━━━━━━━━━━━━━━━━━━\n"+
			"👤 **Name:** `%s`\n"+
			"🆔 **UID:** `%s` (%s)\n"+
			"⭐ **Level:** `%d` (EXP: `%d`)\n"+
			"👍 **Likes:** `%d`\n"+
			"🛡️ **Guild:** `%s` (Lv. %d)\n"+
			"📅 **Account Created:** `%s`\n"+
			"🕒 **Last Login:** `%s`\n"+
			"📝 **Bio:** `%s`",
		data.BasicInfo.Nickname,
		uid,
		strings.ToUpper(region),
		data.BasicInfo.Level,
		data.BasicInfo.Exp,
		data.BasicInfo.Liked,
		guild,
		data.ClanBasicInfo.ClanLevel,
		createdDate,
		lastLogin,
		data.SocialInfo.Signature,
	)
}
