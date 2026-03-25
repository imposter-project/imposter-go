package main

import (
	"strings"
)

// propertyNameMappings maps OpenAPI property names (lowercased) to category.property pairs.
var propertyNameMappings = map[string][2]string{
	// Name
	"firstname":  {"Name", "firstName"},
	"first_name": {"Name", "firstName"},
	"lastname":   {"Name", "lastName"},
	"last_name":  {"Name", "lastName"},
	"name":       {"Name", "fullName"},
	"fullname":   {"Name", "fullName"},
	"full_name":  {"Name", "fullName"},
	"username":   {"Name", "username"},
	"user_name":  {"Name", "username"},

	// Internet
	"email":         {"Internet", "emailAddress"},
	"emailaddress":  {"Internet", "emailAddress"},
	"email_address": {"Internet", "emailAddress"},
	"url":           {"Internet", "url"},
	"website":       {"Internet", "url"},
	"domain":        {"Internet", "domainName"},
	"domainname":    {"Internet", "domainName"},
	"domain_name":   {"Internet", "domainName"},
	"ipaddress":     {"Internet", "ipV4Address"},
	"ip_address":    {"Internet", "ipV4Address"},
	"password":      {"Internet", "password"},

	// Address
	"address":        {"Address", "streetAddress"},
	"streetaddress":  {"Address", "streetAddress"},
	"street_address": {"Address", "streetAddress"},
	"street":         {"Address", "streetAddress"},
	"city":           {"Address", "city"},
	"state":          {"Address", "state"},
	"country":        {"Address", "country"},
	"zipcode":        {"Address", "zipCode"},
	"zip_code":       {"Address", "zipCode"},
	"postalcode":     {"Address", "zipCode"},
	"postal_code":    {"Address", "zipCode"},
	"zip":            {"Address", "zipCode"},
	"latitude":       {"Address", "latitude"},
	"longitude":      {"Address", "longitude"},

	// Phone
	"phone":         {"PhoneNumber", "phoneNumber"},
	"phonenumber":   {"PhoneNumber", "phoneNumber"},
	"phone_number":  {"PhoneNumber", "phoneNumber"},
	"cellphone":     {"PhoneNumber", "phoneNumber"},
	"cell_phone":    {"PhoneNumber", "phoneNumber"},
	"mobilenumber":  {"PhoneNumber", "phoneNumber"},
	"mobile_number": {"PhoneNumber", "phoneNumber"},

	// Company
	"company":      {"Company", "name"},
	"companyname":  {"Company", "name"},
	"company_name": {"Company", "name"},

	// Text
	"description": {"Lorem", "sentence"},
	"summary":     {"Lorem", "sentence"},
	"title":       {"Name", "title"},
	"bio":         {"Lorem", "sentence"},

	// Color
	"color":  {"Color", "name"},
	"colour": {"Color", "name"},
}

// formatMappings maps OpenAPI string formats to category.property pairs.
var formatMappings = map[string][2]string{
	"email":     {"Internet", "emailAddress"},
	"uri":       {"Internet", "url"},
	"hostname":  {"Internet", "domainName"},
	"ipv4":      {"Internet", "ipV4Address"},
	"ipv6":      {"Internet", "ipV6Address"},
	"password":  {"Internet", "password"},
	"date-time": {"Date", "past"},
	"date":      {"Date", "past"},
}

// GenerateForPropertyName generates fake data based on an OpenAPI property name.
func GenerateForPropertyName(propertyName string) (string, bool) {
	key := strings.ToLower(propertyName)
	if mapping, ok := propertyNameMappings[key]; ok {
		return Generate(mapping[0], mapping[1])
	}
	return "", false
}

// GenerateForFormat generates fake data based on an OpenAPI string format.
func GenerateForFormat(format string) (string, bool) {
	key := strings.ToLower(format)
	if mapping, ok := formatMappings[key]; ok {
		return Generate(mapping[0], mapping[1])
	}
	return "", false
}
