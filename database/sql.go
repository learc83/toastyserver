package database

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"
	"math/rand"
	//blank identifer because we only care about side effects
	_ "github.com/learc83/go-sqlite3"
)

//TODO log calling function when logging sql errors

func FindEmployee(keyNum uint64) (name string, err error) {
	stmt, err := db.Prepare(`SELECT Name
							 FROM Employee
							 WHERE Employee.Fob_num=?`)
	if err != nil {
		return
	}
	defer stmt.Close()

	err = stmt.QueryRow(keyNum).Scan(&name)
	if err == sql.ErrNoRows {
		log.Println(err)
		err = nil
	}

	return
}

func FindCustomer(keyNum uint64) (id int, name string, stat bool, lvl int, err error) {
	stmt, err := db.Prepare(`SELECT Id, Name, Status, Level
							 FROM Customer
							 WHERE Customer.Fob_num=?`)
	if err != nil {
		return
	}
	defer stmt.Close()

	err = stmt.QueryRow(keyNum).Scan(&id, &name, &stat, &lvl)
	if err == sql.ErrNoRows {
		log.Println(err)
		err = nil
	}

	return
}

func FindMostRecentSession(cust_id int) (id int, time int64, bed int, err error) {
	stmt, err := db.Prepare(`SELECT Id, Time_stamp, Bed_num
							 FROM Session
							 WHERE Session.Customer_id=?
							 AND Session.Cancelled=0
							 ORDER BY Session.Time_stamp DESC
							 LIMIT 1`)
	if err != nil {
		return
	}
	defer stmt.Close()

	err = stmt.QueryRow(cust_id).Scan(&id, &time, &bed)
	if err == sql.ErrNoRows {
		log.Println(err)
		err = nil
	}

	return
}

func LastCancelledSessionTime(cust_id int) (time int64, err error) {
	stmt, err := db.Prepare(`SELECT Time_stamp
							 FROM Session
							 WHERE Session.Customer_id=?
							 AND Session.Cancelled=1
							 ORDER BY Session.Time_stamp DESC
							 LIMIT 1`)
	if err != nil {
		return
	}
	defer stmt.Close()

	err = stmt.QueryRow(cust_id).Scan(&time)
	if err == sql.ErrNoRows {
		log.Println(err)
		err = nil
	}

	return
}

//TODO limit results to 50
//Work on error for no rows
//TODO abstract out with ListRecords just like CreateRecord
func RecentFiftyCustomers() (customers []Customer, err error) {
	rows, err := db.Query(`SELECT Id, Name, Phone, Status, Level
						   FROM Customer`)
	if err != nil {
		return
	}
	defer rows.Close()

	//equivalent to while rows.Next() == true
	for rows.Next() {
		var c Customer
		rows.Scan(&c.Id, &c.Name, &c.Phone, &c.Status, &c.Level)

		customers = append(customers, c)
	}
	rows.Close()

	return
}

