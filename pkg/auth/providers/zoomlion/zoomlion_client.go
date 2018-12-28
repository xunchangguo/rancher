package zoomlion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/tomnomnom/linkheader"

	"github.com/rancher/types/apis/management.cattle.io/v3"
)

const (
	zleAPI                  = "" //"/api/v3"
	zoomlionDefaultHostName = "https://zoomlion.com"
)

//ZClient implements a httpclient for zoomlion
type ZClient struct {
	httpClient *http.Client
}

func (g *ZClient) getAccessToken(code string, config *v3.ZoomlionConfig) (string, error) {

	form := url.Values{}
	form.Add("client_id", config.ClientID)
	form.Add("client_secret", config.ClientSecret)
	form.Add("code", code)
	form.Add("redirect_uri", "https://10.39.172.65:8443/verify-auth")
	form.Add("grant_type", "authorization_code")

	url := g.getURL("TOKEN", config)

	b, err := g.postTo(url, form)
	if err != nil {
		logrus.Errorf("Zoomlion getAccessToken: GET url %v received error from zoomlion, err: %v", url, err)
		return "", err
	}

	// Decode the response
	var respMap map[string]interface{}

	if err := json.Unmarshal(b, &respMap); err != nil {
		logrus.Errorf("Zoomlion getAccessToken: received error unmarshalling response body, err: %v", err)
		return "", err
	}

	if respMap["error"] != nil {
		desc := respMap["error_description"]
		logrus.Errorf("Received Error from zoomlion %v, description from zoomlion %v", respMap["error"], desc)
		return "", fmt.Errorf("Received Error from zoomlion %v, description from zoomlion %v", respMap["error"], desc)
	}

	acessToken, ok := respMap["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("Received Error reading accessToken from response %v", respMap)
	}
	return acessToken, nil
}

func (g *ZClient) getUser(zlAccessToken string, config *v3.ZoomlionConfig) (Account, error) {

	url := g.getURL("USER_INFO", config)
	b, _, err := g.getFromZoomlion(zlAccessToken, url)
	if err != nil {
		logrus.Errorf("Zoomlion getZoomlionUser: GET url %v received error from zoomlion, err: %v", url, err)
		return Account{}, err
	}
	var zlAccout Account

	if err := json.Unmarshal(b, &zlAccout); err != nil {
		logrus.Errorf("Zoomlion getZoomlionUser: error unmarshalling response, err: %v", err)
		return Account{}, err
	}

	//TODO test
	zlAccout.ID = "100"

	return zlAccout, nil
}

func (g *ZClient) getOrgs(zlAccessToken string, config *v3.ZoomlionConfig) ([]Account, error) {
	var orgs []Account

	url := g.getURL("ORG_INFO", config)
	responses, err := g.paginateZoomlion(zlAccessToken, url)
	if err != nil {
		logrus.Errorf("Zoomlion getOrgs: GET url %v received error from zoomlion, err: %v", url, err)
		return orgs, err
	}

	for _, b := range responses {
		var orgObjs []Account
		if err := json.Unmarshal(b, &orgObjs); err != nil {
			logrus.Errorf("Zoomlion getOrgs: received error unmarshalling org array, err: %v", err)
			return nil, err
		}
		orgs = append(orgs, orgObjs...)
	}

	return orgs, nil
}

func (g *ZClient) getTeams(zlAccessToken string, config *v3.ZoomlionConfig) ([]Account, error) {
	var teams []Account

	url := g.getURL("TEAMS", config)
	responses, err := g.paginateZoomlion(zlAccessToken, url)
	if err != nil {
		logrus.Errorf("Zoomlion getTeams: GET url %v received error from Zoomlion, err: %v", url, err)
		return teams, err
	}
	for _, response := range responses {
		teamObjs, err := g.getTeamInfo(response, config)

		if err != nil {
			logrus.Errorf("Zoomlion getTeams: received error unmarshalling teams array, err: %v", err)
			return teams, err
		}
		for _, teamObj := range teamObjs {
			teams = append(teams, teamObj)
		}

	}
	return teams, nil
}

