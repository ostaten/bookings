package models

import "time"

//User is the user model, primarily represented on backend database
type User struct {
	ID          int
	FirstName   string
	LastName    string
	Email       string
	Password    string
	AccessLevel int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

//Room is the rooms model, primarily represented on backend database
type Room struct {
	ID        int
	RoomName  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

//Restriction is the restrictions model, primarily represented on backend database
type Restriction struct {
	ID              int
	RestrictionName string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

//Reservation is the reservations model, primarily represented on backend database
type Reservation struct {
	ID        int
	FirstName string
	LastName  string
	Email     string
	Phone     string
	StartDate time.Time
	EndDate   time.Time
	RoomID    int
	CreatedAt time.Time
	UpdatedAt time.Time
	Room     Room
}

//RoomRestriction is another restrictions model that encapsulates
//reservations and restrictions for a particular room, primarily represented on backend database
type RoomRestriction struct {
	ID            int
	StartDate     time.Time
	EndDate       time.Time
	RoomID        int
	ReservationID int
	RestrictionID int
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Rooms         Room
	Reservation   Reservation
	Restriction   Restriction
}
