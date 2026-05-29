package main
import (
	"fmt"
	"os"
	"backend/internal/utils"
	"backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)
func main() {
	utils.SetEncryptionKey(os.Getenv("ENCRYPTION_KEY"))
	db, err := gorm.Open(sqlite.Open("dmtool.db"), &gorm.Config{})
	if err != nil { panic(err) }
	r := repository.NewOAuthRepository(db)
	cred, err := r.GetCredential(1, "meta")
	if err != nil { panic(err) }
	fmt.Println("Token:", cred.AccessToken)
}
