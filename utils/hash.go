package utils

import "golang.org/x/crypto/bcrypt"


// takes plain text password and retuens secure bcrypt hash 
//// bcrypt.DefaultCost is a sane default (adjust for security/performance needs).

func HashPassword(raw string)(string , error){
	b , err := bcrypt.GenerateFromPassword([]byte(raw),bcrypt.DefaultCost)
return string(b),err // Return the hash as string (store in DB).
}

// CheckPassword verifies a plaintext password against a stored hash.
// It returns true when the password matches.
func CheckPassword(hash ,raw string) bool{
	return bcrypt.CompareHashAndPassword([]byte(hash),[]byte(raw))==nil
}