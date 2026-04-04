package jobs

type EmptyNotifier struct{}

func (m *EmptyNotifier) Notify(result []CheckNotification) error {
	return nil
}

func (m *EmptyNotifier) IsReady() bool {
	return true
}
