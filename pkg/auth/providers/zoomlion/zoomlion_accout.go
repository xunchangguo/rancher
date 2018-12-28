package zoomlion

type searchResult struct {
	Items []Account `json:"items"`
}

//Account defines properties an account on zoomlion has
type Account struct {
	ID        string `json:"id,omitempty"`
	Login     string `json:"sub,omitempty"`
	Name      string `json:"name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Type      string `json:"type,omitempty"`
}

//Team defines properties a team on zoomlion has
type Team struct {
	ID           string                    `json:"id,omitempty"`
	Organization map[string]interface{} `json:"organization,omitempty"`
	Name         string                 `json:"name,omitempty"`
	Slug         string                 `json:"slug,omitempty"`
}

func (t *Team) toZoomlionAccount(url string, account *Account) {
	account.ID = t.ID
	account.Name = t.Name
	account.AvatarURL = t.Organization["avatar_url"].(string)
	account.Login = t.Slug
}
