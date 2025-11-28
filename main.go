package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// --- МОДЕЛИ БАЗЫ ДАННЫХ ---
type Product struct {
	ID            uint     `json:"id" gorm:"primaryKey"`
	CategoryID    int      `json:"category_id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Price         int      `json:"price"`
	ImageURLs     []string `json:"image_urls" gorm:"type:json"`
	IsRecommended bool     `json:"is_recommended"`
}
func (Product) TableName() string { return "products" }

type Category struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
	ParentID *uint  `json:"parent_id"`
}
func (Category) TableName() string { return "categories" }

type Order struct {
	ID           uint        `json:"id" gorm:"primaryKey"`
	CustomerName string      `json:"customer_name"`
	Phone        string      `json:"phone"`
	Address      string      `json:"address"`
	TotalPrice   int         `json:"total_price"`
	Status       string      `json:"status"`
	CreatedAt    time.Time   `json:"created_at"`
	Items        []OrderItem `json:"items" gorm:"foreignKey:OrderID"`
}
func (Order) TableName() string { return "orders" }

type OrderItem struct {
	ID        uint    `json:"id" gorm:"primaryKey"`
	OrderID   uint    `json:"order_id"`
	ProductID uint    `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     int     `json:"price"`
	Product   Product `json:"product" gorm:"foreignKey:ProductID"`
}
func (OrderItem) TableName() string { return "order_items" }

type OrderInput struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
	Items   []struct {
		ProductID uint `json:"product_id"`
		Quantity  int  `json:"quantity"`
		Price     int  `json:"price"`
	} `json:"items"`
}

type StatusUpdateInput struct {
	Status string `json:"status" binding:"required"`
}

// Функция для наполнения базы
func seedDatabase(db *gorm.DB) {
	log.Println("⚡️ НАЧИНАЕМ ПОЛНУЮ ПЕРЕЗАГРУЗКУ ДАННЫХ...")
	
	// 1. Создаем категории
	mainTools := Category{Name: "Инструменты"}
	db.Create(&mainTools)
	mainMat := Category{Name: "Стройматериалы"}
	db.Create(&mainMat)

	subElectric := Category{Name: "Электроинструмент", ParentID: &mainTools.ID}
	db.Create(&subElectric)
	subHand := Category{Name: "Ручной инструмент", ParentID: &mainTools.ID}
	db.Create(&subHand)
	subMix := Category{Name: "Сухие смеси", ParentID: &mainMat.ID}
	db.Create(&subMix)

	// 2. Создаем товары
	db.Create(&Product{
		CategoryID: int(subElectric.ID), Name: "Дрель Makita", Description: "Мощная ударная дрель (для теста)", Price: 45000, IsRecommended: true,
		ImageURLs: []string{"https://cdn.vseinstrumenti.ru/images/goods/instrument/dreli-shurupoverty/826998/1200x800/53248856.jpg", "https://cdn.vseinstrumenti.ru/images/goods/instrument/dreli-shurupoverty/826998/1200x800/60451475.jpg"},
	})
	db.Create(&Product{
		CategoryID: int(subHand.ID), Name: "Набор отверток", Description: "Профессиональный набор, 8 штук.", Price: 12000, IsRecommended: true,
		ImageURLs: []string{"https://cdn.vseinstrumenti.ru/images/goods/ruchnoy-instrument/otvertki/842358/1200x800/52675276.jpg"},
	})
	db.Create(&Product{
		CategoryID: int(subMix.ID), Name: "Цемент М500", Description: "Мешок 50кг.", Price: 6500, IsRecommended: true,
		ImageURLs: []string{"https://st35.stpulscen.ru/images/product/282/684/669_big.jpg"},
	})
	log.Println("✅ БАЗА ДАННЫХ УСПЕШНО ОБНОВЛЕНА!")
}

func main() {
	// DSN берется из переменной окружения Cloud Run
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Println("⚠️ DATABASE_URL не найден. Невозможно подключиться к БД.")
        // В реальном приложении здесь должен быть log.Fatal
	}
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Ошибка БД:", err)
	}

	db.AutoMigrate(&Product{}, &Category{}, &Order{}, &OrderItem{})
	
	r := gin.Default()

	// --- МАРШРУТЫ API ---
	
	// 🆘 СЕКРЕТНАЯ КНОПКА СБРОСА
	r.GET("/force-reset", func(c *gin.Context) {
		db.Migrator().DropTable(&OrderItem{}, &Order{}, &Product{}, &Category{})
		db.AutoMigrate(&Product{}, &Category{}, &Order{}, &OrderItem{})
		seedDatabase(db)
		c.JSON(200, gin.H{"message": "База данных полностью очищена и восстановлена!"})
	})
    // ... (остальные маршруты)
    
    r.GET("/products", func(c *gin.Context) {
		var products []Product
		query := db.Model(&Product{})
		if catID := c.Query("category_id"); catID != "" { query = query.Where("category_id = ?", catID) }
		if search := c.Query("q"); search != "" { query = query.Where("name ILIKE ?", "%"+search+"%") }
		if c.Query("recommended") == "true" { query = query.Where("is_recommended = ?", true) }
		query.Find(&products)
		c.JSON(200, gin.H{"data": products})
	})

	r.GET("/categories", func(c *gin.Context) {
		var categories []Category
		query := db.Model(&Category{})
		if parentID := c.Query("parent_id"); parentID != "" { query = query.Where("parent_id = ?", parentID) } else { query = query.Where("parent_id IS NULL") }
		query.Find(&categories)
		c.JSON(200, gin.H{"data": categories})
	})

	r.POST("/orders", func(c *gin.Context) {
		var input OrderInput
		if err := c.ShouldBindJSON(&input); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
		total := 0
		for _, item := range input.Items { total += item.Price * item.Quantity }
		order := Order{CustomerName: input.Name, Phone: input.Phone, Address: input.Address, TotalPrice: total, Status: "new", CreatedAt: time.Now()}
		db.Create(&order)
		for _, item := range input.Items { 
			db.Create(&OrderItem{OrderID: order.ID, ProductID: item.ProductID, Quantity: item.Quantity, Price: item.Price}) 
		}
		c.JSON(200, gin.H{"message": "OK", "order_id": order.ID})
	})

	r.GET("/admin/orders", func(c *gin.Context) {
		var orders []Order
		db.Preload("Items.Product").Order("created_at desc").Find(&orders)
		c.JSON(200, gin.H{"data": orders})
	})

	r.PUT("/admin/orders/:id", func(c *gin.Context) {
		var input StatusUpdateInput
		if err := c.ShouldBindJSON(&input); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
		db.Model(&Order{}).Where("id = ?", c.Param("id")).Update("status", input.Status)
		c.JSON(200, gin.H{"message": "Updated"})
	})

	r.DELETE("/admin/orders/:id", func(c *gin.Context) {
		db.Where("order_id = ?", c.Param("id")).Delete(&OrderItem{})
		db.Delete(&Order{}, c.Param("id"))
		c.JSON(200, gin.H{"message": "Deleted"})
	})

	r.GET("/orders/history", func(c *gin.Context) {
		db.Where("phone = ?", c.Query("phone")).Preload("Items.Product").Order("created_at desc").Find(&[]Order{})
	})


	// Порт берется из Cloud Run ($PORT)
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	r.Run("0.0.0.0:" + port)
}