package bootstrap

import (
	"analytics-service/db"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
)

func ClearAnalytics() {

	collection := db.Client.Database("testdb").Collection("analytics")
	_, err := collection.DeleteMany(context.TODO(), bson.D{})
	if err != nil {
		fmt.Println("Error clearing analytics:", err)
	} else {
		fmt.Println("Cleared analytics from database")
	}
}
