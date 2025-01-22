package soap

import "encoding/xml"

// SOAPEnvelope represents a SOAP envelope structure
type SOAPEnvelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Body    SOAPBody
}

// SOAP11Envelope represents a SOAP 1.1 envelope structure
type SOAP11Envelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Body    SOAP11Body
}

// SOAP12Envelope represents a SOAP 1.2 envelope structure
type SOAP12Envelope struct {
	XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Envelope"`
	Body    SOAP12Body
}

// SOAPBody represents a SOAP body structure
type SOAPBody struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	Content []byte   `xml:",innerxml"`
}

// SOAP11Body represents a SOAP 1.1 body structure
type SOAP11Body struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	Content []byte   `xml:",innerxml"`
}

// SOAP12Body represents a SOAP 1.2 body structure
type SOAP12Body struct {
	XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Body"`
	Content []byte   `xml:",innerxml"`
}

// SOAPFault represents a SOAP fault structure
type SOAPFault struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault"`
	Code    string   `xml:"faultcode"`
	String  string   `xml:"faultstring"`
	Detail  string   `xml:"detail,omitempty"`
}

// SOAP11Fault represents a SOAP 1.1 fault structure
type SOAP11Fault struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault"`
	Code    string   `xml:"faultcode"`
	String  string   `xml:"faultstring"`
	Detail  string   `xml:"detail,omitempty"`
}

// SOAP12Fault represents a SOAP 1.2 fault structure
type SOAP12Fault struct {
	XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Fault"`
	Code    struct {
		Value string `xml:"Value"`
	} `xml:"Code"`
	Reason struct {
		Text string `xml:"Text"`
	} `xml:"Reason"`
	Detail string `xml:"Detail,omitempty"`
}

// Operation represents a WSDL operation
type Operation struct {
	Name       string
	SOAPAction string
	Input      *Message
	Output     *Message
	Fault      *Message
	Binding    string
}

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
