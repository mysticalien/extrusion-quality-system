HASH="$HASH" cat > /tmp/check_generated_hash.go <<'EOF'
package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	hash := os.Getenv("HASH")

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("password"))
	fmt.Println("password matches:", err == nil)

	if err != nil {
		fmt.Println("error:", err)
	}
}
EOF

HASH="$HASH" go run /tmp/check_generated_hash.go