# Fake Data Examples

These examples demonstrate how to use the **fake-data** plugin to generate realistic synthetic data in mock responses. Each request returns different randomised values, making your mocks more lifelike for development and testing.

There are two examples:

- **rest/** — Use `${fake.*}` template expressions in REST response bodies.
- **openapi/** — Let Imposter auto-generate fake data from an OpenAPI schema, using property names, string formats, and the `x-fake-data` extension.

## Prerequisites

The quickest way to run these examples is to install [imposter-cli](https://github.com/imposter-project/imposter-cli)

Install the plugin as follows:

```bash
imposter plugin install -d fake-data -t golang
```

## REST Example

### Running

```bash
imposter up examples/fakedata/rest
```

### Endpoints

#### GET /user

Returns a random user profile.

```bash
curl http://localhost:8080/user
```

Example response (values change on every request):

```json
{
  "firstName": "Alice",
  "lastName": "Johnson",
  "email": "alice.johnson@example.com",
  "username": "alicej",
  "phone": "555-867-5309",
  "city": "Portland",
  "company": "Acme Corp"
}
```

#### GET /address

Returns a random address.

```bash
curl http://localhost:8080/address
```

Example response:

```json
{
  "street": "742 Evergreen Terrace",
  "city": "Springfield",
  "state": "Oregon",
  "zipCode": "97403",
  "country": "United States"
}
```

#### GET /text

Returns random lorem ipsum text.

```bash
curl http://localhost:8080/text
```

Example response:

```json
{
  "word": "voluptatem",
  "sentence": "Quisquam est qui dolorem ipsum quia dolor.",
  "paragraph": "Lorem ipsum dolor sit amet consectetur adipiscing elit..."
}
```

#### GET /mixed

Combines fake data expressions with other template functions (`random.uuid`, `datetime.now`).

```bash
curl http://localhost:8080/mixed
```

Example response:

```json
{
  "id": "a3b1f8c2-1234-4d5e-9abc-def012345678",
  "name": "Alice Johnson",
  "email": "alice.johnson@example.com",
  "registered": "2026-03-26T10:15:30Z",
  "colour": "MediumAquamarine"
}
```

### Template Expression Reference

Fake data expressions use the format `${fake.Category.property}`. Supported categories include:

| Category      | Properties                                                                 |
|---------------|---------------------------------------------------------------------------|
| Name          | firstName, lastName, fullName, prefix, suffix, username, title            |
| Internet      | emailAddress, url, domainName, ipV4Address, ipV6Address, password, slug   |
| Address       | streetAddress, city, state, stateAbbr, country, countryCode, zipCode, latitude, longitude, fullAddress |
| PhoneNumber   | phoneNumber                                                               |
| Company       | name, industry, buzzword, catchPhrase, bs                                 |
| Lorem         | word, sentence, paragraph, characters                                     |
| Color / Colour| name, hex                                                                 |
| Number        | digit, randomNumber, numberBetween                                        |
| Bool          | bool                                                                      |
| Finance       | creditCardNumber, iban, bic                                               |
| Date          | past, future, birthday                                                    |

## OpenAPI Example

This example shows how the fake-data plugin enriches auto-generated OpenAPI responses. When Imposter generates example responses from an OpenAPI schema, it uses the fake-data plugin in three ways:

1. **Property name inference** — A property named `email`, `firstName`, `city`, etc. automatically produces a realistic value.
2. **Format inference** — A string with `format: email`, `format: uri`, or `format: hostname` generates an appropriate value.
3. **`x-fake-data` extension** — Explicitly map a property to a fake data category, e.g. `x-fake-data: Color.name`.

### Running

```bash
imposter up examples/fakedata/openapi
```

### Endpoints

#### GET /users

Returns an auto-generated list of users.

```bash
curl http://localhost:8080/users
```

Example response:

```json
[
  {
    "id": 1,
    "firstName": "Marcus",
    "lastName": "Chen",
    "email": "marcus.chen@example.com",
    "username": "marcusc",
    "phone": "555-123-4567",
    "city": "Denver",
    "country": "Canada",
    "website": "https://example.org",
    "bio": "Quisquam est qui dolorem ipsum quia dolor.",
    "favouriteColour": "Teal"
  }
]
```

#### GET /users/{userId}

Returns a single auto-generated user.

```bash
curl http://localhost:8080/users/42
```

#### GET /companies/{companyId}

Returns an auto-generated company.

```bash
curl http://localhost:8080/companies/100
```

Example response:

```json
{
  "id": 100,
  "company": "Acme Corp",
  "street": "742 Evergreen Terrace",
  "city": "Springfield",
  "state": "Oregon",
  "zipCode": "97403",
  "country": "United States",
  "phone": "555-867-5309",
  "domain": "acme.example.com"
}
```

### OpenAPI Schema Snippet

The key parts of the schema that drive fake data generation:

```yaml
User:
  type: object
  properties:
    firstName:          # inferred from property name
      type: string
    email:
      type: string
      format: email     # inferred from format
    favouriteColour:
      type: string
      x-fake-data: Color.name   # explicit category.property mapping
```

## Configuration

Both examples activate the fake-data plugin by adding a second YAML document to `imposter-config.yaml`:

```yaml
plugin: rest          # or openapi
# ... resources ...
---
plugin: fake-data
```

The `---` separator starts a new YAML document. The `plugin: fake-data` document tells Imposter to load the fake-data external plugin, which provides the data generation capability used by template expressions and OpenAPI example generation.
