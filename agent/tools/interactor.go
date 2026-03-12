package tools

// Interactor provides structured user interaction for tools.
// This is the tool's side channel to the user. The agent never sees these messages.
type Interactor interface {
	// Notify sends a status message to the user.
	Notify(msg string) error

	// NotifyLink sends a message with a clickable link.
	NotifyLink(text string, url string) error

	// RequestApproval asks the user to approve an action. Blocks until the user responds.
	RequestApproval(description string) (approved bool, err error)
}

// BatchInteractor extends Interactor with the ability to present multiple
// approval requests at once, reducing approval fatigue. Implementations
// should present all descriptions together and return a boolean for each.
type BatchInteractor interface {
	Interactor
	RequestBatchApproval(descriptions []string) ([]bool, error)
}
