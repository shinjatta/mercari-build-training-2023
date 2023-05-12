package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
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

//DB references to the database so do not have to create it over and over again
//Copied this from the video but I think I am not using it, I am maintaining it for the time being but I will delete if not used
type DB struct {
	Database *sql.DB
}

//Item represents new object item
type Item struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Image    string `json:"image"`
}

type Items struct {
	Items []Item `json:"items"`
}

func prepareDB() {
	database, _ := sql.Open("sqlite3", "../../db/mercari.db")
	statement, _ := database.Prepare(`
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
}

func dbData() (Item, error) {
	prepareDB()
	d, _ := sql.Open("sqlite3", "../../db/mercari.db")
	rows, err := d.Query("SELECT name, category_id, image_filename FROM mercari.Items;")
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()

	for rows.Next() {
		item := Item{}
		err := rows.Scan(&item.Name, &item.Category, &item.Image)
		if err != nil {
			return Item{}, nil
		}
		return Item{}, nil
	}

	err = rows.Err()
	return Item{}, err
}

type Response struct {
	Message string `json:"message"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

func dataJson() (Items, error) {
	//Open our jsonFile
	jsonFile, err := os.Open("items.json")
	if err != nil {
		fmt.Println(err)
	}
	//Defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	//Read our opened jsonFile as a byte array.
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		fmt.Println(err)
	}

	//Inicialize our array of items
	var beforeItems Items

	//Save data into the array
	err = json.Unmarshal(byteValue, &beforeItems)
	if err != nil {
		fmt.Println(err)
	}

	return beforeItems, err
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

	//Create new item
	newItem := Item{}
	newItem.Name = name
	newItem.Category = category
	newItem.Image = newImageName

	items, err := dataJson()
	if err != nil {
		fmt.Println(err)
	}

	//Add new
	items.Items = append(items.Items, newItem)

	//Save into a JSON file
	content, err := json.Marshal(items)
	if err != nil {
		fmt.Println(err)
	}
	err = ioutil.WriteFile("items.json", content, 0644)
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
	items, err := dbData()
	if err != nil {
		fmt.Println(err)
	}
	return c.JSON(http.StatusOK, items)
}

func getItem(c echo.Context) error {
	items, err := dataJson()
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
	for index, element := range items.Items {
		if index == id {
			SelectedItem = element
			return c.JSON(http.StatusOK, SelectedItem)
		}
	}
	res := Response{Message: "Not found"}
	return c.JSON(http.StatusOK, res)
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

	// database := mercari.DB{}
	// database.InitDB()

	// Routes
	e.GET("/", root)
	e.GET("/items", getAllItems)
	e.GET("/items/:id", getItem)
	e.POST("/items", addItem)
	e.GET("/image/:imageFilename", getImg)

	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
