// go mod init github.com/weeraphat2000
// go build

// go get github.com/gofiber/fiber/v2 // ติดตั้ง Fiber
// go run main.go // รันโปรแกรม

// go get github.com/lib/pq // ติดตั้ง PostgreSQL
package main

import (
	"fmt"
	"log"

	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"database/sql"

	_ "github.com/lib/pq" // คนที่ใช้งาน lib นี้คือ "database/sql" ไม่ใช่เราเขียน เลยต้องใส่ _ ด้านหน้า
)

const (
	host         = "localhost"
	port         = 5432
	databaseName = "mydatabase"
	user         = "myuser"
	password     = "mypassword"
)

var db *sql.DB // ประกาศตัวแปร db ให้เป็น global จะได้เรียกใช้ง่าย

type Suppliers struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// db.exec คือ คำสั่งที่ไม่ต้องการค่าที่ return row กลับมา
// queryrow คือ คำสั่งที่ต้องการค่าที่ return กลับมา

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, databaseName)

	fmt.Println("psqlInfo:", psqlInfo)

	sdb, err := sql.Open("postgres", psqlInfo) // คือ เอาไว้เชื่อมต่อกับฐานข้อมูล
	if err != nil {                            // ถ้าเชื่อมต่อไม่ได้ จะแสดง Error
		log.Fatal(err)
	}

	db = sdb

	err = db.Ping() // คือ ทดสอบการเชื่อมต่อ
	if err != nil {
		fmt.Println("Error:", err)
		log.Fatal(err)
	}

	app := fiber.New()

	app.Get("/", middlerware, func(c *fiber.Ctx) error {
		return c.SendString("Hello, Fiber!")
	})

	supplierGroup := app.Group("/supllier")
	SuppliersModule(supplierGroup)

	// app.Post("/supllier", createSubplierController)
	// app.Get("/supllier", getSupplierController)

	app.Listen(":3000")
}

func SuppliersModule(router fiber.Router) {
	router.Post("/", createSubplierController)
	router.Get("/", middlerware, getSupplierController)
	router.Get("/:id", getSupplierByIDController)
	router.Patch("/:id", updateSupplierController)
	router.Delete("/:id", deleteSupplierController)
}

func deleteSupplierController(c *fiber.Ctx) error {
	// ดึงค่า "id" จากพารามิเตอร์ใน URL
	id := c.Params("id")

	// ประกาศตัวแปรเพื่อเก็บ ID ที่มีอยู่ในฐานข้อมูล
	var existingID int

	// เรียกใช้คำสั่ง SQL เพื่อค้นหา Supplier ตาม ID ที่ระบุ
	// QueryRow จะคืนแถวเดียวที่ตรงกับเงื่อนไข หรือ sql.ErrNoRows หากไม่พบแถวใดๆ
	err := db.QueryRow("SELECT id FROM suppliers WHERE id = $1", id).Scan(&existingID)

	// ตรวจสอบว่าเกิดข้อผิดพลาดหรือไม่หลังจากการสแกนข้อมูล
	if err == sql.ErrNoRows {
		// ถ้าไม่มีแถวใดตรงกับ ID ที่ระบุ ให้ส่ง HTTP Status 404 Not Found พร้อมข้อความ
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Not found",
		})
	} else if err != nil {
		// ถ้าเกิดข้อผิดพลาดอื่นๆ จากฐานข้อมูล ให้ส่ง HTTP Status 500 Internal Server Error พร้อมข้อความและรายละเอียดข้อผิดพลาด
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Cannot get supplier",
			"error":   err.Error(),
		})
	}

	// ถ้า Supplier มีอยู่จริง ให้ดำเนินการลบข้อมูลจากฐานข้อมูล
	_, err = db.Exec("DELETE FROM suppliers WHERE id = $1", id)
	if err != nil {
		// ถ้าเกิดข้อผิดพลาดขณะลบข้อมูล ให้ส่ง HTTP Status 500 Internal Server Error พร้อมข้อความและรายละเอียดข้อผิดพลาด
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Cannot delete user",
			"error":   err.Error(),
		})
	}

	// ถ้าลบข้อมูลสำเร็จ ให้ส่ง HTTP Status 200 OK พร้อมข้อความยืนยันการลบ
	return c.JSON(fiber.Map{
		"message": "Deleted",
	})
}

