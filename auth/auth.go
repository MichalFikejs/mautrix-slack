package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	log "maunium.net/go/maulogger/v2"

	"github.com/slack-go/slack"
)

const (
	baseURL = "https://slack.com/api/auth."
)

type Info struct {
	UserEmail string
	UserID    string
	TeamName  string
	TeamID    string
	Token     string
}

type domainResponse struct {
	Okay   bool   `json:"ok"`
	TeamID string `json:"team_id"`
	URL    string `json:"url"`
	SSO    bool   `json:"sso"`
}

type userResponse struct {
	Okay    bool   `json:"ok"`
	Found   bool   `json:"found"`
	CanJoin bool   `json:"can_join"`
	UserID  string `json:"user_id"`
}

type signinResponse struct {
	Okay      bool   `json:"ok"`
	Token     string `json:"token"`
	UserID    string `json:"user"`
	UserEmail string `json:"user_email"`
	TeamID    string `json:"team"`
}

func post(log log.Logger, method string, form url.Values, data interface{}) error {
	resp, err := http.Post(
		baseURL+method,
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(resp.Body)
		log.Debugln("unexpected response", resp.StatusCode, string(buf))

		return fmt.Errorf("Unexpected response, please try again later")
	}

	if err := json.NewDecoder(resp.Body).Decode(data); err != nil {
		log.Debugln("failed to parse json: %v", err)

		return fmt.Errorf("Unexpected response, please try again later")
	}

	return nil
}

func findTeam(log log.Logger, domain string) (string, error) {
	form := url.Values{}
	form.Add("domain", domain)

	var data domainResponse
	err := post(log, "findTeam", form, &data)
	if err != nil {
		return "", err
	}

	if !data.Okay {
		return "", fmt.Errorf("failed to find team")
	}

	if data.SSO {
		return "", fmt.Errorf("teams that use Single Sign On are not currently supported")
	}

	return data.TeamID, nil
}

func findUser(log log.Logger, email, teamID string) (string, error) {
	form := url.Values{}
	form.Add("email", email)
	form.Add("team", teamID)

	var data userResponse
	err := post(log, "findUser", form, &data)
	if err != nil {
		return "", err
	}

	if !data.Okay {
		return "", fmt.Errorf("failed to find user")
	}

	return data.UserID, nil
}

func signin(log log.Logger, userID, teamID, password string) (string, error) {
	form := url.Values{}
	form.Add("user", userID)
	form.Add("team", teamID)
	form.Add("password", password)

	var data signinResponse
	err := post(log, "signin", form, &data)
	if err != nil {
		return "", err
	}

	if !data.Okay {
		return "", fmt.Errorf("incorrect password")
	}

	return data.Token, nil
}

func Login(l log.Logger, email, team, password string) (*Info, error) {
	log := log.Sub("auth")

	teamID, err := findTeam(log, team)
	if err != nil {
		return nil, err
	}

	userID, err := findUser(log, email, teamID)
	if err != nil {
		return nil, err
	}

	token, err := signin(log, userID, teamID, password)
	if err != nil {
		return nil, err
	}

	return &Info{
		UserEmail: email,
		UserID:    userID,
		TeamName:  team,
		TeamID:    teamID,
		Token:     token,
	}, nil
}

func LoginToken(token string) (*Info, error) {
	client := slack.New(token)

	userIdentity, err := client.GetUserIdentity()
	if err != nil {
		return nil, err
	}

	return &Info{
		UserEmail: userIdentity.User.Email,
		UserID:    userIdentity.User.ID,
		TeamName:  userIdentity.Team.Name,
		TeamID:    userIdentity.Team.ID,
		Token:     token,
	}, nil
}
