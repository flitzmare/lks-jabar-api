package main

import (
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"strconv"
	"strings"
	"time"
)

func main() {
	db, err := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	fmt.Println("Connected!")

	err = db.AutoMigrate(&Menu{}, &User{})
	if err != nil {
		panic(err)
	}

	app := fiber.New()

	app.Post("/login", func(ctx *fiber.Ctx) error {
		requestBody := User{}
		err := ctx.BodyParser(&requestBody)
		if err != nil {
			return ctx.JSON(err.Error())
		}

		model := User{}
		err = db.Where("username = ? AND password = ?", requestBody.Username, requestBody.Password).First(&model).Error
		if err != nil {
			return ctx.JSON("user data not found!")
		}

		claims := jwt.MapClaims{}
		claims["authorized"] = true
		claims["username"] = requestBody.Username
		claims["user_id"] = strconv.Itoa(int(model.ID))
		//claims["exp"] = time.Now().Add(999999 * time.Hour).Unix()
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		result, err := token.SignedString([]byte("secret"))
		if err != nil {
			return ctx.JSON(err.Error())
		}
		return ctx.JSON(map[string]string{
			"type":  "Bearer",
			"token": result,
		})
	})

	app.Post("/register", func(ctx *fiber.Ctx) error {
		requestBody := User{}
		err := ctx.BodyParser(&requestBody)
		if err != nil {
			return err
		}

		model := User{}
		err = db.Where("username = ?", requestBody.Username).First(&model).Error
		if err == nil {
			return ctx.JSON("username has already taken!")
		}

		if requestBody.Username == "" {
			return ctx.JSON("field 'username' can't be empty!")
		}
		if requestBody.Password == "" {
			return ctx.JSON("field 'password' can't be empty!")
		}
		err = db.Create(&requestBody).Error
		if err != nil {
			return ctx.JSON(err.Error())
		}
		return ctx.JSON("Register success!")
	})

	//Get all menu
	app.Get("/menu", func(c *fiber.Ctx) error {
		userid, err := GetUserID(c)
		if err != nil {
			return c.JSON(err.Error())
		}
		menus := make([]Menu, 0)
		err = db.Where("user_id = ?", userid).Find(&menus).Error
		if err != nil {
			return c.JSON(err.Error())
		}
		return c.JSON(menus)
	})

	//Get menu by id
	app.Get("/menu/:id", func(c *fiber.Ctx) error {
		userid, err := GetUserID(c)
		if err != nil {
			return c.JSON(err.Error())
		}
		id, err := c.ParamsInt("id")
		if err != nil {
			return c.JSON(err.Error())
		}
		menus := make([]Menu, 0)
		err = db.Where("id = ? AND user_id = ?", id, userid).Find(&menus).Error
		if err != nil {
			return c.JSON(err.Error())
		}
		return c.JSON(menus)
	})

	//Create menu
	app.Post("/menu", func(ctx *fiber.Ctx) error {
		requestBody := Menu{}
		err := ctx.BodyParser(&requestBody)
		if err != nil {
			return ctx.JSON(err.Error())
		}

		userid, err := GetUserID(ctx)
		if err != nil {
			return ctx.JSON(err.Error())
		}

		requestBody.UserID = *userid

		if requestBody.Name == "" {
			return ctx.JSON("field 'name' can't be empty!")
		}
		if requestBody.Price == 0 {
			return ctx.JSON("field 'price' can't be zero or empty!")
		}
		err = db.Create(&requestBody).Error
		if err != nil {
			return ctx.JSON(err.Error())
		}
		return ctx.JSON(requestBody)
	})

	//Update menu
	app.Put("/menu/:id", func(ctx *fiber.Ctx) error {
		requestBody := Menu{}
		id, err := ctx.ParamsInt("id")
		if err != nil {
			return ctx.JSON(err.Error())
		}
		userid, err := GetUserID(ctx)
		if err != nil {
			return ctx.JSON(err.Error())
		}
		requestBody.ID = uint(id)
		err = ctx.BodyParser(&requestBody)
		if err != nil {
			return ctx.JSON(err.Error())
		}
		if requestBody.Name == "" {
			return ctx.JSON("field 'name' can't be empty!")
		}
		if requestBody.Price == 0 {
			return ctx.JSON("field 'price' can't be zero or empty!")
		}
		err = db.Model(&requestBody).Where("id = ? AND user_id = ? AND deleted_at is NULL", requestBody.ID, userid).Updates(&requestBody).Error
		if err != nil {
			return ctx.JSON(err.Error())
		}
		return ctx.JSON(requestBody)
	})

	//Delete menu
	app.Delete("/menu/:id", func(ctx *fiber.Ctx) error {
		id, err := ctx.ParamsInt("id")
		if err != nil {
			return err
		}
		userid, err := GetUserID(ctx)
		if err != nil {
			return ctx.JSON(err.Error())
		}
		menu := Menu{}
		err = db.Model(&menu).Where("id = ? AND user_id = ? AND deleted_at is NULL", id, userid).Delete(&menu).Error
		if err != nil {
			return ctx.JSON(err.Error())
		}
		return ctx.JSON("Delete Success!")
	})

	app.Listen(":3000")
}

func GetUserID(c *fiber.Ctx) (*uint, error) {
	if c.Get("Authorization") == "" {
		return nil, errors.New("header 'Authorization' not found!")
	}

	bearerToken := c.Get("Authorization")
	if len(strings.Split(bearerToken, " ")) == 2 {
		bearerToken = strings.Split(bearerToken, " ")[1]
	} else {
		return nil, errors.New("header 'Authorization' on wrong string format!")
	}

	token, err := jwt.Parse(bearerToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte("secret"), nil
	})

	if err != nil {
		return nil, err
		fmt.Println(err.Error())
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		result := claims["user_id"].(float64)
		var res = uint(result)
		return &res, nil
	} else {
		return nil, nil
	}
}

type User struct {
	ID       uint   `gorm:"primaryKey" json:"-"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Menu struct {
	ID          uint    `gorm:"primaryKey"`
	UserID      uint    `json:"-"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}
