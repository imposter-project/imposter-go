package main

import (
	"golang.org/x/crypto/bcrypt"
)

func (o *OIDCServer) authenticateUser(username, password string) *User {
	user := o.config.GetUser(username)
	if user == nil {
		return nil
	}

	// For simplicity, we're doing plain text comparison
	// In production, passwords should be hashed with bcrypt
	if user.Password == password {
		return user
	}

	// Also check if the stored password is bcrypt hashed
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err == nil {
		return user
	}

	return nil
}

func (o *OIDCServer) hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (o *OIDCServer) getUserClaims(user *User, scopes []string) map[string]interface{} {
	claims := make(map[string]interface{})

	// Always include sub claim
	if sub, exists := user.Claims["sub"]; exists {
		claims["sub"] = sub
	} else {
		claims["sub"] = user.Username
	}

	// Include claims based on requested scopes
	for _, scope := range scopes {
		switch scope {
		case "openid":
			// sub already included above
		case "profile":
			if name, exists := user.Claims["name"]; exists {
				claims["name"] = name
			}
			if givenName, exists := user.Claims["given_name"]; exists {
				claims["given_name"] = givenName
			}
			if familyName, exists := user.Claims["family_name"]; exists {
				claims["family_name"] = familyName
			}
			if nickname, exists := user.Claims["nickname"]; exists {
				claims["nickname"] = nickname
			}
			if preferredUsername, exists := user.Claims["preferred_username"]; exists {
				claims["preferred_username"] = preferredUsername
			}
		case "email":
			if email, exists := user.Claims["email"]; exists {
				claims["email"] = email
			}
			if emailVerified, exists := user.Claims["email_verified"]; exists {
				claims["email_verified"] = emailVerified == "true"
			}
		case "address":
			if address, exists := user.Claims["address"]; exists {
				claims["address"] = address
			}
		case "phone":
			if phoneNumber, exists := user.Claims["phone_number"]; exists {
				claims["phone_number"] = phoneNumber
			}
			if phoneNumberVerified, exists := user.Claims["phone_number_verified"]; exists {
				claims["phone_number_verified"] = phoneNumberVerified == "true"
			}
		}
	}

	return claims
}
