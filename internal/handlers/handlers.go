package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ostaten/bookings/internal/config"
	"github.com/ostaten/bookings/internal/driver"
	"github.com/ostaten/bookings/internal/forms"
	"github.com/ostaten/bookings/internal/helpers"
	"github.com/ostaten/bookings/internal/models"
	"github.com/ostaten/bookings/internal/render"
	"github.com/ostaten/bookings/internal/repository"
	"github.com/ostaten/bookings/internal/repository/dbrepo"
)

var Repo *Repository

type Repository struct {
	App *config.AppConfig
	DB  repository.DatabaseRepo
}

// NewRepo creates a new respository
func NewRepo(a *config.AppConfig, db *driver.DB) *Repository {
	return &Repository{
		App: a,
		DB:  dbrepo.NewPostgresRepo(db.SQL, a),
	}
}

// NewRepo creates a new respository
func NewTestRepo(a *config.AppConfig) *Repository {
	return &Repository{
		App: a,
		DB:  dbrepo.NewTestingRepo(a),
	}
}


// NewHandlers sets the repository for the handlers
func NewHandlers(r *Repository) {
	Repo = r
}

func (m *Repository) Home(w http.ResponseWriter, r *http.Request) {
	m.DB.AllUsers()
	render.Template(w, r, "home.page.tmpl", &models.TemplateData{})
}

func (m *Repository) About(w http.ResponseWriter, r *http.Request) {
	//send the data to the template
	render.Template(w, r, "about.page.tmpl", &models.TemplateData{})
}

// Reservation renders the make a reservation page and displays form
func (m *Repository) Reservation(w http.ResponseWriter, r *http.Request) {
	res, ok := m.App.Session.Get(r.Context(), "reservation").(models.Reservation)
	if !ok {
		m.App.Session.Put(r.Context(), "error", "Can't get reservation from session")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	room, err := m.DB.GetRoomByID(res.RoomID)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "Can't get room by ID")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	res.Room.RoomName = room.RoomName

	m.App.Session.Put(r.Context(), "reservation", res)

	sd := res.StartDate.Format("2006-01-02")
	ed := res.EndDate.Format("2006-01-02")

	stringMap := make(map[string]string)
	stringMap["start_date"] = sd
	stringMap["end_date"] = ed

	data := make(map[string]interface{})
	data["reservation"] = res

	render.Template(w, r, "make-reservation.page.tmpl", &models.TemplateData{
		Form:      forms.New(nil),
		Data:      data,
		StringMap: stringMap,
	})
}

