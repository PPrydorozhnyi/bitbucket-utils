package bitbucket

type PullRequest struct {
	ID           int           `json:"id"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	State        string        `json:"state"`
	Draft        bool          `json:"draft"`
	Queued       bool          `json:"queued"`
	Author       User          `json:"author"`
	Reviewers    []User        `json:"reviewers"`
	Participants []Participant `json:"participants"`
	Destination  Destination   `json:"destination"`
	Links        Links         `json:"links"`
}

type Participant struct {
	User     User `json:"user"`
	Approved bool `json:"approved"`
}

type Destination struct {
	Repository Repository `json:"repository"`
}

type Repository struct {
	Slug     string  `json:"slug"`
	FullName string  `json:"full_name"`
	Project  Project `json:"project"`
}

type Project struct {
	UUID string `json:"uuid"`
}

type Links struct {
	HTML Link `json:"html"`
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
	UUID      string `json:"uuid"`
}
