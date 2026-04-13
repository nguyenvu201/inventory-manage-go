// @title           Inventory Management System API
// @version         1.0.0
// @description     IoT Scale Inventory Management — FDA 21 CFR Part 11 compliant REST API.
// @termsOfService  https://github.com/inventory-manage

// @contact.name   Inventory Manage Team
// @contact.email  dev@inventory-manage.local

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1
package main

import (
	"fmt"

	"inventory-manage/global"
	"inventory-manage/internal/initialize"
)

func main() {
	r := initialize.Run()

	addr := fmt.Sprintf(":%d", global.Config.Server.Port)
	if err := r.Run(addr); err != nil {
		global.Logger.Fatal("HTTP server failed to start")
	}
}
