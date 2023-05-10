package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"encoding/json"
	"io/ioutil"
	"strconv"
	"crypto/sha256"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

const (
	ImgDir = "images"
)

//Create structure for items
type Item struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Image string `json:"image"`
}

type Items struct {
	Items []Item `json:"items"`
}

type Response struct {
	Message string `json:"message"`
}

func root(c echo.Context) error {
	res := Response{Message: "Hello, world!"}
	return c.JSON(http.StatusOK, res)
}

func dataJson() Items{
	//Open our jsonFile
	jsonFile, err := os.Open("items.json")
	if err != nil {
		fmt.Println(err)
	}
	//Defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	//Read our opened jsonFile as a byte array.
	byteValue, _ := ioutil.ReadAll(jsonFile)

	//Inicialize our array of items
	var beforeItems Items

	//Save data into the array
	json.Unmarshal(byteValue, &beforeItems)

	return beforeItems
}

func addItem(c echo.Context) error {
	//Get form data
	name := c.FormValue("name")
	category := c.FormValue("category")
	imagePath := c.FormValue("image")
	//Read the data of the image
	imageData, _ := ioutil.ReadFile(imagePath)

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

	items := dataJson()

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
	//sha256
	// s := "sha256 this string"
    // h := sha256.New()
    // h.Write([]byte(s))
    // bs := h.Sum(nil)
	
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
	items := dataJson()
	return c.JSON(http.StatusOK, items)
}

func getItem(c echo.Context) error {
	items := dataJson()

	//Get the parameter
	idParm := c.Param("id")
	id, err := strconv.Atoi(idParm)

	if err != nil {
		fmt.Println(err)
	}

	//Search for the id
	SelectedItem := Item{}
	for i := 0; i < len(items.Items); i++ {
		if(i==id){
			SelectedItem = items.Items[i]
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


	// Routes
	e.GET("/", root)
	e.GET("/items", getAllItems)
	e.GET("/items/:id", getItem)
	e.POST("/items", addItem)
	e.GET("/image/:imageFilename", getImg)


	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}
