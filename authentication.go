package jira

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// AuthenticationService handles authentication for the JIRA instance / API.
//
// JIRA API docs: https://docs.atlassian.com/jira/REST/latest/#authentication
type AuthenticationService struct {
	client *Client
}

// Session represents a Session JSON response by the JIRA API.
type Session struct {
	Self    string `json:"self,omitempty"`
	Name    string `json:"name,omitempty"`
	Session struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"session,omitempty"`
	LoginInfo struct {
		FailedLoginCount    int    `json:"failedLoginCount"`
		LoginCount          int    `json:"loginCount"`
		LastFailedLoginTime string `json:"lastFailedLoginTime"`
		PreviousLoginTime   string `json:"previousLoginTime"`
	} `json:"loginInfo"`
	Cookies []*http.Cookie
}

// AcquireSessionCookie creates a new session for a user in JIRA.
// Once a session has been successfully created it can be used to access any of JIRA's remote APIs and also the web UI by passing the appropriate HTTP Cookie header.
// The header will by automatically applied to every API request.
// Note that it is generally preferrable to use HTTP BASIC authentication with the REST API.
// However, this resource may be used to mimic the behaviour of JIRA's log-in page (e.g. to display log-in errors to a user).
//
// JIRA API docs: https://docs.atlassian.com/jira/REST/latest/#auth/1/session
func (s *AuthenticationService) AcquireSessionCookie(username, password string) (bool, error) {
	apiEndpoint := "rest/auth/1/session"
	body := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		username,
		password,
	}

	req, err := s.client.NewRequest("POST", apiEndpoint, body)
	if err != nil {
		return false, err
	}

	session := new(Session)
	resp, err := s.client.Do(req, session)

	if resp != nil {
		session.Cookies = resp.Cookies()
	}

	if err != nil {
		return false, fmt.Errorf("Auth at JIRA instance failed (HTTP(S) request). %s", err)
	}
	if resp != nil && resp.StatusCode != 200 {
		return false, fmt.Errorf("Auth at JIRA instance failed (HTTP(S) request). Status code: %d", resp.StatusCode)
	}

	s.client.session = session

	return true, nil
}

// Authenticated reports if the current Client has an authenticated session with JIRA
func (s *AuthenticationService) Authenticated() bool {
	if s != nil {
		return s.client.session != nil
	}
	return false
}

// Logout logs out the current user that has been authenticated and the session in the client is destroyed.
//
// JIRA API docs: https://docs.atlassian.com/jira/REST/latest/#auth/1/session
func (s *AuthenticationService) Logout() error {
	if s == nil {
		return fmt.Errorf("Authenticaiton Service is not instantiated")
	}
	if s.client.session == nil {
		return fmt.Errorf("No user is authenticated yet.")
	}

	apiEndpoint := "rest/auth/1/session"
	req, err := s.client.NewRequest("DELETE", apiEndpoint, nil)
	if err != nil {
		return fmt.Errorf("Creating the request to log the user out failed : %s", err)
	}
	//var dump interface{}
	resp, err := s.client.Do(req, nil)
	if err != nil {
		return fmt.Errorf("Error sending the logout request: %s", err)
	}
	if resp.StatusCode != 204 {
		return fmt.Errorf("The logout was unsuccessful with status %d", resp.StatusCode)
	}

	// if logout successfull, delete session
	s.client.session = nil

	return nil

}

// GetCurrentUser gets the details of the current user.
//
// JIRA API docs: https://docs.atlassian.com/jira/REST/latest/#auth/1/session
func (s *AuthenticationService) GetCurrentUser() (*Session, error) {
	if s == nil {
		return nil, fmt.Errorf("AUthenticaiton Service is not instantiated")
	}
	if s.client.session == nil {
		return nil, fmt.Errorf("No user is authenticated yet")
	}

	apiEndpoint := "rest/auth/1/session"
	req, err := s.client.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("Could not create request for getting user info : %s", err)
	}

	resp, err := s.client.Do(req, nil)
	if err != nil {
		return nil, fmt.Errorf("Error sending request to get user info : %s", err)
	}

	defer resp.Body.Close()
	ret := new(Session)
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Couldn't read body from the response : %s", err)
	}

	err = json.Unmarshal(data, &ret)

	if err != nil {
		return nil, fmt.Errorf("Could not unmarshall recieved user info : %s", err)
	}

	return ret, nil

}
