package background

func RunInBackground(childArgs []string) error {
	err := detach(childArgs)
	if err != nil {
		return err
	}
	return nil
}
