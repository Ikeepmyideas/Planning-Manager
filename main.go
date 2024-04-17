package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	DBUser     string `json:"dbUser"`
	DBPassword string `json:"dbPassword"`
	DBName     string `json:"dbName"`
}

var db *sql.DB
var config Config

type Room struct {
	ID       int
	Name     string
	Capacity int
}

type Reservation struct {
	ID        int
	RoomID    int
	Date      time.Time
	StartTime time.Time
	EndTime   time.Time
}

func initDB() {
	dataSourceName := fmt.Sprintf("%s:%s@/%s", config.DBUser, config.DBPassword, config.DBName)
	var err error
	db, err = sql.Open("mysql", dataSourceName)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
}

func loadConfig(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	err := loadConfig("config.json")
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	initDB()
	defer db.Close()

	fmt.Println("Welcome to the Online Reservation Service")

	for {
		displayMainMenu()
	}
}

func displayMainMenu() {
	fmt.Println("-----------------------------------------------------")
	fmt.Println("1. Lister les salles disponibles")
	fmt.Println("2. Créer une réservation")
	fmt.Println("3. Annuler une réservation")
	fmt.Println("4. Voir les réservations")
	fmt.Println("5. Quitter")
	fmt.Print("Choisissez une option : ")

	var choice int
	fmt.Scanln(&choice)

	switch choice {
	case 1:
		var inputDate string
		fmt.Print("Entrez la date (YYYY-MM-DD) : ")
		fmt.Scanln(&inputDate)

		var inputTime string
		fmt.Print("Entrez l'heure (HH:MM) : ")
		fmt.Scanln(&inputTime)

		date, err := time.Parse("2006-01-02", inputDate)
		if err != nil {
			log.Fatal("Format de date invalide. Veuillez utiliser le format YYYY-MM-DD.")
		}
		timeSlot, err := time.Parse("15:04", inputTime)
		if err != nil {
			log.Fatal("Format d'heure invalide. Veuillez utiliser le format HH:MM.")
		}

		listAvailableRoomsForTime(date, timeSlot)

	case 2:
		createReservation()
	case 3:
		cancelReservation()
	case 4:
		viewReservations()
	case 5:
		fmt.Println("Merci d'utiliser notre service !")
		os.Exit(0)
	default:
		fmt.Println("Option invalide. Veuillez réessayer.")
	}
}
func isRoomAvailable(roomID int, date, startTime, endTime time.Time) bool {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM reservations WHERE room_id = ? AND date = ? AND ((start_time <= ? AND end_time > ?) OR (start_time < ? AND end_time >= ?))", roomID, date.Format("2006-01-02"), startTime.Format("15:04:05"), startTime.Format("15:04:05"), endTime.Format("15:04:05"), endTime.Format("15:04:05")).Scan(&count)
	if err != nil {
		log.Fatal("Failed to check room availability:", err)
	}
	return count == 0
}
func getRoomIDByName(name string) (int, error) {
	var roomID int
	err := db.QueryRow("SELECT id FROM rooms WHERE name = ?", name).Scan(&roomID)
	if err != nil {
		return 0, err
	}
	return roomID, nil
}

func listAvailableRooms() {
	rows, err := db.Query("SELECT * FROM rooms")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var rooms []Room
	for rows.Next() {
		var room Room
		err := rows.Scan(&room.ID, &room.Name, &room.Capacity)
		if err != nil {
			log.Fatal(err)
		}
		rooms = append(rooms, room)
	}

	fmt.Println("Salles disponibles:")
	for _, room := range rooms {
		fmt.Printf("%d. %s (Capacité: %d)\n", room.ID, room.Name, room.Capacity)
	}

	var selectedRoomID int
	fmt.Print("Sélectionnez une salle: ")
	fmt.Scanln(&selectedRoomID)

	var inputDate string
	fmt.Print("Entrez la date (YYYY-MM-DD): ")
	fmt.Scanln(&inputDate)

	date, err := parseDate(inputDate)
	if err != nil {
		log.Fatal(err)
	}

	checkAvailabilityForRoom(selectedRoomID, date)
}

func parseDate(inputDate string) (time.Time, error) {
	date, err := time.Parse("2006-01-02", inputDate)
	if err != nil {
		return time.Time{}, err
	}

	if date.Year() < 1 || date.Year() > 9999 {
		return time.Time{}, fmt.Errorf("year is not in the range [1, 9999]: %d", date.Year())
	}

	return date, nil
}

