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
	params, err := getParams(req, param{"Fob_num", "int"})
	if err != nil {
		result["error"] = stringifyErr(err, "Error With Customer Login")
		return
	}

	id, name, stat, lvl, err := database.FindCustomer(params["Fob_num"].(int))
	if err != nil {
		result["error"] = stringifyErr(err, "Error With Customer Login")
		return
	}

	t, err := database.FindMostRecentSession(id)
	if err != nil {
		result["error"] = stringifyErr(err, "Error With Customer Login")
		return
	}

	log.Println(t)

	if time.Now().Unix()-t < 43200 {
		err = errors.New("Already Tanned Today")
		result["error"] = stringifyErr(err, "Error With Customer Login")
		return
	}

	result["id"] = id
	result["name"] = name
	result["status"] = stat
	result["level"] = lvl
}

func bedStatus(req *http.Request, result map[string]interface{}) {
	params, err := getParams(req, param{"Level", "int"})
	if err != nil {
		result["error"] = stringifyErr(err, "Error Checking Customer Bed Status")
		return
	}

	beds, err := database.BedsByLevel(params["Level"].(int))
	if err != nil {
		result["error"] = stringifyErr(err, "rror Checking Customer Bed Status")
		return
	}
	log.Println(beds)

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
			Bed_num:     params["bed_num"].(int),
			Customer_id: params["cust_num"].(int),
			Time_stamp:  time.Now().Unix()}

		err = database.CreateRecord(session)
		if err != nil {
			log.Println(err)
			return
		}

		log.Println("bed started, session created")
	}()
}
