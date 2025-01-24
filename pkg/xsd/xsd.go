package xsd

import (
	_ "embed"
)

// BaseDatatypes is the XML Schema definition from https://www.w3.org/2001/XMLSchema.xsd
//
//go:embed XMLSchema.xsd
var BaseDatatypes []byte