func checkAvailabilityForRoom(roomID int, date time.Time) {
	rows, err := db.Query("SELECT start_time, end_time FROM reservations WHERE room_id = ? AND date = ?", roomID, date.Format("2006-01-02"))
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Printf("Réservations pour la Salle %d le %s:\n", roomID, date.Format("2006-01-02"))
	count := 0
	for rows.Next() {
		var startTime, endTime time.Time
		err := rows.Scan(&startTime, &endTime)
		if err != nil {
			log.Fatal(err)
		}
		count++
		fmt.Printf("%d. %s - %s\n", count, startTime.Format("2006-01-02 15:04"), endTime.Format("2006-01-02 15:04"))
	}

	if count == 0 {
		fmt.Println("Aucune réservation pour cette date.")
	}
}

func createReservation() {
	var roomName string
	var date, startTime, endTime string

	fmt.Println("Creating a reservation...")
	fmt.Print("Enter Room Name: ")
	fmt.Scanln(&roomName)
	fmt.Print("Enter Date (AAAA-MM-JJ): ")
	fmt.Scanln(&date)
	fmt.Print("Enter Start Time (HH:MM): ")
	fmt.Scanln(&startTime)
	fmt.Print("Enter End Time (HH:MM): ")
	fmt.Scanln(&endTime)

	parsedDate, err := parseDate(date)
	if err != nil {
		fmt.Println(err)
		return
	}

	parsedStartTime, err := time.Parse("15:04", startTime)
	if err != nil {
		log.Fatal("Invalid start time format. Please use HH:MM format.")
	}
	parsedEndTime, err := time.Parse("15:04", endTime)
	if err != nil {
		log.Fatal("Invalid end time format. Please use HH:MM format.")
	}

	roomID, err := getRoomIDByName(roomName)
	if err != nil {
		log.Fatal("Failed to get room ID:", err)
	}

	if !isRoomAvailable(roomID, parsedDate, parsedStartTime, parsedEndTime) {
		fmt.Println("The selected room is not available for the specified date and time.")
		return
	}

	_, err = db.Exec("INSERT INTO reservations (room_id, date, start_time, end_time) VALUES (?, ?, ?, ?)", roomID, parsedDate, parsedStartTime, parsedEndTime)
	if err != nil {
		log.Fatal("Failed to create reservation:", err)
	}

	fmt.Println("Reservation created successfully!")

	displayNavigationOptions()
}

func cancelReservation() {
}

func viewReservations() {
}
func listAvailableRoomsForTime(date time.Time, timeSlot time.Time) {
	rows, err := db.Query("SELECT id, name, capacity FROM rooms")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	dateStr := date.Format("2006-01-02")
	timeSlotStr := timeSlot.Format("15:04")

	fmt.Printf("Salles disponibles pour le %s à %s :\n", dateStr, timeSlotStr)

	var availableRooms []Room

	for rows.Next() {
		var room Room
		err := rows.Scan(&room.ID, &room.Name, &room.Capacity)
		if err != nil {
			log.Fatal(err)
		}

		if isRoomAvailableForTimeSlot(room.ID, date, timeSlot) {
			availableRooms = append(availableRooms, room)
		}
	}

	for i, room := range availableRooms {
		fmt.Printf("%d. %s (Capacité : %d)\n", i+1, room.Name, room.Capacity)
	}

	if len(availableRooms) == 0 {
		fmt.Println("Aucune salle disponible pour ce créneau horaire.")
	}
}
func isRoomAvailableForTimeSlot(roomID int, date, timeSlot time.Time) bool {
	query := `
        SELECT COUNT(*)
        FROM reservations
        WHERE room_id = ? AND date = ? AND (
            (start_time <= ? AND end_time > ?) OR 
            (start_time < ? AND end_time >= ?)
        )
    `

	var count int
	err := db.QueryRow(query, roomID, date.Format("2006-01-02"), timeSlot, timeSlot, timeSlot, timeSlot).Scan(&count)
	if err != nil {
		log.Fatal(err)
	}

	return count == 0
}
func displayNavigationOptions() {
	fmt.Println("1. Retourner au menu principal")
	fmt.Println("2. Quitter")
	fmt.Print("Choisissez une option : ")

	var choice int
	fmt.Scanln(&choice)

	switch choice {
	case 1:
		displayMainMenu()
	case 2:
		fmt.Println("Merci d'utiliser notre service !")
		os.Exit(0)
	default:
		fmt.Println("Option invalide. Veuillez réessayer.")
		displayNavigationOptions()
	}
}
