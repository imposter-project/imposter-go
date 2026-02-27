package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

// Generate produces a fake data value for a Datafaker-compatible category and property.
// Category names are case-insensitive. Property names are case-insensitive.
func Generate(category, property string) (string, bool) {
	cat := strings.ToLower(category)
	prop := strings.ToLower(property)

	switch cat {
	case "name":
		return generateName(prop)
	case "internet":
		return generateInternet(prop)
	case "address":
		return generateAddress(prop)
	case "phonenumber":
		return generatePhoneNumber(prop)
	case "company":
		return generateCompany(prop)
	case "lorem":
		return generateLorem(prop)
	case "color", "colour":
		return generateColor(prop)
	case "number":
		return generateNumber(prop)
	case "bool":
		return generateBool(prop)
	case "finance":
		return generateFinance(prop)
	case "date":
		return generateDate(prop)
	default:
		return "", false
	}
}

func generateName(prop string) (string, bool) {
	switch prop {
	case "firstname":
		return gofakeit.FirstName(), true
	case "lastname":
		return gofakeit.LastName(), true
	case "fullname", "name":
		return gofakeit.Name(), true
	case "prefix", "nameprefix":
		return gofakeit.NamePrefix(), true
	case "suffix", "namesuffix":
		return gofakeit.NameSuffix(), true
	case "username":
		return gofakeit.Username(), true
	case "title":
		return gofakeit.JobTitle(), true
	default:
		return "", false
	}
}

func generateInternet(prop string) (string, bool) {
	switch prop {
	case "emailaddress", "email":
		return gofakeit.Email(), true
	case "url":
		return gofakeit.URL(), true
	case "domainname", "domain":
		return gofakeit.DomainName(), true
	case "ipv4address", "ipv4":
		return gofakeit.IPv4Address(), true
	case "ipv6address", "ipv6":
		return gofakeit.IPv6Address(), true
	case "macaddress":
		return gofakeit.MacAddress(), true
	case "password":
		return gofakeit.Password(true, true, true, true, false, 12), true
	case "useragent":
		return gofakeit.UserAgent(), true
	case "slug":
		return gofakeit.BuzzWord(), true
	default:
		return "", false
	}
}

func generateAddress(prop string) (string, bool) {
	switch prop {
	case "streetaddress", "street":
		return gofakeit.Street(), true
	case "city":
		return gofakeit.City(), true
	case "state":
		return gofakeit.State(), true
	case "stateabbr", "stateabbreviation":
		return gofakeit.StateAbr(), true
	case "country":
		return gofakeit.Country(), true
	case "countrycode", "countryabbreviation":
		return gofakeit.CountryAbr(), true
	case "zipcode", "postalcode", "zip":
		return gofakeit.Zip(), true
	case "latitude":
		return fmt.Sprintf("%f", gofakeit.Latitude()), true
	case "longitude":
		return fmt.Sprintf("%f", gofakeit.Longitude()), true
	case "fulladdress":
		addr := gofakeit.Address()
		return fmt.Sprintf("%s, %s, %s %s", addr.Street, addr.City, addr.State, addr.Zip), true
	default:
		return "", false
	}
}

func generatePhoneNumber(prop string) (string, bool) {
	switch prop {
	case "phonenumber", "cellphone", "phone":
		return gofakeit.Phone(), true
	default:
		return "", false
	}
}

func generateCompany(prop string) (string, bool) {
	switch prop {
	case "name":
		return gofakeit.Company(), true
	case "industry":
		return gofakeit.CompanySuffix(), true
	case "buzzword":
		return gofakeit.BuzzWord(), true
	case "catchphrase":
		return gofakeit.HipsterSentence(5), true
	case "bs":
		return gofakeit.BS(), true
	default:
		return "", false
	}
}

func generateLorem(prop string) (string, bool) {
	switch prop {
	case "word":
		return gofakeit.Word(), true
	case "sentence":
		return gofakeit.Sentence(6), true
	case "paragraph":
		return gofakeit.Paragraph(1, 3, 6, " "), true
	case "characters":
		return gofakeit.LetterN(10), true
	default:
		return "", false
	}
}

func generateColor(prop string) (string, bool) {
	switch prop {
	case "name":
		return gofakeit.Color(), true
	case "hex":
		return gofakeit.HexColor(), true
	default:
		return "", false
	}
}

func generateNumber(prop string) (string, bool) {
	switch prop {
	case "digit", "randomdigit":
		return gofakeit.Digit(), true
	case "randomnumber", "number":
		return fmt.Sprintf("%d", gofakeit.Number(1, 1000)), true
	case "numberbetween":
		return fmt.Sprintf("%d", gofakeit.Number(1, 100)), true
	default:
		return "", false
	}
}

func generateBool(prop string) (string, bool) {
	switch prop {
	case "bool", "boolean":
		return fmt.Sprintf("%t", gofakeit.Bool()), true
	default:
		return "", false
	}
}

func generateFinance(prop string) (string, bool) {
	switch prop {
	case "creditcardnumber":
		return gofakeit.CreditCardNumber(nil), true
	case "iban":
		return gofakeit.AchRouting(), true
	case "bic":
		return gofakeit.AchRouting(), true
	default:
		return "", false
	}
}

func generateDate(prop string) (string, bool) {
	now := time.Now()
	switch prop {
	case "past":
		past := gofakeit.DateRange(now.AddDate(-2, 0, 0), now)
		return past.Format(time.RFC3339), true
	case "future":
		future := gofakeit.DateRange(now, now.AddDate(2, 0, 0))
		return future.Format(time.RFC3339), true
	case "birthday":
		bday := gofakeit.DateRange(now.AddDate(-60, 0, 0), now.AddDate(-18, 0, 0))
		return bday.Format("2006-01-02"), true
	default:
		return "", false
	}
}
