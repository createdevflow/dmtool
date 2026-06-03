package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"



	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type OAuthCredential struct {
	ProjectID      uint   `gorm:"column:project_id"`
	Provider       string `gorm:"column:provider"`
	AccessTokenEnc string `gorm:"column:access_token_enc"`
}

func (OAuthCredential) TableName() string {
	return "o_auth_credentials"
}

type Project struct {
	ID       uint   `gorm:"primaryKey"`
	IGHandle string `gorm:"column:ig_handle"`
	FBHandle string `gorm:"column:fb_handle"`
}

func (Project) TableName() string {
	return "projects"
}

func main() {
	db, err := gorm.Open(sqlite.Open("dmtool.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey == "" {
		log.Fatal("ENCRYPTION_KEY not set")
	}

	fmt.Println("Starting Meta API Audit...")

	// Audit Project 11 (Instagram)
	auditInstagram(db, 11, os.Getenv("META_PAGE_ACCESS_TOKEN"))

	// Audit Project 12 (Facebook)
	auditFacebook(db, 12, os.Getenv("META_PAGE_ACCESS_TOKEN"))
}

func auditInstagram(db *gorm.DB, projectID uint, token string) {
	fmt.Printf("\n--- Auditing Instagram (Project %d) ---\n", projectID)
	var proj Project
	if err := db.Table("projects").Where("id = ?", projectID).First(&proj).Error; err != nil {
		fmt.Println("Error fetching project:", err)
		return
	}

	igUserID := proj.IGHandle // Actually in dmtool ig_handle stores the ig_user_id

	fmt.Println("IG User ID (Handle from DB):", igUserID)

	// Fetch Numeric ID using the token
	numericIGID := ""
	pageToken := token
	respAccounts, err := http.Get(fmt.Sprintf("https://graph.facebook.com/v19.0/me/accounts?access_token=%s", token))
	if err == nil {
		defer respAccounts.Body.Close()
		var data struct {
			Data []struct {
				ID          string `json:"id"`
				AccessToken string `json:"access_token"`
			} `json:"data"`
		}
		json.NewDecoder(respAccounts.Body).Decode(&data)
		if len(data.Data) > 0 {
			pageToken = data.Data[0].AccessToken
			pageID := data.Data[0].ID
			respIG, err := http.Get(fmt.Sprintf("https://graph.facebook.com/v19.0/%s?fields=instagram_business_account&access_token=%s", pageID, pageToken))
			if err == nil {
				defer respIG.Body.Close()
				var igData struct {
					InstagramBusinessAccount struct {
						ID string `json:"id"`
					} `json:"instagram_business_account"`
				}
				json.NewDecoder(respIG.Body).Decode(&igData)
				numericIGID = igData.InstagramBusinessAccount.ID
			}
		}
	}

	if numericIGID != "" {
		igUserID = numericIGID
		fmt.Println("IG User ID (Numeric Resolved):", igUserID)
	}

	// 1. Profile Info
	testEndpoint(pageToken, fmt.Sprintf("https://graph.facebook.com/v19.0/%s?fields=followers_count,follows_count,media_count,biography,website,name,username,profile_picture_url", igUserID), "Profile Information")

	// 2. Reach, Views, Engagement (Insights)
	testEndpoint(pageToken, fmt.Sprintf("https://graph.facebook.com/v19.0/%s/insights?metric=impressions,reach,profile_views&period=day", igUserID), "Basic Insights (Reach/Impressions)")

	// 3. Audience Insights
	testEndpoint(pageToken, fmt.Sprintf("https://graph.facebook.com/v19.0/%s/insights?metric=audience_city,audience_country,audience_gender_age&period=lifetime", igUserID), "Audience Insights")

	// 4. Content Performance
	testEndpoint(pageToken, fmt.Sprintf("https://graph.facebook.com/v19.0/%s/media?fields=id,caption,media_type,media_url,like_count,comments_count,timestamp", igUserID), "Content Performance")
}

func auditFacebook(db *gorm.DB, projectID uint, token string) {
	fmt.Printf("\n--- Auditing Facebook (Project %d) ---\n", projectID)

	// Need to get page ID (which might be in another table or we can get it from /me/accounts)
	pageID := ""
	pageToken := token
	resp, err := http.Get(fmt.Sprintf("https://graph.facebook.com/v19.0/me/accounts?access_token=%s", token))
	if err == nil {
		defer resp.Body.Close()
		var data struct {
			Data []struct {
				ID          string `json:"id"`
				AccessToken string `json:"access_token"`
			} `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&data)
		if len(data.Data) > 0 {
			pageID = data.Data[0].ID
			pageToken = data.Data[0].AccessToken
		}
	}
	fmt.Println("Page ID:", pageID)

	// 1. Profile Info
	testEndpoint(pageToken, fmt.Sprintf("https://graph.facebook.com/v19.0/%s?fields=fan_count,followers_count,name,username,about,link,picture,description", pageID), "Profile Information")

	// 2. Insights
	testEndpoint(pageToken, fmt.Sprintf("https://graph.facebook.com/v19.0/%s/insights?metric=page_impressions,page_impressions_unique,page_post_engagements,page_fans&period=day", pageID), "Basic Insights (Reach/Engagement)")

	// 3. Posts
	testEndpoint(pageToken, fmt.Sprintf("https://graph.facebook.com/v19.0/%s/posts?fields=id,message,created_time,shares,comments.summary(true),likes.summary(true)", pageID), "Content Performance (Posts)")
}

func testEndpoint(token, urlStr, description string) {
	fmt.Printf("\n[TEST] %s\n", description)
	reqURL := urlStr
	if token != "" {
		reqURL += "&access_token=" + token
	}
	resp, err := http.Get(reqURL)
	if err != nil {
		fmt.Println("HTTP Error:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("FAILED (Status %d): %s\n", resp.StatusCode, string(body))
	} else {
		// print truncated success
		str := string(body)
		if len(str) > 300 {
			str = str[:300] + "..."
		}
		fmt.Printf("SUCCESS: %s\n", str)
	}
}