func updateSupplierController(c *fiber.Ctx) error {
	// updateSupplierController เป็นฟังก์ชันคอนโทรลเลอร์สำหรับอัปเดตข้อมูล Supplier ตาม ID ที่ระบุ

	// ดึงค่า "id" จากพารามิเตอร์ใน URL
	id := c.Params("id")

	// ประกาศตัวแปรเพื่อเก็บข้อมูล Supplier ที่ถูกส่งมาใน Body ของคำขอ (Request)
	var supplier Suppliers

	// แปลงข้อมูล JSON จาก Body ของคำขอมาเก็บในตัวแปร supplier
	if err := c.BodyParser(&supplier); err != nil {
		// หากไม่สามารถแปลง JSON ได้ ส่ง HTTP Status 400 Bad Request พร้อมข้อความ
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Cannot parse JSON",
		})
	}

	// เช็คว่ามี Supplier ที่จะอัปเดตหรือไม่ในฐานข้อมูล
	// หากไม่มี Supplier ที่มี ID ตรงกับที่ระบุ จะคืนค่า sql.ErrNoRows

	// 	ใช้ QueryRow เพื่อดึง id ของ Supplier จากฐานข้อมูลที่ตรงกับ id ที่ระบุ
	// ใช้ Scan เพื่อสแกนค่าจากผลลัพธ์มาเก็บในตัวแปร existingID
	// ถ้าไม่พบ Supplier ที่ตรงกับ id ที่ระบุ (sql.ErrNoRows) ให้ส่ง HTTP Status 404 Not Found พร้อมข้อความ "Not found"
	// หากเกิดข้อผิดพลาดอื่น ๆ จากฐานข้อมูล ให้ส่ง HTTP Status 500 Internal Server Error พร้อมข้อความ "Cannot get supplier" และรายละเอียดข้อผิดพลาด
	var existingID int

	// ใช้ QueryRow เพื่อดึง id ของ Supplier จากฐานข้อมูลที่ตรงกับ id ที่ระบุ
	err := db.QueryRow("SELECT id FROM suppliers WHERE id = $1", id).Scan(&existingID)
	if err == sql.ErrNoRows {
		// หากไม่มี Supplier ที่ตรงกับ ID ที่ระบุ ให้ส่ง HTTP Status 404 Not Found พร้อมข้อความ
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Not found",
		})
	} else if err != nil {
		// หากเกิดข้อผิดพลาดอื่นๆ จากฐานข้อมูล ให้ส่ง HTTP Status 500 Internal Server Error พร้อมข้อความและรายละเอียดข้อผิดพลาด
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Cannot get supplier",
			"error":   err.Error(),
		})
	}

	// อัปเดตข้อมูล Supplier ในฐานข้อมูล โดยใช้ชื่อใหม่และ ID ที่ระบุ
	// exec run แล้วไม่เอาค่าที่ return กลับมา
	_, err = db.Exec("UPDATE suppliers SET name = $1 WHERE id = $2", supplier.Name, id)
	if err != nil {
		// หากเกิดข้อผิดพลาดขณะอัปเดตข้อมูล ให้ส่ง HTTP Status 500 Internal Server Error พร้อมข้อความและรายละเอียดข้อผิดพลาด
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Cannot update supplier",
			"error":   err.Error(),
		})
	}

	// หากอัปเดตข้อมูลสำเร็จ ให้ส่ง HTTP Status 200 OK พร้อมข้อความยืนยันการอัปเดต
	return c.JSON(fiber.Map{
		"message": "Updated",
	})
}

