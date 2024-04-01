package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"

	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	postgresUser            string
	postgresPassword        string
	postgresHost            string
	postgresPort            string
	postgresDB              string
	dbInfo                  string
	googleOauthClientID     string
	googleOauthClientSecret string
	googleOauthConfig       *oauth2.Config
)

func init() {
	envFile, err := godotenv.Read(".env")
	if err != nil {
		log.Fatalf("Couldn't load the .env: %v", err)
	}
	googleOauthClientID = envFile["GoogleOauthClientID"]
	googleOauthClientSecret = envFile["GoogleOauthClientSecret"]

	postgresUser = envFile["POSTGRES_USER"]
	postgresPassword = envFile["POSTGRES_PASSWORD"]
	postgresHost = envFile["POSTGRES_HOST"]
	postgresPort = envFile["POSTGRES_PORT"]
	postgresDB = envFile["POSTGRES_DB"]

	dbInfo = fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		postgresHost, postgresPort, postgresUser, postgresPassword, postgresDB,
	)

	googleOauthConfig = &oauth2.Config{
		ClientID:     googleOauthClientID,
		ClientSecret: googleOauthClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost:8080/callback",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}
}

func getDB() *sql.DB {
	log.Println(dbInfo)
	db, err := sql.Open("postgres", dbInfo)
	if err != nil {
		log.Fatalf("Couldn't connect to database: %v", err)
	}
	return db
}

type UserInfo struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Profile       string `json:"profile"`
	Picture       string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Gender        string `json:"gender"`
}

func FetchUserInfo(client *http.Client) (*UserInfo, error) {
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result UserInfo
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func main() {
	db := getDB()
	defer db.Close()
	log.Println(googleOauthClientID, googleOauthClientSecret, db)
	sql_file, err := os.ReadFile("users.sql")
	if err != nil {
		log.Fatalf("Couldn't read the sql schema file: %v", err)
	}
	log.Println(string(sql_file))
	_, err = db.Exec(string(sql_file))
	if err != nil {
		log.Fatalf("Sql query error: %v", err)
	}
	router := gin.Default()
	router.LoadHTMLGlob("./*.html")
	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "home.html", gin.H{
			"msg": "home page",
		})
	})

	router.GET("/login", func(c *gin.Context) {
		state := fmt.Sprint(rand.Int())
		code_varifier := oauth2.GenerateVerifier()
		c.SetCookie("google_oauth_state", state, 60*5, "/", "localhost", false, true)
		c.SetCookie("google_oauth_code_varifier", code_varifier, 60*5, "/", "localhost", false, true)
		c.Redirect(
			http.StatusTemporaryRedirect,
			googleOauthConfig.AuthCodeURL(state, oauth2.S256ChallengeOption(code_varifier)),
		)
	})

	router.GET("/callback", func(c *gin.Context) {
		oauthState, err := c.Cookie("google_oauth_state")
		stateParam := c.Query("state")
		if err != nil || oauthState != stateParam {
			c.JSON(200, gin.H{
				"msg": "state mismatch",
			})
			return
		}
		code_varifier, err := c.Cookie("google_oauth_code_varifier")
		if err != nil {
			c.JSON(200, gin.H{
				"msg": "code varifier not found",
			})
			return
		}
		token, err := googleOauthConfig.Exchange(
			context.Background(),
			c.Query("code"),
			oauth2.VerifierOption(code_varifier),
		)
		if err != nil {
			c.JSON(200, gin.H{
				"msg": "code exchange failed",
			})
			return
		}
		client := googleOauthConfig.Client(context.Background(), token)
		info, err := FetchUserInfo(client)
		if err != nil {
			c.JSON(200, gin.H{
				"msg": "Couldn't get user info",
			})
			return
		}
		c.JSON(200, info)
	})

	router.Run("localhost:8080")
}
