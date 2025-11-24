package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client
var Ctx = context.Background()

/*
Here The COnnection Of Redis
Will Happen we look for the Connection of
Redis with Username and Password
*/
func ConnectRedis() {

	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	addr := os.Getenv("REDIS_ADDR")
	userName := os.Getenv("REDIS_USERNAME")
	password := os.Getenv("REDIS_PASSWORD")
	dbStr := os.Getenv("REDIS_DB")
	db, err := strconv.Atoi(dbStr)
	if err != nil {
		log.Println("Error while converting string to int")
		db = 0
	}
	Rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: userName,
		Password: password,
		DB:       db,
	})

	_, err = Rdb.Ping(Ctx).Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}

	log.Println("Connected to Redis successfully!")
}

func CreateCacheKey(collectionName, code string) (string, error) {
	if strings.TrimSpace(collectionName) == "" {
		return "", errors.New("collection name is required")
	}
	if strings.TrimSpace(code) == "" {
		return "", errors.New("code is required")
	}

	// Convert collection name to UPPERCASE
	upperName := strings.ToUpper(collectionName)

	key := fmt.Sprintf("%s#%s", upperName, code)
	return key, nil
}

/*
* Give key,value and duration to set in cache
* Try to marshal into key and value types
* If inserted nil,if not inserted return err
 */
func SetCache(c context.Context, key string, value interface{}) error {
	dataBytes, err := json.Marshal(value)
	if err != nil {
		log.Println("Failed to marshal data for cache:", err)
		return err
	}
	expirationStr := os.Getenv("EXPIRATION_TIME")
	expirationMins, err := strconv.Atoi(expirationStr)
	if err != nil {
		log.Println("Failed to convert the string to integer", err)
	}
	expiration := time.Duration(expirationMins) * time.Minute
	err = Rdb.Set(c, key, dataBytes, expiration).Err()
	if err != nil {
		log.Println("Failed to set cache:", err)
		return err
	}
	return nil
}

/*
* Take key and search in cache
* If found then pass the value to the desired variable declared
* If not found pass the error and return false
 */
func GetCache(c context.Context, key string, dest interface{}) (bool, error) {
	dataStr, err := Rdb.Get(c, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, err
		}
		log.Println("error while fetching the cache")
		return false, err
	}
	err = json.Unmarshal([]byte(dataStr), dest)
	if err != nil {
		log.Println("Failed to unmarsahl to destination variable")
		return false, err
	}
	return true, nil
}

func DeleteCache(c context.Context, key string) error {
	return Rdb.Del(c, key).Err()
}
