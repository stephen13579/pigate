package messenger

type MQTTClientInterface interface {
	Connect() error
	Disconnect()
	NotifyNewCredentials() error
	CommandOpen() error
	CommandLockOpen() error
	CommandClose() error
	NotifyGateOpen() error
	NotifyGateLockedOpen() error
	NotifyGateClosed() error
	IsConnected() bool
	SubscribePigateCommand(callback func(topic string, command string)) error
	SubscribePigateStatus(callback func(topic string, command string)) error
	SubscribeCredentialStatus(callback func(topic string, command string)) error
}
