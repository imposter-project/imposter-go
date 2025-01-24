package wsdlmsg

import "encoding/xml"

// WSDLMessageType represents the type of WSDL message
type WSDLMessageType int

const (
	// ElementMessageType represents a simple WSDL message with an element
	ElementMessageType WSDLMessageType = iota + 1
	// TypeMessageType represents a simple WSDL message with a type
	TypeMessageType
	// CompositeMessageType represents a composite WSDL message
	CompositeMessageType
)

// Message represents a WSDL message
type Message interface {
	GetMessageType() WSDLMessageType
}

// ElementMessage represents a simple WSDL message
type ElementMessage struct {
	Element *xml.Name
}

func (m *ElementMessage) GetMessageType() WSDLMessageType {
	return ElementMessageType
}

// TypeMessage represents a simple WSDL message
type TypeMessage struct {
	PartName string
	Type     *xml.Name
}

func (m *TypeMessage) GetMessageType() WSDLMessageType {
	return TypeMessageType
}

// CompositeMessage represents a composite WSDL message
type CompositeMessage struct {
	MessageName string
	Parts       *[]Message
}

func (m *CompositeMessage) GetMessageType() WSDLMessageType {
	return CompositeMessageType
}
