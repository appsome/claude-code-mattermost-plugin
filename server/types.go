package main

// ActionOption represents an option for dropdown menus
type ActionOption struct {
	Label string
	Value string
}

// ChangeProposal represents a code change proposal from Claude
type ChangeProposal struct {
	ID       string
	Filename string
	Content  string
	Diff     string
}

// ActionContext contains context data for post actions
type ActionContext struct {
	ChangeID  string
	SessionID string
	Action    string
}
