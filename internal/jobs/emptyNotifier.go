package jobs

type EmptyNotifier struct{}

func (m *EmptyNotifier) Notify(groups []CheckNotificationGroup) error {
	return nil
}

func (m *EmptyNotifier) IsReady() bool {
	return true
}