//TODO limit results to 50
//TODO abstract out with ListRecords just like CreateRecord
func FindCustomersByName(name string) (customers []Customer, err error) {
	stmt, err := db.Prepare(`SELECT Id, Name, Phone, Status, Level
						   	 FROM Customer
						   	 WHERE Customer.Name LIKE ?`)
	if err != nil {
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query("%" + name + "%")
	if err != nil {
		return
	}
	defer rows.Close()

	//equivalent to while rows.Next() == true
	for rows.Next() {
		var c Customer
		err = rows.Scan(&c.Id, &c.Name, &c.Phone, &c.Status, &c.Level)
		if err != nil {
			return
		}

		customers = append(customers, c)
	}
	if rows.Err() != nil {
		err = rows.Err()
		return
	}
	rows.Close()

	return
}

//TODO Change so that levels aren't ints but strings and there
//is no level hierarchy
func BedsCustomerCanAccess(cust_id int) (beds []Bed, err error) {
	stmt, err := db.Prepare(`SELECT Level
							 FROM Customer
							 WHERE Customer.Id=?`)
	if err != nil {
		return
	}
	defer stmt.Close()

	var lvl int
	err = stmt.QueryRow(cust_id).Scan(&lvl)
	if err == sql.ErrNoRows {
		log.Println(err)
		err = nil
	}

	stmt2, err := db.Prepare(`SELECT Bed_num, Level, Max_time, Name
						     FROM Bed
						     WHERE Level <= ?`)
	if err != nil {
		return
	}
	defer stmt2.Close()

	rows, err := stmt2.Query(lvl)
	if err != nil {
		return
	}
	defer rows.Close()

	//equivalent to while rows.Next() == true
	for rows.Next() {
		var b Bed
		err = rows.Scan(&b.Bed_num, &b.Level, &b.Max_time, &b.Name)
		if err != nil {
			return
		}

		beds = append(beds, b)
	}
	if rows.Err() != nil {
		err = rows.Err()
		return
	}
	rows.Close()

	return
}

//creates a record from an initalized struct, set autoIncrement to true if the
//first field defined in the struct is an autoincrement field
//Uses reflection to set the Table name to the Type name of the struct, and to get
//the names and values of an arbitrary number of fields
//TODO check for race condition when adding new customer--make sure keyfob exists
func CreateRecord(record interface{}) (err error) {
	t := reflect.TypeOf(record)
	v := reflect.ValueOf(record)

	var fields []string
	var values []interface{}
	var qMarkSum int

	for i := 0; i < t.NumField(); i++ {
		//skip if StructTag metadata says non DB backed field
		if t.Field(i).Tag.Get("db") == "false" {
			continue
		}

		fields = append(fields, t.Field(i).Name)
		qMarkSum = qMarkSum + 1

		//set value to nill if auto increment field
		if t.Field(i).Tag.Get("db") == "autoInc" {
			values = append(values, nil)
		} else {
			values = append(values, v.Field(i).Interface())
		}
		//qMarkSum = qMarkSum + 1
	}

	fieldStr := strings.Join(fields, ", ")
	qMarks := strings.Repeat("?,", qMarkSum-1) + "?" //-1 for last ? with no comma

	sqls := fmt.Sprintf(`INSERT INTO %s(%s)
		                 values(%s)`, t.Name(), fieldStr, qMarks)

	stmt, err := db.Prepare(sqls)
	if err != nil {
		log.Println(err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(values...)
	if err != nil {
		log.Println(err)
		return
	}

	return
}

func DeleteCustomer(id int) (err error) {
	stmt, err := db.Prepare(`DELETE FROM Customer
							 WHERE Customer.Id = ?`)
	if err != nil {
		log.Println(err)
		return
	}
	defer stmt.Close()

	//WARNING will not return error if record doesn't exist
	//TODO add error for no record found
	_, err = stmt.Exec(id)
	if err != nil {
		log.Println(err)
		return
	}

	return
}

func DeleteSession(id int) (err error) {
	stmt, err := db.Prepare(`DELETE FROM Session
							 WHERE Session.Id = ?`)
	if err != nil {
		log.Println(err)
		return
	}
	defer stmt.Close()

	//WARNING will not return error if record doesn't exist
	//TODO add error for no record found
	_, err = stmt.Exec(id)
	if err != nil {
		log.Println(err)
		return
	}

	return
}

func AvailableCustomerKeyfobs() (base10 []int32, base16 []string, err error) {
	rows, err := db.Query(`SELECT Keyfob.Fob_num
						   FROM Keyfob
						   LEFT OUTER JOIN Customer
						   ON Keyfob.Fob_num = Customer.Fob_num
						   WHERE Customer.Id IS null
						   AND Keyfob.Admin = 0`)
	if err != nil {
		return
	}
	defer rows.Close()

	//equivalent to while rows.Next() == true
	for rows.Next() {
		var i int32
		err = rows.Scan(&i)
		if err != nil {
			return
		}

		base10 = append(base10, i)
		base16 = append(base16, fmt.Sprintf("%X", i))
	}
	if rows.Err() != nil {
		err = rows.Err()
		return
	}
	rows.Close()

	return
}

//Return most recent 500. 
//TODO add date filter
func RecentDoorAccesses() (doorAccesses []DoorAccess, err error) {
	rows, err := db.Query(`SELECT DoorAccess.Id, Customer_id, Name, Time_stamp, Phone 
						   FROM DoorAccess
						   INNER JOIN Customer
						   ON DoorAccess.Customer_id == Customer.Id
						   ORDER BY DoorAccess.Id DESC
						   LIMIT 500`)
	if err != nil {
		log.Println(err)
		return
	}
	defer rows.Close()

	//equivalent to while rows.Next() == true
	for rows.Next() {
		var d DoorAccess
		rows.Scan(&d.Id, &d.Customer_id, &d.Name, &d.Time_stamp, &d.Phone)

		d.Local_time = time.Unix(d.Time_stamp, 0).Local().Format("3:04pm")
		d.Month = time.Unix(d.Time_stamp, 0).Local().Format("01")
		d.Day = time.Unix(d.Time_stamp, 0).Local().Format("02")

		doorAccesses = append(doorAccesses, d)
	}
	rows.Close()

	return
}

//Return most recent 500. 
//TODO add date filter
func RecentTanSessions() (sessions []Session, err error) {
	rows, err := db.Query(`SELECT Session.Id, Customer_id, Name, Bed_num, 
						     Cancelled, Time_stamp, Session_time 
						   FROM Session
						   INNER JOIN Customer
						   ON Session.Customer_id == Customer.Id
						   ORDER BY Session.Id DESC
						   LIMIT 500`)
	if err != nil {
		log.Println(err)
		return
	}
	defer rows.Close()

	//equivalent to while rows.Next() == true
	for rows.Next() {
		var s Session
		rows.Scan(&s.Id, &s.Customer_id, &s.Name, &s.Bed_num, &s.Cancelled, 
			&s.Time_stamp, &s.Session_time)

		s.Local_time = time.Unix(s.Time_stamp, 0).Local().Format("3:04pm")
		s.Month = time.Unix(s.Time_stamp, 0).Local().Format("01")
		s.Day = time.Unix(s.Time_stamp, 0).Local().Format("02")

		sessions = append(sessions, s)
	}
	rows.Close()

	return
}

func CancelSession(id int) (err error) {
	stmt, err := db.Prepare(`UPDATE Session
							 SET Cancelled = 1
							 WHERE Session.Id = ?`)
	if err != nil {
		log.Println(err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(id)
	if err != nil {
		log.Println(err)
		return
	}

	return
}

func DeleteBed(id int) (err error) {
	stmt, err := db.Prepare(`DELETE FROM Bed
							 WHERE Bed.Bed_num = ?`)
	if err != nil {
		log.Println(err)
		return
	}
	defer stmt.Close()

	//WARNING will not return error if record doesn't exist
	//TODO add error for no record found
	_, err = stmt.Exec(id)
	if err != nil {
		log.Println(err)
		return
	}

	return
}

func UpdateBed(bed Bed) (err error) {
	stmt, err := db.Prepare(`UPDATE Bed
							 SET Level = ?,
							 Max_time = ?,
							 Name = ?
							 WHERE Bed.Bed_num = ?`)
	if err != nil {
		log.Println(err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(bed.Level, bed.Max_time, bed.Name, bed.Bed_num)
	if err != nil {
		log.Println(err)
		return
	}

	return
}

//TODO limit results to 50
//Work on error for no rows
//TODO abstract out with ListRecords just like CreateRecord
func ListBeds() (beds []Bed, err error) {
	rows, err := db.Query(`SELECT Bed_num, Level, Max_time, Name
						   FROM Bed`)
	if err != nil {
		return
	}
	defer rows.Close()

	//equivalent to while rows.Next() == true
	for rows.Next() {
		var b Bed
		rows.Scan(&b.Bed_num, &b.Level, &b.Max_time, &b.Name)

		beds = append(beds, b)
	}
	rows.Close()

	return
}

//TODO make sure transaction is functioning properly
//and this sql transaction is atomic also range checking
//also handle closing statments and transactions using defer
//Swap Bed_num with the bed who's  Bed_num is one number higher
//swapping using temporary value 999
func MoveBedDown(bed_num int) (err error) {
	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		return
	}

	stmt, err := tx.Prepare(`UPDATE Bed
							 SET Bed_num = ?
							 WHERE Bed_num = ?`)
	_, err = stmt.Exec(999, bed_num+1)
	_, err = stmt.Exec(bed_num+1, bed_num)
	_, err = stmt.Exec(bed_num, 999)

	if err != nil {
		log.Println(err)
		tx.Rollback()
		return
	}
	tx.Commit()

	return	
}

//TODO make sure transaction is functioning properly
//and this sql transaction is atomic also range checking
//also handle closing statments and transactions using defer
//Swap Bed_num with the bed who's  Bed_num is one number higher
//swapping using temporary value 999
func MoveBedUp(bed_num int) (err error) {
	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		return
	}

	stmt, err := tx.Prepare(`UPDATE Bed
							 SET Bed_num = ?
							 WHERE Bed_num = ?`)
	_, err = stmt.Exec(999, bed_num-1)
	_, err = stmt.Exec(bed_num-1, bed_num)
	_, err = stmt.Exec(bed_num, 999)

	if err != nil {
		log.Println(err)
		tx.Rollback()
		return
	}
	tx.Commit()

	return	
}

func AddFakeDoorAccesses() (err error) {
	log.Println("Adding fake door access")

	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		return
	}

	stmt, err := tx.Prepare(`INSERT INTO DoorAccess (Customer_id, Time_stamp)
							 VALUES (?, ?)`)

	for i := 0; i < 20000; i++ {
		_, err = stmt.Exec(rand.Intn(7) + 1, time.Now().Unix() - int64(500000) + int64(609 * i))
		
		if err != nil {
			log.Println(err)
			tx.Rollback()
			return
		}
	}

	tx.Commit()

	return
}

func AddFakeSessions() (err error) {
	log.Println("Adding fake sessions")

	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		return
	}

	stmt, err := tx.Prepare(`INSERT INTO Session (Bed_num, Session_time, Customer_id, Time_stamp, Cancelled)
							 VALUES (?, ?, ?, ?, ?)`)

	for i := 0; i < 20000; i++ {
		_, err = stmt.Exec(rand.Intn(5) + 1, rand.Intn(8) + 2, rand.Intn(7) + 1, time.Now().Unix() - int64(500000) + int64(609 * i), 0)
		
		if err != nil {
			log.Println(err)
			tx.Rollback()
			return
		}
	}

	tx.Commit()

	return
}