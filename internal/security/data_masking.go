package security

import (
	"regexp"
	"strings"
)

// DataMasker handles PII masking for GDPR compliance
type DataMasker struct {
	emailRegex *regexp.Regexp
	phoneRegex *regexp.Regexp
}

func NewDataMasker() *DataMasker {
	return &DataMasker{
		emailRegex: regexp.MustCompile(`([a-zA-Z0-9._%+-]+)@([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`),
		phoneRegex: regexp.MustCompile(`(\+?1?[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}`),
	}
}

// MaskEmail masks email addresses: john.doe@example.com -> j***@example.com
func (m *DataMasker) MaskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***@***"
	}
	local := parts[0]
	domain := parts[1]
	if len(local) <= 1 {
		return local + "***@" + domain
	}
	return string(local[0]) + "***@" + domain
}

// MaskPhone masks phone numbers: +1-555-123-4567 -> ***-***-4567
func (m *DataMasker) MaskPhone(phone string) string {
	if len(phone) < 4 {
		return "****"
	}
	return "***-***-" + phone[len(phone)-4:]
}

// MaskAccountNumber masks account numbers: 1234567890123456 -> ************3456
func (m *DataMasker) MaskAccountNumber(account string) string {
	if len(account) <= 4 {
		return strings.Repeat("*", len(account))
	}
	return strings.Repeat("*", len(account)-4) + account[len(account)-4:]
}

// MaskName masks names: John Doe -> J*** D***
func (m *DataMasker) MaskName(name string) string {
	parts := strings.Fields(name)
	masked := make([]string, len(parts))
	for i, part := range parts {
		if len(part) <= 1 {
			masked[i] = part + "***"
		} else {
			masked[i] = string(part[0]) + "***"
		}
	}
	return strings.Join(masked, " ")
}

// MaskAddress masks addresses except city/country
func (m *DataMasker) MaskAddress(street, city, country string) string {
	return "*** " + city + ", " + country
}

// MaskSSN masks SSN: 123-45-6789 -> ***-**-6789
func (m *DataMasker) MaskSSN(ssn string) string {
	clean := strings.ReplaceAll(ssn, "-", "")
	if len(clean) < 4 {
		return "***-**-****"
	}
	return "***-**-" + clean[len(clean)-4:]
}

// MaskableData interface for domain objects that support masking
type MaskableData interface {
	Mask(masker *DataMasker)
}
