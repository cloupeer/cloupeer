package core

type EventType string

const (
	EventRegister      EventType = "agent.register"
	EventOnline        EventType = "agent.online"
	EventOTACommand    EventType = "ota.command"
	EventOTARequest    EventType = "ota.request"
	EventOTAResponse   EventType = "ota.response"
	EventCommandStatus EventType = "command.status"
)
