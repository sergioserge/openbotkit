package tools

// RequestBatchOrSequential attempts batch approval if the interactor
// supports it, otherwise falls back to sequential approval requests.
// Returns a boolean for each description (true = approved).
func RequestBatchOrSequential(interactor Interactor, descriptions []string) ([]bool, error) {
	if len(descriptions) == 0 {
		return nil, nil
	}
	if bi, ok := interactor.(BatchInteractor); ok {
		return bi.RequestBatchApproval(descriptions)
	}
	results := make([]bool, len(descriptions))
	for i, desc := range descriptions {
		approved, err := interactor.RequestApproval(desc)
		if err != nil {
			return nil, err
		}
		results[i] = approved
	}
	return results, nil
}
