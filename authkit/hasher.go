package authkit

// Hasher holds logic of hashing and checking the password.
type Hasher interface {
	// HashPassword takes password and return a hasher from it.
	HashPassword(pass string) (string, error)
	// CheckPassword takes password and perform comparison with
	// stored hashed password.
	CheckPassword(hash, pass string) error
}