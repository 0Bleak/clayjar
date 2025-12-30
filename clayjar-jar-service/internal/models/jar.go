package models

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*
Domain model
*/

type Jar struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Category    string             `bson:"category" json:"category"`
	Price       float64            `bson:"price" json:"price"`
	StockQty    int                `bson:"stock_qty" json:"stock_qty"`
	ImageUrl    string             `bson:"image_url" json:"image_url"`
	Attributes  JarAttributes      `bson:"attributes" json:"attributes"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

type JarAttributes struct {
	ClayType       string `bson:"clay_type" json:"clay_type"`
	Dimensions     string `bson:"dimensions" json:"dimensions"`
	Capacity       string `bson:"capacity" json:"capacity"`
	Weight         string `bson:"weight" json:"weight"`
	FoodSafe       bool   `bson:"food_safe" json:"food_safe"`
	MicrowaveSafe  bool   `bson:"microwave_safe" json:"microwave_safe"`
	DishwasherSafe bool   `bson:"dishwasher_safe" json:"dishwasher_safe"`
	GlazeType      string `bson:"glaze_type,omitempty" json:"glaze_type,omitempty"`
	ProductionType string `bson:"production_type" json:"production_type"`
}

/*
DTOs Request model
*/

type CreateJarRequest struct { // This is a data transfer object & is used to obtain data from outside the system
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Category    string        `json:"category"`
	Price       float64       `json:"price"`
	StockQty    int           `json:"stock_qty"`
	ImageURL    string        `json:"image_url"`
	Attributes  JarAttributes `json:"attributes"`
}

/*
Validation
*/

func (j *Jar) Validate() error { //Error validation checks if the attributes of the Jar are correct and logical
	switch {
	case j.Name == "":
		return errors.New("Name attribute is mandatory")
	case len(j.Name) > 200:
		return errors.New("Name attribute must be less than 200 characters")
	case j.Price < 0.01:
		return errors.New("Price must be at least 0.01")
	case j.Price > 10000:
		return errors.New("Price must not exceed 10000")
	case j.StockQty < 0:
		return errors.New("Stock quantity cannot be negative")
	case j.StockQty > 100000:
		return errors.New("Stock quantity exceeds allowed maximum")
	}
	return nil
}

/*
Lifecycle Hooks
*/

func (j *Jar) PrepareForCreate() { //Assigns a new unique ID to a newly created Jar & gives current time to created_at and updated_at
	if j.ID.IsZero() {
		j.ID = primitive.NewObjectID()
	}
	now := time.Now().UTC()
	j.CreatedAt = now
	j.UpdatedAt = now
}

func (j *Jar) PrepareForUpdate() { //Used to update updated_at when updating a jar
	j.UpdatedAt = time.Now().UTC()
}

/*
Domain Events : will be used by kafka for the event driven design
*/

type JarEvent struct {
	Type      string    `json:"type"`              //type of the change
	JarID     string    `json:"jar_id"`            //ID of the jar that changed
	Payload   *Jar      `json:"payload,omitempty"` //Any useful information abt the jar in question
	Timestamp time.Time `json:"timestamp"`         //What time exactly did this change fire
}