func getSupplierByIDController(c *fiber.Ctx) error {
	// ดึงค่า "id" จากพารามิเตอร์ใน URL
	id := c.Params("id")

	// ตรวจสอบว่า id เป็นตัวเลขหรือไม่
	var supplierID int
	if _, err := fmt.Sscanf(id, "%d", &supplierID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid supplier ID",
		})
	}

	// สร้าง context เพื่อควบคุม timeout ของคำสั่ง SQL
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ประกาศตัวแปรสำหรับเก็บข้อมูล Supplier
	var supplier Suppliers

	// ใช้ QueryRowContext เพื่อดึงข้อมูล Supplier ตาม ID
	err := db.QueryRowContext(ctx, "SELECT id, name FROM suppliers WHERE id = $1", supplierID).
		Scan(&supplier.ID, &supplier.Name)

	// ตรวจสอบข้อผิดพลาด
	if err != nil {
		if err == sql.ErrNoRows {
			// ไม่พบข้อมูล Supplier
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "Supplier not found",
			})
		}
		// ข้อผิดพลาดอื่นๆ จากฐานข้อมูล
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Internal server error",
			"error":   err.Error(), // สำหรับ production อาจไม่ควรส่ง error ไปตรงๆ
		})
	}

	// ส่งข้อมูล Supplier กลับในรูปแบบ JSON
	return c.Status(fiber.StatusOK).JSON(supplier)
}

// ทำเป็น pointer เพื่อจัดการกับ memory ให้ดีขึ้น(ไม่ต้องใช้ pointer ก็ได้)
func createSubplier(suppliers *Suppliers) error {
	_, err := db.Exec("INSERT INTO suppliers (name) VALUES ($1)", suppliers.Name)

	if err != nil {
		return err
	}

	return nil
}

func createSubplierController(c *fiber.Ctx) error {
	var suppliers Suppliers

	err := c.BodyParser(&suppliers)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"message": "Cannot parse JSON",
		})
	}

	if suppliers.Name == "" {
		return c.Status(400).JSON(fiber.Map{
			"message": "Name is required",
		})
	}

	fmt.Println("suppliers:", suppliers)

	err = createSubplier(&suppliers)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"message": "Cannot create user",
		})
	}

	return c.JSON(fiber.Map{
		"name": suppliers.Name,
	})
}

func getSupplierController(c *fiber.Ctx) error {
	// ดึงข้อมูลจากฐานข้อมูล
	rows, err := db.Query("SELECT id, name FROM suppliers")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Cannot retrieve suppliers",
			"error":   err.Error(),
		})
	}
	// ใช้ defer เพื่อปิด rows ทันทีหลังใช้งานเสร็จ
	defer rows.Close() // defer คือ จะทำงานหลังจากทำงานทุกอย่างเสร็จ

	var suppliers []Suppliers

	// วนลูปอ่านข้อมูลแต่ละแถวจากผลลัพธ์
	for rows.Next() {
		var supplier Suppliers

		// อ่านข้อมูลจากฐานข้อมูลและเก็บในโครงสร้าง supplier
		if err := rows.Scan(&supplier.ID, &supplier.Name); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Cannot parse supplier data",
				"error":   err.Error(),
			})
		}

		// เพิ่ม supplier ที่ดึงได้เข้าไปใน slice
		suppliers = append(suppliers, supplier)
	}

	// ตรวจสอบ error ที่อาจเกิดขึ้นระหว่างการวนลูป
	if err := rows.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Error occurred while retrieving suppliers",
			"error":   err.Error(),
		})
	}

	// ส่งข้อมูล suppliers กลับในรูปแบบ JSON
	return c.Status(fiber.StatusOK).JSON(suppliers)
}

func middlerware(c *fiber.Ctx) error {
	// Middleware ที่จะทำงานก่อนทุกๆ request
	fmt.Println("Middleware called")
	c.Next()                                       // เรียกใช้ c.Next() เพื่อให้ Fiber ไปยัง handler ถัดไป
	fmt.Println("Middleware after handler called") // จะทำงานหลังจาก handler เสร็จ
	return nil                                     // คืนค่า nil เพื่อบอกว่าไม่มีข้อผิดพลาด
}
