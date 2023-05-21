package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	_ "github.com/mattn/go-sqlite3"
)

const (
	ImgDir = "images"
)

//Item represents new object item
type Item struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Image    string `json:"image"`
}

type Items struct {
	Items []Item `json:"items"`
}

type Response struct {
	Message string `json:"message"`
}

//prepareDB creates the database in case it does not exit
func prepareDB() {
	database, err := sql.Open("sqlite3", "mercari.db")
	if err != nil {
		log.Fatal(err)
	}
	statement, err := database.Prepare(`
	CREATE TABLE IF NOT EXISTS Category (
		id INT PRIMARY KEY,
		name VARCHAR(255) NOT NULL
	);
	CREATE TABLE IF NOT EXISTS Items (
		id INT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		category_id INT NOT NULL,
		image_filename TEXT,
		FOREIGN KEY (category_id) REFERENCES Category(id)
	);
	`)
	if err != nil {
		log.Fatal(err)
	}
	statement.Exec()
	defer database.Close()
}

//dbData gets all the data for all items
func dbData() ([]Item, error) {
	prepareDB()
	d, err := sql.Open("sqlite3", "mercari.db")
	if err != nil {
		log.Fatal(err)
	}
	//Query to get the information from both the Category table and the Items table
	rows, err := d.Query(`SELECT Items.id, Items.name, Category.name, Items.image_filename 
	FROM Items
	INNER JOIN Category
	ON Category.id = Items.category_id`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	//Create an struct slice to put all the items recived
	allItems := []Item{}

	//Iterate over the results and scan the values in the item structs
	for rows.Next() {
		item := Item{}
		err := rows.Scan(&item.Id, &item.Name, &item.Category, &item.Image)
		if err != nil {
			log.Fatal(err)
		}
		allItems = append(allItems, item)
	}

	err = rows.Err()
	return allItems, err
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

//addItem adds a new item and if there is not the category given, creates a new one
func addItem(c echo.Context) error {
	// Get form data
	name := c.FormValue("name")
	category := c.FormValue("category")
	imagePath, err := c.FormFile("image")
	if err != nil {
		return fmt.Errorf("Invalid parameter: %v", err)
	}

	// Open the uploaded image file
	imageFile, err := imagePath.Open()
	if err != nil {
		return fmt.Errorf("Failed to open image: %v", err)
	}
	defer imageFile.Close()

	// Create a new image file
	imageDataPath := path.Join(ImgDir, imagePath.Filename)
	newFile, err := os.Create(imageDataPath)
	if err != nil {
		return fmt.Errorf("Failed to create image file: %v", err)
	}
	defer newFile.Close()

	// Copy the image data to the new file
	_, err = io.Copy(newFile, imageFile)
	if err != nil {
		return fmt.Errorf("Failed to copy image data: %v", err)
	}

	// Create a new image name with sha256
	newImageName := fmt.Sprintf("%x%s", sha256.Sum256([]byte(imagePath.Filename)), ".jpg")

	// Create image path
	imgPath := path.Join(ImgDir, newImageName)

	// Rename the image file with the new name
	err = os.Rename(imageDataPath, imgPath)
	if err != nil {
		return fmt.Errorf("Failed to rename image file: %v", err)
	}

	// Message
	c.Logger().Infof("We received a %s from category: %s", name, category)
	message := fmt.Sprintf("We received a %s from category: %s", name, category)
	res := Response{Message: message}

	prepareDB()
	database, err := sql.Open("sqlite3", "mercari.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	// Insert the data into the database
	statement, err := database.Prepare("INSERT INTO `Items` (`name`, `category_id`, `image_filename`) VALUES (?, ?, ?);")
	if err != nil {
		log.Fatal(err)
	}
	defer statement.Close()

	// Get the ID corresponding to the category that was given
	var categoryID int64
	err = database.QueryRow("SELECT id FROM Category WHERE name = ?", category).Scan(&categoryID)
	if err != nil {
		fmt.Println("This category does not exist")
		newCategoryID, err := addCategory(category)
		if err != nil {
			log.Fatal(err)
		}
		categoryID = newCategoryID
	}

	// Execute the INSERT statement with the values
	_, err = statement.Exec(name, categoryID, newImageName)
	if err != nil {
		log.Fatal(err)
	}

	return c.JSON(http.StatusOK, res)
}

//addCategory is called when there is no Category when creating a new item
func addCategory(category string) (int64, error) {
	prepareDB()
	database, err := sql.Open("sqlite3", "mercari.db")
	if err != nil {
		log.Fatal(err)
	}

	// Close the database connection at the end of the function
	defer database.Close()

	// Execute the INSERT statement
	result, err := database.Exec("INSERT INTO Category (name) VALUES (?)", category)
	if err != nil {
		return 0, err
	}

	// Retrieve the inserted category's ID
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func getImg(c echo.Context) error {
	// Create image path
	imgPath := path.Join(ImgDir, c.Param("imageFilename"))

	if !strings.HasSuffix(imgPath, ".jpg") {
		res := Response{Message: "Image path does not end with .jpg"}
		return c.JSON(http.StatusBadRequest, res)
	}
	if _, err := os.Stat(imgPath); err != nil {
		c.Logger().Debugf("Image not found: %s, %v", imgPath, err)
		imgPath = path.Join(ImgDir, "default.jpg")
	}
	c.Logger().Debugf(imgPath)
	return c.File(imgPath)
}

// getAllItems gets all items
func getAllItems(c echo.Context) error {
	prepareDB()
	items, err := dbData()
	if err != nil {
		res := Response{Message: "Not found"}
		return c.JSON(http.StatusNotFound, res)
	}

	// Wrap the items in the expected response structure
	response := Items{Items: items}
	return c.JSON(http.StatusOK, response)
}

//getItem gets the item with the specified id
func getItem(c echo.Context) error {
	//Get the parameter
	idParm := c.Param("id")

	//Prepare the database
	prepareDB()
	database, err := sql.Open("sqlite3", "mercari.db")
	if err != nil {
		log.Fatal(err)
	}

	defer database.Close()

	//Prepare the query
	query := `SELECT Items.id, Items.name, Category.name, Items.image_filename
          FROM Items
          INNER JOIN Category ON Items.category_id = Category.id
          WHERE Items.id = ?`

	//Getting the item
	SelectedItem := Item{}
	err = database.QueryRow(query, idParm).Scan(&SelectedItem.Id, &SelectedItem.Name, &SelectedItem.Category, &SelectedItem.Image)
	if err != nil {
		res := Response{Message: "Not found"}
		return c.JSON(http.StatusNotFound, res)
	}

	return c.JSON(http.StatusOK, SelectedItem)
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Logger.SetLevel(log.DEBUG)

	front_url := os.Getenv("FRONT_URL")
	if front_url == "" {
		front_url = "http://localhost:3000"
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{front_url},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

	// Routes
	e.GET("/", root)
	e.GET("/items", getAllItems)
	e.GET("/items/:id", getItem)
	e.POST("/items", addItem)
	e.GET("/image/:imageFilename", getImg)

	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
