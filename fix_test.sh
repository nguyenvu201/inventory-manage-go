#!/bin/bash
# Telemetry: missing "inventory-manage/internal/model" in import
sed -i '' 's/"inventory-manage\/internal\/repository\/postgres"/"inventory-manage\/internal\/model"\n\t"inventory-manage\/internal\/repository\/postgres"/g' internal/repository/postgres/telemetry_repository_integration_test.go

# Calibration: undefined model.DeviceRepository
# Let's use grep to see what it refers to.
