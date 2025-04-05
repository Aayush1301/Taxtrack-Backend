package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TaxDistribution represents the response structure
type TaxDistribution struct {
	Year         int                `json:"year"`
	Distribution map[string]float64 `json:"distribution"`
}

// Sector-wise percentage distribution
var sectorDistribution = map[string]float64{
	"Education":      0.15,
	"Healthcare":     0.20,
	"Defense":        0.30,
	"Infrastructure": 0.25,
	"Other":          0.10,
}

// TaxDistributionHandler processes tax input and saves data with user_id
func TaxDistributionHandler(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
		return
	}

	fmt.Println("✅ Extracted userID:", userID)

	var request struct {
		TotalTaxPaid float64 `json:"total_tax_paid"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// ✅ Print TotalTaxPaid value received from Postman
	fmt.Println("🟡 Received TotalTaxPaid:", request.TotalTaxPaid)

	distribution := make(map[string]float64)
	for sector, percentage := range sectorDistribution {
		distribution[sector] = request.TotalTaxPaid * percentage
	}

	fmt.Println("🟡 Inserting into database:", distribution)

	_, err := db.Exec(`
		INSERT INTO tax_records (user_id, total_tax_paid, education, healthcare, defense, infrastructure, other) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		userID, request.TotalTaxPaid,
		distribution["Education"],
		distribution["Healthcare"],
		distribution["Defense"],
		distribution["Infrastructure"],
		distribution["Other"],
	)
	if err != nil {
		fmt.Println("❌ Database Insert Error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save tax data"})
		return
	}

	fmt.Println("✅ Data successfully inserted!")

	response := TaxDistribution{
		Year:         2024,
		Distribution: distribution,
	}

	c.JSON(http.StatusOK, response)
}

// GetTaxHistory fetches only the logged-in user's tax records
func GetTaxHistory(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
		return
	}

	rows, err := db.Query(`
		SELECT total_tax_paid, education, healthcare, defense, infrastructure, other, created_at 
		FROM tax_records 
		WHERE user_id = $1 
		ORDER BY created_at DESC`, userID)
	if err != nil {
		fmt.Println("❌ Database Query Error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tax records"})
		return
	}
	defer rows.Close()

	type TaxRecord struct {
		TotalTaxPaid   float64 `json:"total_tax_paid"`
		Education      float64 `json:"education"`
		Healthcare     float64 `json:"healthcare"`
		Defense        float64 `json:"defense"`
		Infrastructure float64 `json:"infrastructure"`
		Other          float64 `json:"other"`
		CreatedAt      string  `json:"created_at"`
	}

	var taxRecords []TaxRecord

	for rows.Next() {
		var record TaxRecord
		err := rows.Scan(&record.TotalTaxPaid, &record.Education, &record.Healthcare, &record.Defense, &record.Infrastructure, &record.Other, &record.CreatedAt)
		if err != nil {
			fmt.Println("❌ Row Scan Error:", err)
			continue
		}
		taxRecords = append(taxRecords, record)
	}

	if len(taxRecords) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No tax history found", "data": []TaxRecord{}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tax_history": taxRecords})
}
