 package global

const (
	AppVersion = "1.0.0" //project version shown in logs or health endponit 

	// Context keys (avoid  using short unique strings)
	// Gin context key for storing the authenticated user ID.
	// Using a string constant reduces risk of typos and collisions.
	
	CtxUserIDKey = "uid"
)