// PostReservation handles the posting of a reservation form
func (m *Repository) PostReservation(w http.ResponseWriter, r *http.Request) {
	reservation, ok := m.App.Session.Get(r.Context(), "reservation").(models.Reservation)

	if !ok {
		m.App.Session.Put(r.Context(), "error", "cannot get the reservation from session")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	err := r.ParseForm()

	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't parse form")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	roomID, _ := strconv.Atoi(r.Form.Get("room_id"))

	room, err := m.DB.GetRoomByID(roomID)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "no room id")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	reservation.FirstName = r.Form.Get("first_name")
	reservation.LastName = r.Form.Get("last_name")
	reservation.Phone = r.Form.Get("phone")
	reservation.Email = r.Form.Get("email")
	reservation.Room = room

	stringMap := make(map[string]string)
	stringMap["start_date"] = r.Form.Get("start_date")
	stringMap["end_date"] = r.Form.Get("end_date")

	form := forms.New(r.PostForm)

	form.Required("first_name", "last_name", "email")
	form.MinLength("first_name", 3)
	form.IsEmail("email")

	if !form.Valid() {
		data := make(map[string]interface{})
		data["reservation"] = reservation

		render.Template(w, r, "make-reservation.page.tmpl", &models.TemplateData{
			Form: form,
			Data: data,
			StringMap: stringMap,
		})
		return
	}

	newReservationID, err := m.DB.InsertReservation(reservation)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't insert reservation into DB")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	restriction := models.RoomRestriction{
		StartDate:     reservation.StartDate,
		EndDate:       reservation.EndDate,
		RoomID:        reservation.RoomID,
		ReservationID: newReservationID,
		RestrictionID: 1,
	}

	err = m.DB.InsertRoomRestriction(restriction)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't insert room restriction")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	log.Println(reservation.StartDate.Format("2006/01/02"))
	log.Println(reservation.StartDate.Year())

	//send notifications - first to the guest
	htmlMessage := fmt.Sprintf(`
		<strong>Reservation Confirmation</strong>
		<br>
		Dear %s, <br>
		Just confirming your reservation from %s to %s in %s. <br>
		Thanks, <br>
		The Crew at Pirate's Galley
	`, reservation.FirstName, reservation.StartDate.Format("2006/01/02"), reservation.EndDate.Format("2006/01/02"), reservation.Room.RoomName)

	msg := models.MailData{
		To:	reservation.Email,
		From: "me@here.com",
		Subject: "Reservation Confirmation",
		Content: htmlMessage,
		Template: "basic.html",
	}

	m.App.MailChan <- msg

	//next send email to property owner
	htmlMessage = fmt.Sprintf(`
		<strong>Reservation Confirmation</strong>
		<br>
		Person: %s %s<br>
		Start Date: %s<br>
		End Date: %s<br>
		Lodging: %s<br>
	`, reservation.FirstName, reservation.LastName, reservation.StartDate.Format("2006/01/02"), reservation.EndDate.Format("2006/01/02"), reservation.Room.RoomName)

	msg = models.MailData{
		To:	"bigBoss@owner.com",
		From: "me@here.com",
		Subject: "Reservation Confirmation",
		Content: htmlMessage,
	}

	m.App.MailChan <- msg

	m.App.Session.Put(r.Context(), "reservation", reservation)

	http.Redirect(w, r, "/reservation-summary", http.StatusSeeOther)
}

// Captains offers reservation for Captain's Quarters
func (m *Repository) Captains(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "captains.page.tmpl", &models.TemplateData{})
}

// Crews offers reservation for Crew's Cabin
func (m *Repository) Crews(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "crews.page.tmpl", &models.TemplateData{})
}

// Availability allows the user to check if a room is available
func (m *Repository) Availability(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "search-availability.page.tmpl", &models.TemplateData{})
}

// PostAvailability posts the availability info
func (m *Repository) PostAvailability(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "Can't parse form")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	start := r.Form.Get("start")
	end := r.Form.Get("end")
	layout := "2006-01-02"
	startDate, err := time.Parse(layout, start)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "Can't parse start date")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	endDate, err := time.Parse(layout, end)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "Can't parse end date")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	rooms, err := m.DB.SearchAvailabilityForAllRooms(startDate, endDate)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "Can't find room availability")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	for _, i := range rooms {
		m.App.InfoLog.Println("Room:", i.ID, i.RoomName)
	}

	if len(rooms) == 0 {
		m.App.InfoLog.Println("Ugh, no rooms available")
		m.App.Session.Put(r.Context(), "error", "No availablity")
		http.Redirect(w, r, "/search-availability", http.StatusSeeOther)
		return
	}

	data := make(map[string]interface{})
	data["rooms"] = rooms

	res := models.Reservation{
		StartDate: startDate,
		EndDate:   endDate,
	}

	m.App.Session.Put(r.Context(), "reservation", res)

	render.Template(w, r, "choose-room.page.tmpl", &models.TemplateData{
		Data: data,
	})
}

// Displays list of available rooms
func (m *Repository) ChooseRoom(w http.ResponseWriter, r *http.Request) {
	roomID, err := strconv.Atoi(strings.Split(r.RequestURI, "/")[2])
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "Can't find atoi")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	res, ok := m.App.Session.Get(r.Context(), "reservation").(models.Reservation)
	if !ok {
		m.App.Session.Put(r.Context(), "error", "Can't get session context")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	res.RoomID = roomID

	m.App.Session.Put(r.Context(), "reservation", res)

	http.Redirect(w, r, "/make-reservation", http.StatusSeeOther)
}

