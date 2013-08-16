package server

import (
	"github.com/learc83/toastyserver/database"
	"net/http"
)

//http handlers--result should be returned as a hashmap with an
//"error" key, and a data key. Example: result["name"] = "jane",
//result["error"] = ""

//TODO replace stringifyErr with fmt.ErrorF() now that I know it exists

func employeeLogin(req *http.Request, result map[string]interface{}) {
	params, err := getParams(req, param{"Fob_num", "int"})
	if err != nil {
		result["error"] = stringifyErr(err, "Error Logging In")
		return
	}

	name, err := database.FindEmployee(params["Fob_num"].(int))
	if err != nil {
		result["error"] = stringifyErr(err, "Error Logging In")
		return
	}

	result["name"] = name
}

func customerList(req *http.Request, result map[string]interface{}) {
	customers, err := database.RecentFiftyCustomers()
	if err != nil {
		result["error"] = stringifyErr(err, "Error Displaying Customer List")
		return
	}

	result["customers"] = customers
}

func customerListByName(req *http.Request, result map[string]interface{}) {
	params, err := getParams(req, param{"name", "str"})
	if err != nil {
		result["error"] = stringifyErr(err, "Error Searching Customers")
		return
	}

	customers, err := database.FindCustomersByName(params["name"].(string))
	if err != nil {
		result["error"] = stringifyErr(err, "Error Searching Customers")
		return
	}

	result["customers"] = customers
}

//TODO enforce non-blank customer name string
func addNewCustomer(req *http.Request, result map[string]interface{}) {
	params, err := getParams(req,
		param{"name", "str"},
		param{"phone number", "str"},
		param{"level", "int"},
		param{"keyfob number", "int"})

	if err != nil {
		result["error"] = stringifyErr(err, "Error Adding New Customer")
		return
	}

	customer := database.Customer{Name: params["name"].(string),
		Phone: params["phone number"].(string), Status: true,
		Level: params["level"].(int), Fob_num: params["keyfob number"].(int)}

	err = database.CreateRecord(customer)

	if err != nil {
		result["error"] = stringifyErr(err, "Error Adding New Customer")
		return
	}
}

func availableCustomerKeyfobs(req *http.Request, result map[string]interface{}) {
	keyfobsTen, keyfobsHex, err := database.AvailableCustomerKeyfobs()
	if err != nil {
		result["error"] = stringifyErr(err, "Error Finding Available Customer Keyfobs")
		return
	}

	result["keyfobsTen"] = keyfobsTen
	result["keyfobsHex"] = keyfobsHex
}