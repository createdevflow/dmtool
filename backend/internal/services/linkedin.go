package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"backend/internal/models"
)

type LinkedinService interface {
	FetchLinkedInMetrics(projectID uint, handle, accessToken string) (*models.SocialMetric, error)
	PublishLinkedInContent(handle, accessToken string, task *models.Task) (string, error)
}

type linkedinService struct {
	client *http.Client
}

func NewLinkedinService() LinkedinService {
	return &linkedinService{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *linkedinService) resolveProfile(accessToken string) (id, displayName, picture string, err error) {
	if accessToken == "" {
		return "", "", "", fmt.Errorf("linkedin access token is required")
	}

	infoReq, _ := http.NewRequest("GET", "https://api.linkedin.com/v2/userinfo", nil)
	infoReq.Header.Set("Authorization", "Bearer "+accessToken)

	infoResp, err := s.client.Do(infoReq)
	if err == nil {
		defer infoResp.Body.Close()
		bodyBytes, _ := io.ReadAll(infoResp.Body)
		
		if infoResp.StatusCode == http.StatusOK {
			var ui struct {
				Sub        string `json:"sub"`
				Name       string `json:"name"`
				GivenName  string `json:"given_name"`
				FamilyName string `json:"family_name"`
				Picture    string `json:"picture"`
			}
			if err := json.Unmarshal(bodyBytes, &ui); err == nil && ui.Sub != "" {
				displayName = ui.Name
				if displayName == "" {
					displayName = strings.TrimSpace(ui.GivenName + " " + ui.FamilyName)
				}
				return ui.Sub, displayName, ui.Picture, nil
			}
		} else {
			fmt.Printf("[LinkedIn] userinfo API returned status %d\n", infoResp.StatusCode)
		}
	} else {
		fmt.Printf("[LinkedIn] userinfo call failed: %v\n", err)
	}

	meReq, _ := http.NewRequest("GET", "https://api.linkedin.com/v2/me?projection=(id,localizedFirstName,localizedLastName,profilePicture(displayImage~:playableStreams))", nil)
	meReq.Header.Set("Authorization", "Bearer "+accessToken)

	meResp, err := s.client.Do(meReq)
	if err != nil {
		fmt.Printf("[LinkedIn] /v2/me call error: %v\n", err)
		return "", "", "", err
	}
	defer meResp.Body.Close()

	bodyBytes, _ := io.ReadAll(meResp.Body)

	if meResp.StatusCode != http.StatusOK {
		fmt.Printf("[LinkedIn] /v2/me error status %d\n", meResp.StatusCode)
		return "", "", "", fmt.Errorf("linkedin api error: status %d", meResp.StatusCode)
	}

	var me struct {
		ID                 string `json:"id"`
		LocalizedFirstName string `json:"localizedFirstName"`
		LocalizedLastName  string `json:"localizedLastName"`
		ProfilePicture     struct {
			DisplayImage struct {
				Elements []struct {
					Identifiers []struct {
						Identifier string `json:"identifier"`
					} `json:"identifiers"`
				} `json:"elements"`
			} `json:"displayImage~"`
		} `json:"profilePicture"`
	}
	if err := json.Unmarshal(bodyBytes, &me); err != nil {
		fmt.Printf("[LinkedIn] /v2/me parse error: %v\n", err)
		return "", "", "", err
	}

	displayName = strings.TrimSpace(me.LocalizedFirstName + " " + me.LocalizedLastName)
	if displayName == "" {
		displayName = me.ID
	}
	if len(me.ProfilePicture.DisplayImage.Elements) > 0 && len(me.ProfilePicture.DisplayImage.Elements[0].Identifiers) > 0 {
		picture = me.ProfilePicture.DisplayImage.Elements[0].Identifiers[0].Identifier
	}

	return me.ID, displayName, picture, nil
}

// FetchLinkedInMetrics gets the user profile and basic stats
func (s *linkedinService) FetchLinkedInMetrics(projectID uint, handle, accessToken string) (*models.SocialMetric, error) {
	id, name, picture, err := s.resolveProfile(accessToken)
	if err != nil {
		return nil, err
	}

	website := ""
	if handle != "" {
		if strings.HasPrefix(handle, "http") {
			website = handle
		} else {
			website = "https://linkedin.com/in/" + handle
		}
	} else if id != "" {
		website = "https://www.linkedin.com/in/" + id
	}

	sm := &models.SocialMetric{
		ProjectID:         projectID,
		Platform:          "LinkedIn",
		Followers:         0,
		FollowingCount:    0,
		PostsCount:        0,
		Reach:             0,
		Engagement:        0,
		DisplayName:       name,
		ProfilePictureURL: picture,
		Biography:         "LinkedIn Profile",
		Website:           website,
		Status:            "stable",
		IsSimulated:       false,
		RecordedAt:        time.Now(),
	}

	return sm, nil
}

// PublishLinkedInContent publishes a text or image post to LinkedIn
func (s *linkedinService) PublishLinkedInContent(handle, accessToken string, task *models.Task) (string, error) {
	if accessToken == "" {
		return "", fmt.Errorf("linkedin access token is required")
	}

	// 1. Get user URN
	id, _, _, err := s.resolveProfile(accessToken)
	if err != nil {
		return "", err
	}
	if id == "" {
		return "", fmt.Errorf("failed to resolve linkedin author URN")
	}

	authorURN := "urn:li:person:" + id

	// 2. Build the ugcPost payload
	// For simplicity in this iteration, we post purely text if no image logic is fully mapped,
	// but we can map the text caption.

	caption := task.Caption
	if caption == "" {
		caption = task.Title
	}
	if task.Tags != "" {
		caption = caption + "\n\n" + task.Tags
	}

	payload := map[string]interface{}{
		"author":         authorURN,
		"lifecycleState": "PUBLISHED",
		"specificContent": map[string]interface{}{
			"com.linkedin.ugc.ShareContent": map[string]interface{}{
				"shareCommentary": map[string]interface{}{
					"text": caption,
				},
				"shareMediaCategory": "NONE",
			},
		},
		"visibility": map[string]interface{}{
			"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC",
		},
	}

	// If there's an image, we need to register upload, upload image, then post.
	// This requires complex 3-step upload. For MVP, we stick to text post or article link if assetUrl exists.
	if task.AssetURL != "" {
		// Just append link to text
		payload["specificContent"].(map[string]interface{})["com.linkedin.ugc.ShareContent"].(map[string]interface{})["shareCommentary"].(map[string]interface{})["text"] = caption + "\n\n" + task.AssetURL
	}

	jsonPayload, _ := json.Marshal(payload)
	reqPost, _ := http.NewRequest("POST", "https://api.linkedin.com/v2/ugcPosts", bytes.NewBuffer(jsonPayload))
	reqPost.Header.Set("Authorization", "Bearer "+accessToken)
	reqPost.Header.Set("X-Restli-Protocol-Version", "2.0.0")
	reqPost.Header.Set("Content-Type", "application/json")

	respPost, err := s.client.Do(reqPost)
	if err != nil {
		return "", err
	}
	defer respPost.Body.Close()

	bodyPost, _ := io.ReadAll(respPost.Body)
	if respPost.StatusCode != http.StatusCreated && respPost.StatusCode != http.StatusOK {
		return "", fmt.Errorf("linkedin post error: %s", string(bodyPost))
	}

	var result struct {
		ID string `json:"id"`
	}
	json.Unmarshal(bodyPost, &result)

	if result.ID == "" {
		return "published", nil // Fallback if no ID is returned
	}
	return result.ID, nil
}