//BookRoom takes the url query params and makes a reservation object, puts in session, then goes to reservation screen
func (m *Repository) BookRoom(w http.ResponseWriter, r *http.Request) {
	roomID, _ := strconv.Atoi(r.URL.Query().Get("id"))
	sd := r.URL.Query().Get("s")
	ed := r.URL.Query().Get("e")
	log.Println(r.URL.Query().Get("s"))

	layout := "2006-01-02"
	startDate, err := time.Parse(layout, sd)
	if err != nil {
		log.Println("start time err")
		m.App.Session.Put(r.Context(), "error", "Can't parse start time")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	endDate, err := time.Parse(layout, ed)
	if err != nil {
		log.Println("end time err")
		m.App.Session.Put(r.Context(), "error", "Can't parse end time")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	room, err := m.DB.GetRoomByID(roomID)
	if err != nil {
		log.Println("room id err")
		m.App.Session.Put(r.Context(), "error", "Can't get room id")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	var res models.Reservation

	res.Room.RoomName = room.RoomName
	res.StartDate = startDate
	res.EndDate = endDate
	res.RoomID = roomID

	m.App.Session.Put(r.Context(), "reservation", res)

	http.Redirect(w, r, "/make-reservation", http.StatusSeeOther)
	log.Println(roomID, startDate, endDate)
}

type jsonResponse struct {
	OK        bool   `json:"ok"`
	Message   string `json:"message"`
	RoomID    string `json:"room_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// Availability handles request for availability and sends JSON response
func (m *Repository) AvailabilityJSON(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()

	if err != nil {
		resp := jsonResponse{
			OK: false,
			Message: "parse-fail:internal server error",
		}

		out, _ := json.MarshalIndent(resp, "", "   ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
		return
	}
	sd := r.Form.Get("start")
	ed := r.Form.Get("end")

	layout := "2006-01-02"
	startDate, _ := time.Parse(layout, sd)
	endDate, _ := time.Parse(layout, ed)

	roomID, _ := strconv.Atoi(r.Form.Get("room_id"))

	available, err := m.DB.SearchAvailabilityByDatesByRoomID(startDate, endDate, roomID)
	if err != nil {
		resp := jsonResponse{
			OK:      false,
			Message: "Error connecting to my DB",
		}

		out, _ := json.MarshalIndent(resp, "", "   ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
		return
	}

	resp := jsonResponse{
		OK:      available,
		Message: "",
		StartDate: sd,
		EndDate: ed,
		RoomID: strconv.Itoa(roomID),
	}

	//no need to test error since the json is constructed by us right above
	out, _ := json.MarshalIndent(resp, "", "     ")
	log.Println(string(out))
	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

// Contact allows the user to contact us via some means
func (m *Repository) Contact(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "contact.page.tmpl", &models.TemplateData{})
}

// Reservation Summary displays a summary of a completed reservation
func (m *Repository) ReservationSummary(w http.ResponseWriter, r *http.Request) {
	reservation, ok := m.App.Session.Get(r.Context(), "reservation").(models.Reservation)
	if !ok {
		m.App.Session.Put(r.Context(), "error", "Can't get reservation from session")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	m.App.Session.Remove(r.Context(), "reservation")
	data := make(map[string]interface{})
	data["reservation"] = reservation

	sd := reservation.StartDate.Format("2006-01-02")
	ed := reservation.EndDate.Format("2006-01-02")
	stringMap := make(map[string]string)
	stringMap["start_date"] = sd
	stringMap["end_date"] = ed
	render.Template(w, r, "reservation-summary.page.tmpl", &models.TemplateData{
		Data:      data,
		StringMap: stringMap,
	})
}

//ShowLogin depicts the login screen
func (m *Repository) ShowLogin(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "login.page.tmpl", &models.TemplateData{
		Form: forms.New(nil),
	})
}

//PostShowLogin handles logging the user in
func (m *Repository) PostShowLogin(w http.ResponseWriter, r *http.Request) {
	_ = m.App.Session.RenewToken(r.Context())
	log.Println("works")

	err := r.ParseForm()
	if err != nil {
		log.Println(err)
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")
	form := forms.New(r.PostForm)
	form.Required("email", "password")
	form.IsEmail("email")
	if !form.Valid() {
		render.Template(w, r, "login.page.tmpl", &models.TemplateData{
			Form: form,
		})
		return
	}

	id, _, err := m.DB.Authenticate(email, password)
	log.Println("err: ", err)
	log.Println()
	if err != nil {
		log.Println(err)
		m.App.Session.Put(r.Context(), "error", "Invalid login credentials")
		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		return
	}

	m.App.Session.Put(r.Context(), "user_id", id)
	m.App.Session.Put(r.Context(), "flash", "Logged in successfully")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

//Logout logs a user out
func (m *Repository) Logout(w http.ResponseWriter, r *http.Request) {
	_ = m.App.Session.Destroy(r.Context())
	_ = m.App.Session.RenewToken(r.Context())

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

//Displays base dashboard for admin
func (m *Repository) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "admin-dashboard.page.tmpl", &models.TemplateData{})
}

//AdminNewReservations shows all new reservations in admin tool
func (m *Repository) AdminNewReservations(w http.ResponseWriter, r *http.Request) {
	reservations, err := m.DB.AllNewReservations()
	if err != nil {
		log.Println(err)
		m.App.Session.Put(r.Context(), "error", "Could not get new reservations")
		http.Redirect(w, r, "/user/admin/dashboard", http.StatusSeeOther)
		return
	}

	data := make(map[string]interface{})
	data["reservations"] = reservations

	render.Template(w, r, "admin-new-reservations.page.tmpl", &models.TemplateData{
		Data: data,
	})
}

//AdminAllReservations shows all reservations in admin tool
func (m *Repository) AdminAllReservations(w http.ResponseWriter, r *http.Request) {
	reservations, err := m.DB.AllReservations()
	if err != nil {
		log.Println(err)
		m.App.Session.Put(r.Context(), "error", "Could not get all reservations")
		http.Redirect(w, r, "/user/admin/dashboard", http.StatusSeeOther)
		return
	}

	data := make(map[string]interface{})
	data["reservations"] = reservations

	render.Template(w, r, "admin-all-reservations.page.tmpl", &models.TemplateData{
		Data: data,
	})
}

//Shows a single reservation in the admin tool
func (m *Repository) AdminShowReservation(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.RequestURI)
	exploded := strings.Split(r.RequestURI, "/")
	fmt.Println(exploded)
	//id of reservation
	id, err := strconv.Atoi(exploded[4])
	if err != nil {
		log.Println(err)
		m.App.Session.Put(r.Context(), "error", "Could not split string")
		http.Redirect(w, r, "/user/admin/dashboard", http.StatusSeeOther)
		return
	}

	//this is "all" or "new"
	src := exploded[3]

	stringMap := make(map[string]string)
	stringMap["src"] = src

	year := r.URL.Query().Get("y")
	month := r.URL.Query().Get("m")

	stringMap["month"] = month
	stringMap["year"] = year
	//get reservation from database
	res, err := m.DB.GetReservationByID(id)
	if err != nil {
		log.Println(err)
		m.App.Session.Put(r.Context(), "error", "Could not get id")
		http.Redirect(w, r, "/user/admin/dashboard", http.StatusSeeOther)
		return
	}

	data := make(map[string]interface{})
	data["reservation"] = res
	render.Template(w, r, "admin-reservations-show.page.tmpl", &models.TemplateData{
		StringMap: stringMap,
		Data: data,
		Form: forms.New(nil),
	})
}

func (m *Repository) AdminPostShowReservation(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't parse form")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	
	exploded := strings.Split(r.RequestURI, "/")
	//id of reservation
	id, err := strconv.Atoi(exploded[4])
	if err != nil {
		log.Println(err)
		m.App.Session.Put(r.Context(), "error", "Could not split string")
		http.Redirect(w, r, "/user/admin/dashboard", http.StatusSeeOther)
		return
	}

	//this is "all" or "new" or "cal"
	src := exploded[3]

	stringMap := make(map[string]string)
	stringMap["src"] = src
	//get reservation from database
	res, err := m.DB.GetReservationByID(id)
	if err != nil {
		log.Println(err)
		m.App.Session.Put(r.Context(), "error", "Could not update id")
		http.Redirect(w, r, "/user/admin/dashboard", http.StatusSeeOther)
		return
	}

	res.FirstName = r.Form.Get("first_name")
	res.LastName = r.Form.Get("last_name")
	res.Email = r.Form.Get("email")
	res.Phone = r.Form.Get("phone")

	err = m.DB.UpdateReservation(res)
	if err != nil {
		log.Println(err)
		m.App.Session.Put(r.Context(), "error", "Could not update reservation")
		http.Redirect(w, r, "/user/admin/dashboard", http.StatusSeeOther)
		return
	}

	month := r.Form.Get("month")
	year := r.Form.Get("year")

	m.App.Session.Put(r.Context(), "flash", "Changes saved")

	if year == "" {
		http.Redirect(w, r, fmt.Sprintf("/admin/reservations-%s", src), http.StatusSeeOther)
	}
	http.Redirect(w, r, fmt.Sprintf("/admin/reservations-calendar?y=%s&m=%s", year, month), http.StatusSeeOther)
}

//AdminReservationsCalendar displays the reservation calendar
//shows current month in current year by default
//other dates done via queryparams "y, "m"
func (m *Repository) AdminReservationsCalendar(w http.ResponseWriter, r *http.Request) {
	//assume that there is no month/year specified
	now := time.Now()

	if r.URL.Query().Get("y") != "" {
		year, _ := strconv.Atoi(r.URL.Query().Get("y"))
		month, _ := strconv.Atoi(r.URL.Query().Get("m"))
		now = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	}

	data := make(map[string]interface{})
	data["now"] = now

	next := now.AddDate(0, 1, 0)
	last := now.AddDate(0, -1, 0)

	nextMonth := next.Format("01")
	nextMonthYear := next.Format("2006")

	lastMonth := last.Format("01")
	lastMonthYear := last.Format("2006")

	stringMap := make(map[string]string)
	stringMap["next_month"] = nextMonth
	stringMap["next_month_year"] = nextMonthYear
	stringMap["last_month"] = lastMonth
	stringMap["last_month_year"] = lastMonthYear

	stringMap["this_month"] = now.Format("01")
	stringMap["this_month_year"] = now.Format("2006")

	// get the first and last days of the month
	currentYear, currentMonth, _ := now.Date()
	currentLocation := now.Location()
	firstOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)

	intMap := make(map[string]int)
	intMap["days_in_month"] = lastOfMonth.Day()

	rooms, err := m.DB.AllRooms()
	if err != nil {
		helpers.ServerError(w, err)
		return
	}

	data["rooms"] = rooms
	log.Println(rooms)

	for _, x := range rooms {
		//create maps
		reservationMap := make(map[string]int)
		blockMap := make(map[string]int)

		for d := firstOfMonth; !d.After(lastOfMonth); d = d.AddDate(0, 0, 1)  {
			reservationMap[d.Format("2006-01-2")] = 0
			blockMap[d.Format("2006-01-2")] = 0
		}
		//get all the restrictions for the current room
		restrictions, err := m.DB.GetRestrictionsForRoomByDate(x.ID, firstOfMonth, lastOfMonth)
		if err != nil {
			helpers.ServerError(w, err)
			return
		}
		for _, y := range restrictions {
			// log.Println(y)
			if y.ReservationID > 0 {
				//it's a reservation
				for d := y.StartDate; !d.After(y.EndDate); d = d.AddDate(0, 0, 1) {
					reservationMap[d.Format("2006-01-2")] = y.ReservationID
				}
			} else {
				// it's a block
				blockMap[y.StartDate.Format("2006-01-2")] = y.ID
			}
		}
		data[fmt.Sprintf("reservation_map_%d", x.ID)] = reservationMap
		data[fmt.Sprintf("block_map_%d", x.ID)] = blockMap
		// log.Println(reservationMap)
		// log.Println(blockMap["2023-09-13"])
		// log.Println(reservationMap["2023-09-13"])

		m.App.Session.Put(r.Context(), fmt.Sprintf("block_map_%d", x.ID), blockMap)
	}

	render.Template(w, r, "admin-reservations-calendar.page.tmpl", &models.TemplateData{
		StringMap: stringMap,
		Data: data,
		IntMap: intMap,
	})
}

//AdminProcessReservation marks a reservation as processed
func (m *Repository) AdminProcessReservation(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	src := chi.URLParam(r, "src")
	log.Println("id: ", id)
	_ = m.DB.UpdateProcessedForReservation(id, 1)

	year := r.URL.Query().Get("y")
	month := r.URL.Query().Get("m")

	m.App.Session.Put(r.Context(), "flash", "Reservation marked as processed")
	if year == "" {
		http.Redirect(w, r, fmt.Sprintf("/admin/reservations-%s", src), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/admin/reservations-calendar?y=%s&m=%s", year, month), http.StatusSeeOther)
	}
}

//AdminDeleteReservation deletes a reservation 
func (m *Repository) AdminDeleteReservation(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	src := chi.URLParam(r, "src")
	log.Println("id: ", id)
	_ = m.DB.DeleteReservation(id)

	year := r.URL.Query().Get("y")
	month := r.URL.Query().Get("m")

	m.App.Session.Put(r.Context(), "flash", "Reservation deleted")
	if year == "" {
		http.Redirect(w, r, fmt.Sprintf("/admin/reservations-%s", src), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/admin/reservations-calendar?y=%s&m=%s", year, month), http.StatusSeeOther)
	}
}

//AdminPostReservationsCalendar handles post of reservation calendar
func (m *Repository) AdminPostReservationsCalendar(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		helpers.ServerError(w, err)
		return
	}

	year, _ := strconv.Atoi(r.Form.Get("y"))
	month, _ := strconv.Atoi(r.Form.Get("m"))

	//process blocks
	rooms, err := m.DB.AllRooms()
	if err != nil {
		log.Println(err)
		m.App.Session.Put(r.Context(), "error", "Could not get all rooms")
		http.Redirect(w, r, "/user/admin/dashboard", http.StatusSeeOther)
		return
	}

	form := forms.New(r.PostForm)
	for _, x := range rooms {
		//Get the block map from the session.  Loop through entire map.  If we have entry that
		//does not exist in the posted data, and if the restriction id > 0, this is a block we need to remove.
		curMap := m.App.Session.Get(r.Context(), fmt.Sprintf("block_map_%d", x.ID)).(map[string]int)
		for name, value := range curMap {
			//ok will be false if the value is not in the map
			if val, ok := curMap[name]; ok {
				//only pay attention to values > 0 and that are not in the form post
				//the rest are just placeholders for days with out blocks
				if val > 0 {
					if !form.Has(fmt.Sprintf("remove_block_%d_%s", x.ID, name)) {
						//form has this unchecked (no block) but it was a block before, meaning we want to make it unblocked in db
						err := m.DB.DeleteBlockByID(value)
						if err != nil {
							log.Println(err)
						}
					}
				}
			}
		}

	}
	//now handle adding in new blocks
	for name, _ := range r.PostForm {
		//is of format "add_block_2_2023-01-1" meaning add/remove + block + id + date
		if strings.HasPrefix(name, "add_block") {
			exploded := strings.Split(name, "_")
			roomID, _ := strconv.Atoi(exploded[2])
			t, _ := time.Parse("2006-01-2", exploded[3])
			// insert a new block
			log.Println("Would insert block for room id", roomID, "for date", exploded[3])
			err := m.DB.InsertBlockForRoom(roomID, t)
			if err != nil {
				log.Println(err)
			}
		}
	}

	m.App.Session.Put(r.Context(), "flash", "Changes saved")
	http.Redirect(w, r, fmt.Sprintf("/admin/reservations-calendar?y=%d&m=%d", year, month), http.StatusSeeOther)
}