package xsd

import (
	_ "embed"
)

// BaseDatatypes is the XML Schema datatypes definition from https://www.w3.org/2001/XMLSchema-datatypes.xsd
//
//go:embed XMLSchema-datatypes.xsd
var BaseDatatypes []byte
