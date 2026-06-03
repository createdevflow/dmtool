package main

import (
	"fmt"
	"backend/internal/services"
	"os"
	"backend/internal/config"
	"backend/internal/repository"
	"backend/internal/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")
	db, _ := gorm.Open(sqlite.Open("dmtool.db"), &gorm.Config{})
	oauthRepo := repository.NewOAuthRepository(db)
	
	metaCred, _ := oauthRepo.FindByUserAndProvider(1, "meta")
	encKey := []byte(os.Getenv("ENCRYPTION_KEY"))
	if len(encKey) != 32 {
		encKey = []byte("01234567890123456789012345678901")
	}
	token, _ := utils.Decrypt(metaCred.AccessTokenEnc, encKey)

	meta := services.NewMetaService()
	accounts, _ := meta.GetFacebookPageAccounts(token)
	fmt.Printf("ACCOUNTS: %+v\n", accounts)
}