func (g *ZClient) getTeamInfo(b []byte, config *v3.ZoomlionConfig) ([]Account, error) {
	var teams []Account
	var teamObjs []Team
	if err := json.Unmarshal(b, &teamObjs); err != nil {
		logrus.Errorf("Zoomlion getTeamInfo: received error unmarshalling team array, err: %v", err)
		return teams, err
	}

	url := g.getURL("TEAM_PROFILE", config)
	for _, team := range teamObjs {
		teamAcct := Account{}
		team.toZoomlionAccount(url, &teamAcct)
		teams = append(teams, teamAcct)
	}

	return teams, nil
}

func (g *ZClient) getTeamByID(id string, zlAccessToken string, config *v3.ZoomlionConfig) (Account, error) {
	var teamAcct Account

	url := g.getURL("TEAM", config) + id
	b, _, err := g.getFromZoomlion(zlAccessToken, url)
	if err != nil {
		logrus.Errorf("Zoomlion getTeamByID: GET url %v received error from zoomlion, err: %v", url, err)
		return teamAcct, err
	}
	var teamObj Team
	if err := json.Unmarshal(b, &teamObj); err != nil {
		logrus.Errorf("Zoomlion getTeamByID: received error unmarshalling team array, err: %v", err)
		return teamAcct, err
	}
	url = g.getURL("TEAM_PROFILE", config)
	teamObj.toZoomlionAccount(url, &teamAcct)

	return teamAcct, nil
}

func (g *ZClient) paginateZoomlion(zlAccessToken string, url string) ([][]byte, error) {
	var responses [][]byte
	var err error
	var response []byte
	nextURL := url
	for nextURL != "" {
		response, nextURL, err = g.getFromZoomlion(zlAccessToken, nextURL)
		if err != nil {
			return nil, err
		}
		responses = append(responses, response)
	}

	return responses, nil
}

func (g *ZClient) nextZoomlionPage(response *http.Response) string {
	header := response.Header.Get("link")

	if header != "" {
		links := linkheader.Parse(header)
		for _, link := range links {
			if link.Rel == "next" {
				return link.URL
			}
		}
	}

	return ""
}

