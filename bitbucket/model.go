package bitbucket

type PullRequest struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
	State string `json:"state"`
	Links Links  `json:"links"`
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
