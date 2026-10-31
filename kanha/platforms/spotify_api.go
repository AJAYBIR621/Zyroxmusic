package platforms

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"KanhaMusic/config"
	state "KanhaMusic/kanha/core/models"
)

var spotifyTokenState struct {
	sync.Mutex
	value   string
	expires time.Time
}

type spotifyAPITrack struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DurationMS  int    `json:"duration_ms"`
	ExternalURL struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
	Artists []struct {
		Name string `json:"name"`
	} `json:"artists"`
	Album struct {
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
	} `json:"album"`
}

type spotifyAPIPage struct {
	Items []json.RawMessage `json:"items"`
}

func spotifyAccessToken() (string, error) {
	if config.SpotifyClientID == "" || config.SpotifyClientSecret == "" {
		return "", errors.New("SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET are not configured")
	}

	spotifyTokenState.Lock()
	defer spotifyTokenState.Unlock()
	if spotifyTokenState.value != "" && time.Now().Add(30*time.Second).Before(spotifyTokenState.expires) {
		return spotifyTokenState.value, nil
	}

	form := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequest(http.MethodPost, "https://accounts.spotify.com/api/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(config.SpotifyClientID, config.SpotifyClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := spotifyHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned HTTP %d", resp.StatusCode)
	}

	var token struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "", err
	}
	if token.AccessToken == "" {
		return "", errors.New("Spotify returned an empty access token")
	}
	spotifyTokenState.value = token.AccessToken
	spotifyTokenState.expires = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return spotifyTokenState.value, nil
}

func spotifyAPIGetTracks(kind, id string) ([]*state.Track, error) {
	token, err := spotifyAccessToken()
	if err != nil {
		return nil, err
	}

	endpoint := "https://api.spotify.com/v1/tracks/" + id
	switch kind {
	case "album":
		endpoint = "https://api.spotify.com/v1/albums/" + id + "/tracks?limit=50"
	case "playlist":
		endpoint = "https://api.spotify.com/v1/playlists/" + id + "/tracks?limit=50"
	case "track":
	default:
		return nil, fmt.Errorf("unsupported Spotify item type %q", kind)
	}

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := spotifyHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Spotify API returned HTTP %d", resp.StatusCode)
	}

	limit := config.QueueLimit
	if limit <= 0 {
		limit = 15
	}
	tracks := make([]*state.Track, 0, limit)

	if kind == "track" {
		var item spotifyAPITrack
		if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
			return nil, err
		}
		if t := spotifyAPITrackToTrack(item); t != nil {
			tracks = append(tracks, t)
		}
		return tracks, nil
	}

	var page spotifyAPIPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}
	for _, raw := range page.Items {
		var item spotifyAPITrack
		if kind == "playlist" {
			var wrapper struct {
				Track *spotifyAPITrack `json:"track"`
			}
			if err := json.Unmarshal(raw, &wrapper); err != nil || wrapper.Track == nil {
				continue
			}
			item = *wrapper.Track
		} else if err := json.Unmarshal(raw, &item); err != nil {
			continue
		}
		if t := spotifyAPITrackToTrack(item); t != nil {
			tracks = append(tracks, t)
		}
		if len(tracks) >= limit {
			break
		}
	}
	return tracks, nil
}

func spotifyAPITrackToTrack(item spotifyAPITrack) *state.Track {
	if item.ID == "" || strings.TrimSpace(item.Name) == "" {
		return nil
	}
	artists := make([]string, 0, len(item.Artists))
	for _, artist := range item.Artists {
		if strings.TrimSpace(artist.Name) != "" {
			artists = append(artists, artist.Name)
		}
	}
	trackURL := item.ExternalURL.Spotify
	if trackURL == "" {
		trackURL = "https://open.spotify.com/track/" + item.ID
	}
	artwork := ""
	if len(item.Album.Images) > 0 {
		artwork = item.Album.Images[0].URL
	}
	return &state.Track{
		ID:       item.ID,
		Title:    spotifyTitle(item.Name, strings.Join(artists, ", ")),
		Duration: item.DurationMS / 1000,
		Artwork:  artwork,
		URL:      trackURL,
		Source:   PlatformSpotify,
	}
}
