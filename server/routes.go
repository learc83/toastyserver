package server

import (
//"net/http"
)

//routes to match handlers to url strings

func getRoutes() map[string]toastyHndlrFnc {
	r := make(map[string]toastyHndlrFnc)

	//admin routes
	r["/employee_login"] = employeeLogin
	r["/customer_list"] = customerList
	r["/customer_list_by_name"] = customerListByName
	r["/add_new_customer"] = addNewCustomer
	r["/available_customer_keyfobs"] = availableCustomerKeyfobs
	r["/delete_customer"] = deleteCustomer
	r["/door_report"] = doorReport
	r["/tan_report"] = tanReport
	r["/add_new_bed"] = addNewBed
	r["/delete_bed"] = deleteBed
	r["/update_bed"] = updateBed
	r["/list_beds"] = listBeds
	r["/move_bed_down"] = moveBedDown
	r["/move_bed_up"] = moveBedUp

	//customer routes
	r["/customer_login"] = customerLogin
	r["/bed_status"] = bedStatus
	r["/start_bed"] = startBed
	r["/cancel_session"] = cancelSession

	return r
}