func (g *ZClient) searchUsers(searchTerm, searchType string, zlAccessToken string, config *v3.ZoomlionConfig) ([]Account, error) {
	if searchType == "group" {
		searchType = orgType
	}

	search := searchTerm
	if searchType != "" {
		search += "+type:" + searchType
	}
	search = URLEncoded(search)
	url := g.getURL("USER_SEARCH", config) + search

	b, _, err := g.getFromZoomlion(zlAccessToken, url)
	if err != nil {
		// no match on search returns an error. do not log
		return nil, nil
	}

	result := &searchResult{}
	if err := json.Unmarshal(b, result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

func (g *ZClient) getOrgByName(org string, zlAccessToken string, config *v3.ZoomlionConfig) (Account, error) {
	org = URLEncoded(org)
	url := g.getURL("ORGS", config) + org

	b, _, err := g.getFromZoomlion(zlAccessToken, url)
	if err != nil {
		logrus.Debugf("Zoomlion getOrgByName: GET url %v received error from zoomlion, err: %v", url, err)
		return Account{}, err
	}
	var zlAccout Account
	if err := json.Unmarshal(b, &zlAccout); err != nil {
		logrus.Errorf("Zoomlion getOrgByName: error unmarshalling response, err: %v", err)
		return Account{}, err
	}

	return zlAccout, nil
}

func (g *ZClient) getUserOrgByID(id string, zlAccessToken string, config *v3.ZoomlionConfig) (Account, error) {
	url := g.getURL("USER_INFO", config) + "/" + id

	b, _, err := g.getFromZoomlion(zlAccessToken, url)
	if err != nil {
		logrus.Errorf("Zoomlion getUserOrgById: GET url %v received error from zoomlion, err: %v", url, err)
		return Account{}, err
	}
	var zlAccout Account

	if err := json.Unmarshal(b, &zlAccout); err != nil {
		logrus.Errorf("Zoomlion getUserOrgById: error unmarshalling response, err: %v", err)
		return Account{}, err
	}

	return zlAccout, nil
}

//URLEncoded encodes the string
func URLEncoded(str string) string {
	u, err := url.Parse(str)
	if err != nil {
		logrus.Errorf("Error encoding the url: %s, error: %v", str, err)
		return str
	}
	return u.String()
}

func (g *ZClient) postTo(url string, form url.Values) ([]byte, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(form.Encode()))
	if err != nil {
		logrus.Error(err)
	}
	req.PostForm = form
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")
	resp, err := g.httpClient.Do(req)
	if err != nil {
		logrus.Errorf("Received error from zoomlion: %v", err)
		return nil, err
	}

	defer resp.Body.Close()
	// Check the status code
	switch resp.StatusCode {
	case 200:
	case 201:
	default:
		var body bytes.Buffer
		io.Copy(&body, resp.Body)
		return nil, fmt.Errorf("Request failed, got status code: %d. Response: %s",
			resp.StatusCode, body.Bytes())
	}
	return ioutil.ReadAll(resp.Body)
}

func (g *ZClient) getFromZoomlion(zlAccessToken string, url string) ([]byte, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Add("Authorization", "Bearer "+zlAccessToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.103 Safari/537.36)")
	resp, err := g.httpClient.Do(req)
	if err != nil {
		logrus.Errorf("Received error from zoomlion: %v", err)
		return nil, "", err
	}
	defer resp.Body.Close()
	// Check the status code
	switch resp.StatusCode {
	case 200:
	case 201:
	default:
		var body bytes.Buffer
		io.Copy(&body, resp.Body)
		return nil, "", fmt.Errorf("request failed, got status code: %d. Response: %s",
			resp.StatusCode, body.Bytes())
	}

	nextURL := g.nextZoomlionPage(resp)
	b, err := ioutil.ReadAll(resp.Body)
	return b, nextURL, err
}

func (g *ZClient) getURL(endpoint string, config *v3.ZoomlionConfig) string {

	var hostName, apiEndpoint, toReturn string

	if config.Hostname != "" {
		scheme := "http://"
		if config.TLS {
			scheme = "https://"
		}
		hostName = scheme + config.Hostname
		apiEndpoint = scheme + config.Hostname + zleAPI
	} else {
		hostName = zoomlionDefaultHostName
		apiEndpoint = zoomlionDefaultHostName  + zleAPI
	}

	switch endpoint {
	case "API":
		toReturn = apiEndpoint
	case "TOKEN":
		toReturn = hostName + "/oauth2/token"
	case "USERS":
		toReturn = apiEndpoint + "/users/"
	case "ORGS":
		toReturn = apiEndpoint + "/orgs/"
	case "USER_INFO":
		toReturn = apiEndpoint + "/userinfo"
	case "ORG_INFO":
		toReturn = apiEndpoint + "/user/orgs?per_page=1"
	case "USER_PICTURE":
		toReturn = "https://avatars.githubusercontent.com/u/" + endpoint + "?v=3&s=72"
	case "USER_SEARCH":
		toReturn = apiEndpoint + "/search/users?q="
	case "TEAM":
		toReturn = apiEndpoint + "/teams/"
	case "TEAMS":
		toReturn = apiEndpoint + "/user/teams?per_page=100"
	case "TEAM_PROFILE":
		toReturn = hostName + "/orgs/%s/teams/%s"
	default:
		toReturn = apiEndpoint
	}

	return toReturn
}
