// internal/handlers/validators.go - INPUT VALIDATION
package handlers

import (
    "html"
    "regexp"
    "strings"
)

var (
    usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
    gamenameRegex = regexp.MustCompile(`^.{1,30}$`)
)

// SanitizeString removes HTML and limits length
func SanitizeString(input string, maxLen int) string {
    input = html.EscapeString(input)
    input = strings.TrimSpace(input)
    if len(input) > maxLen {
        input = input[:maxLen]
    }
    return input
}

// ValidateUsername checks username format
func ValidateUsername(username string) bool {
    return usernameRegex.MatchString(username)
}

// ValidatePassword checks minimum length
func ValidatePassword(password string) bool {
    return len(password) >= 4 && len(password) <= 100
}

// ValidateBetAmount checks bet limits
func ValidateBetAmount(amount float64) (bool, string) {
    if amount < 10 {
        return false, "Minimum bet is KSh 10"
    }
    if amount > 50000 {
        return false, "Maximum bet is KSh 50,000"
    }
    return true, ""
}

// ValidateBetSelections checks number of picks
func ValidateBetSelections(count int) (bool, string) {
    if count < 1 {
        return false, "Select at least one match"
    }
    if count > 20 {
        return false, "Maximum 20 selections per bet"
    }
    return true, ""
}

// ValidatePhoneNumber checks Kenyan phone format
func ValidatePhoneNumber(phone string) bool {
    phone = strings.TrimSpace(phone)
    if len(phone) == 10 && strings.HasPrefix(phone, "07") {
        return true
    }
    if len(phone) == 12 && strings.HasPrefix(phone, "2547") {
        return true
    }
    return false
}