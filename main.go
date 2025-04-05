package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var db *sql.DB
var budgetData map[string]float64
var budgetMutex sync.RWMutex

func main() {
	// ✅ Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("❌ Error loading .env file: %v", err)
	}

	// Load budget data
	loadBudgetData()

	// Initialize database connection
	initDB()

	r := gin.Default()

	// Public Routes
	r.POST("/api/signup", SignupHandler)
	r.POST("/api/login", LoginHandler)

	// Budget API Route (Optional, for verification)
	r.GET("/api/budget", func(c *gin.Context) {
		c.JSON(200, getBudgetData())
	})

	// Protected Routes (Require JWT)
	authRoutes := r.Group("/api")
	authRoutes.Use(AuthMiddleware())
	{
		authRoutes.GET("/tax-history", GetTaxHistory)
		authRoutes.POST("/tax-distribution", TaxDistributionHandler)
		authRoutes.POST("/budget-tax-distribution", BudgetBasedTaxDistributionHandler)
	}

	fmt.Println("✅ Server running on port 8080")
	r.Run(":8080")
}

// Load budget data from JSON file
func loadBudgetData() {
	data, err := ioutil.ReadFile("budget.json")
	if err != nil {
		log.Fatalf("❌ Error reading budget.json: %v", err)
	}

	err = json.Unmarshal(data, &budgetData)
	if err != nil {
		log.Fatalf("❌ Error parsing budget.json: %v", err)
	}

	log.Println("✅ Budget data loaded successfully!")
}

// Function to safely get budget data
func getBudgetData() map[string]float64 {
	budgetMutex.RLock()
	defer budgetMutex.RUnlock()
	return budgetData
}

// Database Initialization using .env values
func initDB() {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Database unreachable: %v", err)
	}

	fmt.Println("✅ Connected to database successfully!")
}

// Dummy Signup Handler (Replace with actual DB logic)
func SignupHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Signup successful"})
}

// Login Handler (Generates JWT Token)
func LoginHandler(c *gin.Context) {
	userID := "11" // Hardcoded for testing, replace with DB lookup
	token, err := GenerateJWT(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// ✅ Budget-based Tax Distribution Handler
func BudgetBasedTaxDistributionHandler(c *gin.Context) {
	var req struct {
		TotalTaxPaid float64 `json:"total_tax_paid"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	if req.TotalTaxPaid <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tax amount must be greater than zero"})
		return
	}

	// Calculate distribution based on India's budget allocation
	distributedTax := make(map[string]string)

	var totalBudget float64 = 0
	for _, amount := range budgetData {
		totalBudget += amount
	}

	for ministry, allocation := range budgetData {
		percentage := allocation / totalBudget
		rawAmount := percentage * req.TotalTaxPaid
		roundedAmount := math.Round(rawAmount*100) / 100
		formattedAmount := fmt.Sprintf("₹%.2f", roundedAmount)
		distributedTax[ministry] = formattedAmount
	}

	c.JSON(http.StatusOK, gin.H{
		"total_tax_paid":  req.TotalTaxPaid,
		"distributed_tax": distributedTax,
	})
}
