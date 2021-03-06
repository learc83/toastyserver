package server

import (
	"github.com/learc83/toastyserver/database"
	"github.com/learc83/toastyserver/tmak"
	"log"
	"time"
	//"github.com/learc83/toastyserver/tmak"
	"errors"
	"net/http"
)

//http handlers--result should be returned as a hashmap with an
//"error" key, and a data key. Example: result["name"] = "jane",
//result["error"] = ""

func customerLogin(req *http.Request, result map[string]interface{}) {
	//Error Codes 1: Nothing to inform tanner except, that something went wrong
	//            2: Tanner not found in database
	//            3: Tanner not authorized
	//            4: Already tanned today.

	//Params Error
	params, err := getParams(req, param{"fob_num", "uint64"})
	if err != nil {
		result["error_code"] = 1
		result["error_message"] = stringifyErr(err, "Error With Customer Login")
		return
	}

	//DB Error
	id, name, stat, lvl, err := database.FindCustomer(params["fob_num"].(uint64))
	if err != nil {
		result["error_code"] = 1
		result["error_message"] = stringifyErr(err, "Error With Customer Login")
		return
	}

	//Customer Not Found--customer id has a default value of 0
	if id == 0 {
		err = errors.New("Keyfob not found in database")
		result["error_code"] = 2
		result["error_message"] = stringifyErr(err, "Error With Customer Login")
		return
	}

	//Customer Not Authorized--Customer status bit 0
	if !stat {
		err = errors.New("Tanner Status False (not authorized)")
		result["error_code"] = 3
		result["error_message"] = stringifyErr(err, "Error With Customer Login")
		return
	}

	//get last session information--time, and bed number default values
	//for both are 0, so if there is no last session both with be set to 0
	_, lastSessionTime, lastSessionBedId, err := database.FindMostRecentSession(id)
	if err != nil {
		result["error_code"] = 1
		result["error_message"] = stringifyErr(err, "Error With Customer Login")
		return
	}

	//These next if statements work based on the assumption that
	//lastSessionTime and lastSessionBed return 0 if no values are found

	//Cancel Session
	//if session started in last 5 minutes
	//then return error code 5 which allows customer to cancel bed
	//will not take this path if lastSesssionBedId == 0, i.e. no last session
	//TODOneed to check that it hasn't been cancelled at least 1 time. to prevent popup box 
	//from popping up in the first place
	if (lastSessionBedId != 0) && (time.Now().Unix()-lastSessionTime < 300) {
		err = errors.New("Session in Progress")
		result["error_code"] = 5
		result["error_message"] = stringifyErr(err, "Error With Customer Login")
		result["customer_id"] = id
		return
	}

	// at least 12 hours
	if time.Now().Unix()-lastSessionTime < 43200 {
		err = errors.New("Already Tanned Today")
		result["error_code"] = 4
		result["error_message"] = stringifyErr(err, "Error With Customer Login")
		return
	}

	// can't tan 2x on same date - get local time, set hour and minutes to 0 = midnight
	// local time, convert to Unix time check to see if that is smaller than last
	// session time
	t := time.Now()
	midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	//fmt.Println(midnight)
	//fmt.Println(midnight.Unix())
	//fmt.Println(lastSessionTime)
	if lastSessionTime > midnight.Unix() {
		err = errors.New("Already Tanned Today")
		result["error_code"] = 4
		result["error_message"] = stringifyErr(err, "Error With Customer Login")
		return
	}

	result["id"] = id
	result["name"] = name
	result["level"] = lvl
}

func cancelSession(req *http.Request, result map[string]interface{}) {
	params, err := getParams(req, param{"customer_id", "int"})
	if err != nil {
		result["error"] = stringifyErr(err, "Error Cancelling Session")
		return
	}

	//get last session information id and time time, default values
	//for both are 0, so if there is no last session both with be set to 0
	id := params["customer_id"].(int)
	lastSessionId, lastSessionTime, bed, err := database.FindMostRecentSession(id)
	if err != nil {
		result["error_code"] = 1
		result["error_message"] = stringifyErr(err, "Error Cancelling Session")
		return
	}

	//Can't cancel after 5 minutes
	if time.Now().Unix()-lastSessionTime > 300 {
		//return error bed already started--more than 5 minutes since session creation
		err = errors.New("More than 5 minutes has passed")
		result["error_code"] = 2
		result["error_message"] = stringifyErr(err, "Error Cancelling Session")
		return
	}

	//Can't cancel more than 1x per day. Check if last cancelled time within last 12 hours
	cancelledTime, err := database.LastCancelledSessionTime(id)
	if time.Now().Unix()-cancelledTime < 43200 {
		err = errors.New("You can't restart a session more than once")
		result["error_code"] = 2
		result["error_message"] = stringifyErr(err, "Error Cancelling Session")
		return
	}

	err = database.CancelSession(lastSessionId)
	if err != nil {
		result["error_code"] = 1
		result["error_message"] = stringifyErr(err, "Error Cancelling Session")
		return
	}

	//stop bed--send 1 minute to do that, 0 doesn't work--I think b/c the prop code on the toasty board is handling 0 oddly
	//1 works b/c tanning beds have a minimum time of 2 minutes
	go func() {
		err := tmak.StartBed(bed, 1)
		time.Sleep(0.10 * 1e9)
		err = tmak.StartBed(bed, 1)
		if err != nil {
			log.Println(err)
			return
		}

		log.Println("bed stopped by customer")
	}()

	//Empty braces == succcess or no sessions to delete
}

func bedStatus(req *http.Request, result map[string]interface{}) {
	params, err := getParams(req, param{"customer_id", "int"})
	if err != nil {
		result["error"] = stringifyErr(err, "Error Checking Customer Bed Status")
		return
	}

	beds, err := database.BedsCustomerCanAccess(params["customer_id"].(int))
	if err != nil {
		result["error"] = stringifyErr(err, "Error Checking Customer Bed Status")
		return
	}
	log.Println(beds)

	//edits bed statuses in place--true means ready for tanning
	tmak.BedStatuses(beds)

	result["beds"] = beds
}

func startBed(req *http.Request, result map[string]interface{}) {
	params, err := getParams(req,
		param{"bed_num", "int"},
		param{"time", "int"},
		param{"cust_num", "int"})

	if err != nil {
		result["error"] = stringifyErr(err, "Error Creating Session")
		return
	}

	log.Println(params)

	//starts bed and creates session in the background b/c it may take a few seconds
	//TODO try to start bed 3 or 4 times, starting bed twice to handle dirty beds
	go func() {
		err := tmak.StartBed(params["bed_num"].(int), 1)
		time.Sleep(0.10 * 1e9)
		err = tmak.StartBed(params["bed_num"].(int), params["time"].(int))
		if err != nil {
			log.Println(err)
			return
		}

		//TODO enforce foreign key constraints
		session := database.Session{
			Bed_num:      params["bed_num"].(int),
			Customer_id:  params["cust_num"].(int),
			Session_time: params["time"].(int),
			Time_stamp:   time.Now().Unix()}

		err = database.CreateRecord(session)
		if err != nil {
			log.Println(err)
			return
		}

		log.Println("bed started, session created")
	}()
}
