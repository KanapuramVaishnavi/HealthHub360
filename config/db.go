package config

import (
	"HealthHub360/util"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DB *mongo.Database

/*
* connection with dbClient uri
* connect clinet with the db
* check cient is active or not
 */
func ConnectDB() *mongo.Database {
	uri := os.Getenv("MONGO_URI")
	dbName := os.Getenv("DB_NAME")

	clientOptions := options.Client().ApplyURI(uri)

	client, err := mongo.NewClient(clientOptions)
	if err != nil {
		log.Fatal("Error creating Mongo client:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		log.Fatal("Error connecting to MongoDB:", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("MongoDB ping failed:", err)
	}

	fmt.Println("Connected to MongoDB Atlas!")

	DB = client.Database(dbName)
	return DB
}

/*
* Get collection and pass the collection
 */
func OpenCollections(collectionName string) *mongo.Collection {
	return DB.Collection(collectionName)
}

/*
* Input parameters:context,collection,document
* Which insert into particular collection
* Return count,error
 */
func InsertOne(c context.Context, collection *mongo.Collection, document map[string]interface{}) (*mongo.InsertOneResult, error) {
	count, err := collection.InsertOne(c, document)
	if err != nil {

		log.Println("Error while inserting the document", err)
		return nil, errors.New(util.ERR_WHILE_INSERTING)
	}
	return count, nil
}

/*
* To find the document inside particular db collection
* Check for it if error doesnot occur pass the variable data to it
* if err occur either no document found nor the findone error
* Return error
 */
func FindOne(c context.Context, collection *mongo.Collection, filter interface{}, opts *options.FindOneOptions, result interface{}) error {
	SingleResult := collection.FindOne(c, filter, opts)
	if err := SingleResult.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return errors.New(util.ERR_NO_DOC_FOUND)
		}
	}
	if err := SingleResult.Decode(result); err != nil {
		log.Println("Error decoding document:", err)
		return err
	}
	return nil
}

/*
* To findAll inside the particular db collection
* Pass each document into the list of interface
* Check if the document present or not and then Decode and pass to the results
 */

func FindAll(c context.Context, collection *mongo.Collection, filter interface{}, opts *options.FindOptions, results []interface{}) error {

	if filter == nil {
		filter = bson.M{}
	}
	cursor, err := collection.Find(c, filter, opts)
	if cursor.Next(c) {
		doc := make(map[string]interface{})
		if err := cursor.Decode(&doc); err != nil {
			log.Println("Error decoding document:", err)
			return err
		}
		results = append(results, doc)
	}
	if err != nil {
		return err
	}
	return nil
}

/*
* Delete the particular document for the given collection filter condition
 */
func DeleteOne(c context.Context, collection *mongo.Collection, filter interface{}) (*mongo.DeleteResult, error) {
	count, err := collection.DeleteOne(c, filter)
	if err != nil {
		log.Println("Error while deleting the document")
		return nil, errors.New(util.ERR_WHILE_DELETING)
	}
	return count, nil
}

/*
* Delete documents in the particular collection based on the filter provided
 */
func DeleteMany(ctx context.Context, collection *mongo.Collection, filter interface{}) (*mongo.DeleteResult, error) {
	count, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		log.Println("Error while deleting the document")
		return nil, errors.New(util.ERR_WHILE_DELETING)
	}
	return count, nil
}

/*
* Update the document based on the collection given
* Return error if the update fail
* Else return the updated count
 */
func UpdateOne(ctx context.Context, collection *mongo.Collection, filter interface{}, update interface{}) (*mongo.UpdateResult, error) {
	count, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Println("Error while updating the document")
		return nil, errors.New(util.ERR_WHILE_UPDATING)
	}
	return count, nil
}

/*
* Update the document based on the collection given
* Return error if the update fail
* Else return the updated counts
 */
func UpdateMany(ctx context.Context, collection *mongo.Collection, filter interface{}, update interface{}, opts *options.UpdateOptions) (*mongo.UpdateResult, error) {
	count, err := collection.UpdateMany(ctx, filter, update, opts)
	if err != nil {
		log.Println("Error while updating the documents")
		return nil, errors.New(util.ERR_WHILE_UPDATING)
	}
	return count, nil
}
