package bitbucket

type PullRequest struct {
	Id           int           `json:"id"`
	Title        string        `json:"title"`
	State        string        `json:"state"`
	Author       User          `json:"author"`
	Participants []Participant `json:"participants"`
	Links        Links         `json:"links"`
}

type Participant struct {
	User     User `json:"user"`
	Approved bool `json:"approved"`
}

type Links struct {
	Self    Link `json:"self"`
	Approve Link `json:"approve"`
	Html    Link `json:"html"`
}

type Link struct {
	Href string `json:"href"`
}

type Credentials struct {
	User  string
	Token string
}

type User struct {
	AccountID string `json:"account_id"`
}
