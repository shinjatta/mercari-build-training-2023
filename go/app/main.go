package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
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
	statement.Exec()
	if err != nil {
		log.Fatal(err)
	}
}

func dbData() ([]Item, error) {
	prepareDB()
	d, err := sql.Open("sqlite3", "mercari.db")
	if err != nil {
		log.Fatal(err)
	}
	//Query to get the information from both the Category table and the Items table
	rows, err := d.Query(`SELECT Items.name, Category.name, Items.image_filename 
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
		err := rows.Scan(&item.Name, &item.Category, &item.Image)
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

func addItem(c echo.Context) error {
	//Get form data
	name := c.FormValue("name")
	category := c.FormValue("category")
	imagePath := c.FormValue("image")
	//Read the data of the image
	imageData, err := ioutil.ReadFile(imagePath)
	if err != nil {
		fmt.Println(err)
	}

	//Create new image name with sha256
	newImageName := fmt.Sprintf("%x%s", sha256.Sum256(imageData), ".jpg")

	//Message
	c.Logger().Infof("We recived a %s from category: %s", name, category)
	message := fmt.Sprintf("We recived a %s from category: %s", name, category)
	res := Response{Message: message}

	prepareDB()
	database, err := sql.Open("sqlite3", "mercari.db")
	if err != nil {
		log.Fatal(err)
	}

	//Insert the data into the database
	statement, err := database.Prepare("INSERT INTO `Items` (`id`, `name`, `category_id`, `image_filename`) VALUES (?, ?, ?, ?);")
	if err != nil {
		log.Fatal(err)
	}
	defer statement.Close()

	//Getting the id corresponding to the last item
	var itemID int
	err = database.QueryRow("SELECT id FROM Items ORDER BY id DESC LIMIT 1").Scan(&itemID)
	if err != nil {
		log.Fatal(err)
	}

	//Getting the id corresponding to the category that was given
	var categoryID int
	err = database.QueryRow("SELECT id FROM Category WHERE name = ?", category).Scan(&categoryID)
	if err != nil {
		fmt.Println("This category does not exist")
		log.Fatal(err)
	}

	//Execute the INSERT statement with the values
	_, err = statement.Exec((itemID + 1), name, categoryID, newImageName)
	if err != nil {
		log.Fatal(err)
	}

	return c.JSON(http.StatusOK, res)
}

func getImg(c echo.Context) error {
	// Create image path
	imgPath := path.Join(ImgDir, c.Param("imageFilename"))

	if !strings.HasSuffix(imgPath, ".jpg") {
		res := Response{Message: "Image path does not end with .jpg"}
		return c.JSON(http.StatusBadRequest, res)
	}
	if _, err := os.Stat(imgPath); err != nil {
		c.Logger().Debugf("Image not found: %s", imgPath)
		imgPath = path.Join(ImgDir, "default.jpg")
	}
	return c.File(imgPath)
}

func getAllItems(c echo.Context) error {
	prepareDB()
	items, err := dbData()
	if err != nil {
		fmt.Println(err)
	}
	return c.JSON(http.StatusOK, items)
}

func getItem(c echo.Context) error {
	items, err := dbData()
	if err != nil {
		fmt.Println(err)
	}

	//Get the parameter
	idParm := c.Param("id")
	id, err := strconv.Atoi(idParm)

	if err != nil {
		fmt.Println(err)
	}

	//Search for the id
	SelectedItem := Item{}
	for index, element := range items {
		if index == id {
			SelectedItem = element
			return c.JSON(http.StatusOK, SelectedItem)
		}
	}
	//TODO: cambiar statusOK
	res := Response{Message: "Not found"}
	return c.JSON(http.StatusNotFound, res)
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Logger.SetLevel(log.INFO)

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
